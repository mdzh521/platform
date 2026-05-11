package handlers

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"cluster-enrollment-worker/internal/domain"
)

type awsDescribeClusterResponse struct {
	Cluster struct {
		Name      string `json:"name"`
		Version   string `json:"version"`
		Status    string `json:"status"`
		Endpoint  string `json:"endpoint"`
		Resources struct {
			VpcConfig struct {
				VpcID     string   `json:"vpcId"`
				SubnetIDs []string `json:"subnetIds"`
			} `json:"resourcesVpcConfig"`
		} `json:"resourcesVpcConfig"`
	} `json:"cluster"`
}

func enrollAWSCluster(ctx context.Context, claim *domain.Claim) domain.Result {
	clusterName := firstNonEmptyString(
		strings.TrimSpace(claim.CloudID),
		strings.TrimSpace(claim.ResourceName),
		strings.TrimSpace(stringValue(claim.Metadata["cluster_name"])),
	)
	if clusterName == "" {
		return domain.Result{Status: "error", ErrorMessage: "missing cluster name"}
	}
	cmd := exec.CommandContext(ctx, "aws", "eks", "describe-cluster", "--name", clusterName, "--region", claim.Region, "--output", "json")
	cmd.Env = append(os.Environ(), flattenEnvMap(claim.Environment)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return domain.Result{Status: "error", ErrorMessage: strings.TrimSpace(string(output))}
	}
	var response awsDescribeClusterResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return domain.Result{Status: "error", ErrorMessage: err.Error()}
	}
	return domain.Result{
		Status:         deriveEnrollmentStatus(response.Cluster.Status, response.Cluster.Endpoint),
		ClusterName:    response.Cluster.Name,
		Endpoint:       response.Cluster.Endpoint,
		Version:        response.Cluster.Version,
		VPCID:          response.Cluster.Resources.VpcConfig.VpcID,
		SubnetRefs:     response.Cluster.Resources.VpcConfig.SubnetIDs,
		AccessMode:     "direct",
		ProviderStatus: response.Cluster.Status,
		Metadata: map[string]any{
			"source": "aws-eks-describe-cluster",
		},
	}
}
