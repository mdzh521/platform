package handlers

import (
	"context"
	"testing"

	"cluster-enrollment-worker/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	types "github.com/aws/aws-sdk-go-v2/service/eks/types"
)

func TestAWSSDKAdapterMapsDescribeClusterResponse(t *testing.T) {
	previousLoadConfig := awsLoadDefaultConfig
	previousDescribe := awsDescribeEKSCluster
	t.Cleanup(func() {
		awsLoadDefaultConfig = previousLoadConfig
		awsDescribeEKSCluster = previousDescribe
	})

	var loaded bool
	var described bool
	awsLoadDefaultConfig = func(_ context.Context, _ ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
		loaded = true
		return aws.Config{Region: "ap-southeast-1"}, nil
	}
	awsDescribeEKSCluster = func(_ context.Context, cfg aws.Config, clusterName string) (*eks.DescribeClusterOutput, error) {
		described = true
		if cfg.Region != "ap-southeast-1" {
			t.Fatalf("expected region to be propagated, got %q", cfg.Region)
		}
		if clusterName != "demo-eks" {
			t.Fatalf("expected cluster name demo-eks, got %q", clusterName)
		}
		return &eks.DescribeClusterOutput{
			Cluster: &types.Cluster{
				Name:    aws.String("demo-eks"),
				Status:  types.ClusterStatusActive,
				Endpoint: aws.String("https://demo.eks.local"),
				Version: aws.String("1.31"),
				ResourcesVpcConfig: &types.VpcConfigResponse{
					VpcId:     aws.String("vpc-123"),
					SubnetIds: []string{"subnet-a", "subnet-b"},
				},
			},
		}, nil
	}

	result, err := (AWSSDKAdapter{}).Enroll(context.Background(), &domain.Claim{
		Provider:     "aws",
		Region:       "ap-southeast-1",
		ResourceName: "demo-eks",
	})
	if err != nil {
		t.Fatalf("expected adapter to succeed, got error: %v", err)
	}
	if !loaded || !described {
		t.Fatalf("expected load+describe to be called, loaded=%v described=%v", loaded, described)
	}
	if result.Status != "enrolled" {
		t.Fatalf("expected enrolled status, got %q", result.Status)
	}
	if result.Endpoint != "https://demo.eks.local" {
		t.Fatalf("unexpected endpoint: %q", result.Endpoint)
	}
	if result.Version != "1.31" {
		t.Fatalf("unexpected version: %q", result.Version)
	}
	if result.VPCID != "vpc-123" {
		t.Fatalf("unexpected vpc id: %q", result.VPCID)
	}
	if len(result.SubnetRefs) != 2 || result.SubnetRefs[0] != "subnet-a" || result.SubnetRefs[1] != "subnet-b" {
		t.Fatalf("unexpected subnet refs: %#v", result.SubnetRefs)
	}
	if result.Metadata["source"] != "aws-sdk-eks-describe-cluster" {
		t.Fatalf("unexpected metadata source: %#v", result.Metadata["source"])
	}
}

func TestAWSSDKAdapterRequiresClusterNameAndRegion(t *testing.T) {
	adapter := AWSSDKAdapter{}
	if _, err := adapter.Enroll(context.Background(), &domain.Claim{Provider: "aws", Region: "ap-southeast-1"}); err == nil {
		t.Fatal("expected missing cluster name error")
	}
	if _, err := adapter.Enroll(context.Background(), &domain.Claim{Provider: "aws", ResourceName: "demo-eks"}); err == nil {
		t.Fatal("expected missing region error")
	}
}
