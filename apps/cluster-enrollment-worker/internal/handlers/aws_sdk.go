package handlers

import (
	"context"
	"errors"
	"strings"

	"cluster-enrollment-worker/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/eks"
)

var awsLoadDefaultConfig = awsconfig.LoadDefaultConfig

var awsDescribeEKSCluster = func(ctx context.Context, cfg aws.Config, clusterName string) (*eks.DescribeClusterOutput, error) {
	return eks.NewFromConfig(cfg).DescribeCluster(ctx, &eks.DescribeClusterInput{Name: &clusterName})
}

func (AWSSDKAdapter) Enroll(ctx context.Context, claim *domain.Claim) (domain.Result, error) {
	clusterName := firstNonEmptyString(
		strings.TrimSpace(claim.CloudID),
		strings.TrimSpace(claim.ResourceName),
		strings.TrimSpace(stringValue(claim.Metadata["cluster_name"])),
	)
	if clusterName == "" {
		return domain.Result{}, errors.New("missing cluster name")
	}
	region := firstNonEmptyString(
		strings.TrimSpace(claim.Region),
		strings.TrimSpace(claim.Environment["AWS_REGION"]),
		strings.TrimSpace(claim.Environment["AWS_DEFAULT_REGION"]),
	)
	if region == "" {
		return domain.Result{}, errors.New("missing aws region")
	}

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	accessKey := strings.TrimSpace(claim.Environment["AWS_ACCESS_KEY_ID"])
	secretKey := strings.TrimSpace(claim.Environment["AWS_SECRET_ACCESS_KEY"])
	sessionToken := strings.TrimSpace(claim.Environment["AWS_SESSION_TOKEN"])
	if accessKey != "" && secretKey != "" {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken)))
	}
	cfg, err := awsLoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return domain.Result{}, err
	}

	response, err := awsDescribeEKSCluster(ctx, cfg, clusterName)
	if err != nil {
		return domain.Result{}, err
	}
	if response.Cluster == nil {
		return domain.Result{}, errors.New("eks describe-cluster returned empty cluster")
	}

	cluster := response.Cluster
	return domain.Result{
		Status:         deriveEnrollmentStatus(strings.TrimSpace(string(cluster.Status)), strings.TrimSpace(stringValue(cluster.Endpoint))),
		ClusterName:    strings.TrimSpace(stringValue(cluster.Name)),
		Endpoint:       strings.TrimSpace(stringValue(cluster.Endpoint)),
		Version:        strings.TrimSpace(stringValue(cluster.Version)),
		VPCID:          strings.TrimSpace(stringValue(cluster.ResourcesVpcConfig.VpcId)),
		SubnetRefs:     compactStringPointers(cluster.ResourcesVpcConfig.SubnetIds),
		AccessMode:     "direct",
		ProviderStatus: strings.TrimSpace(string(cluster.Status)),
		Metadata: map[string]any{
			"source": "aws-sdk-eks-describe-cluster",
		},
	}, nil
}

func compactStringPointers(items []string) []string {
	return compactStrings(items)
}
