package worker

import (
	"testing"

	"resource-sync-worker/internal/domain"
)

func TestBuildEntriesCarriesMetadataAndLifecycle(t *testing.T) {
	executor := NewExecutor("sync-worker", nil)
	claim := domain.Claim{
		ID:                9,
		Name:              "aws-vpc-job",
		Status:            "succeeded",
		Action:            "apply",
		AccountName:       "aws-demo",
		BlueprintCode:     "aws-vpc-base",
		BlueprintCategory: "network",
		NetworkPlan:       "plan-a",
		Region:            "ap-southeast-1",
		Resources: []domain.DeploymentResource{
			{
				ResourceType: "aws_vpc",
				ResourceName: "main",
				CloudID:      "vpc-123",
				Metadata: map[string]any{
					"cidr_block": "10.0.0.0/16",
				},
			},
		},
	}

	entries, err := executor.buildEntries(claim)
	if err != nil {
		t.Fatalf("buildEntries returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.LifecycleState != "managed" {
		t.Fatalf("unexpected lifecycle state: %s", entry.LifecycleState)
	}
	if entry.Region != "ap-southeast-1" {
		t.Fatalf("unexpected region: %s", entry.Region)
	}
	if entry.Metadata["source_job_id"] != uint(9) {
		t.Fatalf("expected source_job_id metadata, got %#v", entry.Metadata["source_job_id"])
	}
	if entry.Metadata["blueprint_code"] != "aws-vpc-base" {
		t.Fatalf("expected blueprint_code metadata, got %#v", entry.Metadata["blueprint_code"])
	}
}

func TestBuildEntriesReturnsEmptyForDestroy(t *testing.T) {
	executor := NewExecutor("sync-worker", nil)
	entries, err := executor.buildEntries(domain.Claim{
		Status: "destroyed",
		Action: "destroy",
		Resources: []domain.DeploymentResource{
			{ResourceType: "aws_vpc", ResourceName: "main"},
		},
	})
	if err != nil {
		t.Fatalf("buildEntries returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries for destroy, got %d", len(entries))
	}
}
