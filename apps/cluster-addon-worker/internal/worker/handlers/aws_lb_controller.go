package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cluster-addon-worker/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	smithymiddleware "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

var addonAWSLoadDefaultConfig = awsconfig.LoadDefaultConfig

var addonAWSDescribeEKSCluster = func(ctx context.Context, cfg aws.Config, clusterName string) (*eks.DescribeClusterOutput, error) {
	return eks.NewFromConfig(cfg).DescribeCluster(ctx, &eks.DescribeClusterInput{Name: &clusterName})
}

func InstallAWSLoadBalancerController(ctx context.Context, claim *domain.Claim) (map[string]any, error) {
	clusterName := firstNonEmpty(
		claim.ClusterName,
		stringValue(claim.Contract["cluster_name"]),
	)
	if clusterName == "" {
		return nil, errors.New("missing cluster name")
	}
	region := firstNonEmpty(
		claim.Region,
		claim.Environment["AWS_REGION"],
		claim.Environment["AWS_DEFAULT_REGION"],
		stringValue(claim.Contract["region"]),
	)
	if region == "" {
		return nil, errors.New("missing aws region")
	}
	cfg, err := loadAWSConfig(ctx, claim.Environment, region)
	if err != nil {
		return nil, err
	}
	describeOutput, err := addonAWSDescribeEKSCluster(ctx, cfg, clusterName)
	if err != nil {
		return nil, err
	}
	if describeOutput.Cluster == nil {
		return nil, errors.New("eks describe-cluster returned empty cluster")
	}
	endpoint := stringValue(describeOutput.Cluster.Endpoint)
	caData := stringValue(describeOutput.Cluster.CertificateAuthority.Data)
	if endpoint == "" || caData == "" {
		return nil, errors.New("eks cluster endpoint or certificate authority is empty")
	}
	token, err := generateEKSToken(ctx, cfg, clusterName)
	if err != nil {
		return nil, err
	}

	workdir, err := os.MkdirTemp("", "cluster-addon-worker-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workdir)

	kubeconfigPath, err := writeKubeconfig(workdir, clusterName, endpoint, caData, token)
	if err != nil {
		return nil, err
	}
	valuesPath, err := writeAWSLoadBalancerValues(workdir, claim, clusterName, region)
	if err != nil {
		return nil, err
	}

	helmHome := filepath.Join(workdir, "helm")
	for _, dir := range []string{
		filepath.Join(helmHome, "cache"),
		filepath.Join(helmHome, "config"),
		filepath.Join(helmHome, "data"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	commandEnv := append(os.Environ(),
		"KUBECONFIG="+kubeconfigPath,
		"HELM_CACHE_HOME="+filepath.Join(helmHome, "cache"),
		"HELM_CONFIG_HOME="+filepath.Join(helmHome, "config"),
		"HELM_DATA_HOME="+filepath.Join(helmHome, "data"),
	)
	if _, err := runCommand(ctx, commandEnv, "helm", "repo", "add", "eks", firstNonEmpty(stringValue(claim.Contract["chart_repository"]), "https://aws.github.io/eks-charts")); err != nil {
		return nil, err
	}
	if _, err := runCommand(ctx, commandEnv, "helm", "repo", "update", "eks"); err != nil {
		return nil, err
	}
	chartName := firstNonEmpty(stringValue(claim.Contract["chart_name"]), "aws-load-balancer-controller")
	chartVersion := firstNonEmpty(stringValue(claim.Contract["chart_version"]), "1.14.0")
	if _, err := runCommand(ctx, commandEnv,
		"helm", "upgrade", "--install", claim.ReleaseName, "eks/"+chartName,
		"--namespace", claim.Namespace,
		"--create-namespace",
		"--version", chartVersion,
		"-f", valuesPath,
		"--wait",
		"--timeout", "10m0s",
	); err != nil {
		return nil, err
	}
	if _, err := runCommand(ctx, commandEnv,
		"kubectl", "rollout", "status", "deployment/"+claim.ReleaseName,
		"-n", claim.Namespace,
		"--timeout=10m",
	); err != nil {
		return nil, err
	}

	deploymentSummary, err := collectJSONSummary(ctx, commandEnv, "kubectl", "get", "deployment", claim.ReleaseName, "-n", claim.Namespace, "-o", "json")
	if err != nil {
		return nil, err
	}
	serviceAccountSummary, err := collectJSONSummary(ctx, commandEnv, "kubectl", "get", "serviceaccount", firstNonEmpty(stringValue(claim.Contract["service_account_name"]), "aws-load-balancer-controller"), "-n", claim.Namespace, "-o", "json")
	if err != nil {
		return nil, err
	}
	podSummary, err := collectJSONSummary(ctx, commandEnv, "kubectl", "get", "pods", "-n", claim.Namespace, "-l", "app.kubernetes.io/name=aws-load-balancer-controller", "-o", "json")
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"adapter":      "platform-managed-helm",
		"cluster_name": clusterName,
		"release": map[string]any{
			"name":      claim.ReleaseName,
			"namespace": claim.Namespace,
			"chart":     chartName,
			"version":   chartVersion,
		},
		"deployment":      summarizeDeployment(deploymentSummary),
		"service_account": summarizeServiceAccount(serviceAccountSummary),
		"pods":            summarizePods(podSummary),
		"endpoint":        endpoint,
	}, nil
}

func loadAWSConfig(ctx context.Context, env map[string]string, region string) (aws.Config, error) {
	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	accessKey := strings.TrimSpace(env["AWS_ACCESS_KEY_ID"])
	secretKey := strings.TrimSpace(env["AWS_SECRET_ACCESS_KEY"])
	sessionToken := strings.TrimSpace(env["AWS_SESSION_TOKEN"])
	if accessKey != "" && secretKey != "" {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken)))
	}
	return addonAWSLoadDefaultConfig(ctx, loadOptions...)
}

func generateEKSToken(ctx context.Context, cfg aws.Config, clusterName string) (string, error) {
	client := sts.NewFromConfig(cfg, func(options *sts.Options) {
		options.APIOptions = append(options.APIOptions, addClusterIDHeader(clusterName))
	})
	presignClient := sts.NewPresignClient(client)
	presigned, err := presignClient.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return "k8s-aws-v1." + base64.RawURLEncoding.EncodeToString([]byte(presigned.URL)), nil
}

func addClusterIDHeader(clusterName string) func(*smithymiddleware.Stack) error {
	return func(stack *smithymiddleware.Stack) error {
		return stack.Build.Add(smithymiddleware.BuildMiddlewareFunc("eksClusterIDHeader", func(
			ctx context.Context, in smithymiddleware.BuildInput, next smithymiddleware.BuildHandler,
		) (out smithymiddleware.BuildOutput, metadata smithymiddleware.Metadata, err error) {
			req, ok := in.Request.(*smithyhttp.Request)
			if ok {
				req.Header.Set("x-k8s-aws-id", clusterName)
			}
			return next.HandleBuild(ctx, in)
		}), smithymiddleware.After)
	}
}

func writeKubeconfig(dir, clusterName, endpoint, certificateAuthorityData, token string) (string, error) {
	path := filepath.Join(dir, "kubeconfig.yaml")
	content := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: %s
  cluster:
    server: %s
    certificate-authority-data: %s
users:
- name: %s
  user:
    token: %s
contexts:
- name: %s
  context:
    cluster: %s
    user: %s
current-context: %s
`, clusterName, endpoint, certificateAuthorityData, clusterName, token, clusterName, clusterName, clusterName, clusterName)
	return path, os.WriteFile(path, []byte(content), 0o600)
}

func writeAWSLoadBalancerValues(dir string, claim *domain.Claim, clusterName, region string) (string, error) {
	path := filepath.Join(dir, "aws-load-balancer-controller-values.yaml")
	serviceAccountName := firstNonEmpty(stringValue(claim.Contract["service_account_name"]), "aws-load-balancer-controller")
	iamRoleARN := strings.TrimSpace(stringValue(claim.Contract["iam_role_arn"]))
	if iamRoleARN == "" {
		return "", errors.New("missing aws load balancer controller iam role arn")
	}
	vpcID := firstNonEmpty(stringValue(claim.Contract["vpc_id"]), stringValue(claim.Contract["VpcId"]))
	content := fmt.Sprintf(`clusterName: %s
region: %s
vpcId: %s
serviceAccount:
  create: true
  name: %s
  annotations:
    eks.amazonaws.com/role-arn: %s
`, clusterName, region, vpcID, serviceAccountName, iamRoleARN)
	return path, os.WriteFile(path, []byte(content), 0o600)
}

func runCommand(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), message)
	}
	return output, nil
}

func collectJSONSummary(ctx context.Context, env []string, name string, args ...string) (map[string]any, error) {
	output, err := runCommand(ctx, env, name, args...)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func summarizeDeployment(payload map[string]any) map[string]any {
	spec, _ := payload["spec"].(map[string]any)
	status, _ := payload["status"].(map[string]any)
	return map[string]any{
		"name":             nestedString(payload, "metadata", "name"),
		"namespace":        nestedString(payload, "metadata", "namespace"),
		"desired_replicas": intValue(spec["replicas"]),
		"ready_replicas":   intValue(status["readyReplicas"]),
		"updated_replicas": intValue(status["updatedReplicas"]),
		"available":        intValue(status["availableReplicas"]) > 0,
	}
}

func summarizeServiceAccount(payload map[string]any) map[string]any {
	annotations, _ := nestedMap(payload, "metadata", "annotations")
	return map[string]any{
		"name":        nestedString(payload, "metadata", "name"),
		"namespace":   nestedString(payload, "metadata", "namespace"),
		"annotations": annotations,
	}
}

func summarizePods(payload map[string]any) []map[string]any {
	items, _ := payload["items"].([]any)
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, map[string]any{
			"name":      nestedString(row, "metadata", "name"),
			"namespace": nestedString(row, "metadata", "namespace"),
			"phase":     nestedString(row, "status", "phase"),
			"pod_ip":    nestedString(row, "status", "podIP"),
		})
	}
	return result
}

func nestedString(payload map[string]any, path ...string) string {
	current := any(payload)
	for _, key := range path {
		row, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = row[key]
	}
	return stringValue(current)
}

func nestedMap(payload map[string]any, path ...string) (map[string]any, bool) {
	current := any(payload)
	for _, key := range path {
		row, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current = row[key]
	}
	row, ok := current.(map[string]any)
	return row, ok
}

func intValue(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	default:
		return 0
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case *string:
		if typed == nil {
			return ""
		}
		return strings.TrimSpace(*typed)
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
