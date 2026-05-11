package worker

import (
	"context"
	"strings"

	"resource-sync-worker/internal/domain"
)

type Executor struct {
	workerName string
	backend    domain.ControlPlane
}

func NewExecutor(workerName string, backend domain.ControlPlane) *Executor {
	return &Executor{
		workerName: workerName,
		backend:    backend,
	}
}

func (e *Executor) Execute(ctx context.Context) error {
	claim, err := e.backend.ClaimNextSync(ctx, e.workerName)
	if err != nil || claim == nil {
		return err
	}

	result := domain.Result{
		WorkerName: e.workerName,
		Status:     "synced",
	}
	entries, err := e.buildEntries(*claim)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = err.Error()
	} else {
		result.Entries = entries
		result.SyncedResources = len(entries)
	}
	return e.backend.ReportSyncResult(ctx, claim.ID, result)
}

func (e *Executor) buildEntries(claim domain.Claim) ([]domain.InventoryEntry, error) {
	category := strings.TrimSpace(claim.BlueprintCategory)
	if category == "" {
		category = "network"
	}
	if claim.Action == "destroy" || claim.Status == "destroyed" {
		return []domain.InventoryEntry{}, nil
	}
	state := lifecycleStateForStatus(claim.Status)
	entries := make([]domain.InventoryEntry, 0, len(claim.Resources))
	for _, resource := range claim.Resources {
		resourceType := strings.TrimSpace(resource.ResourceType)
		resourceName := strings.TrimSpace(resource.ResourceName)
		if resourceType == "" || resourceName == "" {
			continue
		}
		metadata := cloneMap(resource.Metadata)
		if metadata == nil {
			metadata = map[string]any{}
		}
		if _, ok := metadata["source_job_id"]; !ok {
			metadata["source_job_id"] = claim.ID
		}
		if _, ok := metadata["source_job_name"]; !ok {
			metadata["source_job_name"] = claim.Name
		}
		if _, ok := metadata["account_name"]; !ok && strings.TrimSpace(claim.AccountName) != "" {
			metadata["account_name"] = claim.AccountName
		}
		if _, ok := metadata["network_plan"]; !ok && strings.TrimSpace(claim.NetworkPlan) != "" {
			metadata["network_plan"] = claim.NetworkPlan
		}
		if _, ok := metadata["blueprint_code"]; !ok && strings.TrimSpace(claim.BlueprintCode) != "" {
			metadata["blueprint_code"] = claim.BlueprintCode
		}
		entries = append(entries, domain.InventoryEntry{
			Category:       category,
			Region:         strings.TrimSpace(claim.Region),
			ResourceType:   resourceType,
			ResourceName:   resourceName,
			CloudID:        strings.TrimSpace(resource.CloudID),
			LifecycleState: state,
			Metadata:       metadata,
		})
	}
	return entries, nil
}

func lifecycleStateForStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "succeeded":
		return "managed"
	case "destroyed":
		return "destroyed"
	default:
		return "planned"
	}
}

func cloneMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
