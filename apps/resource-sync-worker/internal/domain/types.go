package domain

import "context"

type DeploymentResource struct {
	ID           uint           `json:"id"`
	JobID        uint           `json:"job_id"`
	ResourceType string         `json:"resource_type"`
	ResourceName string         `json:"resource_name"`
	CloudID      string         `json:"cloud_id"`
	Metadata     map[string]any `json:"metadata"`
}

type Claim struct {
	ID                uint                 `json:"id"`
	Name              string               `json:"name"`
	Provider          string               `json:"provider"`
	Status            string               `json:"status"`
	Action            string               `json:"action"`
	AccountID         uint                 `json:"account_id"`
	AccountName       string               `json:"account_name"`
	BlueprintID       uint                 `json:"blueprint_id"`
	BlueprintName     string               `json:"blueprint_name"`
	BlueprintCode     string               `json:"blueprint_code"`
	BlueprintCategory string               `json:"blueprint_category"`
	NetworkPlanID     *uint                `json:"network_plan_id"`
	NetworkPlan       string               `json:"network_plan"`
	Region            string               `json:"region"`
	Input             map[string]any       `json:"input"`
	PlanSummary       map[string]any       `json:"plan_summary"`
	Output            map[string]any       `json:"output"`
	Resources         []DeploymentResource `json:"resources"`
}

type InventoryEntry struct {
	Category       string         `json:"category"`
	Region         string         `json:"region"`
	ResourceType   string         `json:"resource_type"`
	ResourceName   string         `json:"resource_name"`
	CloudID        string         `json:"cloud_id"`
	LifecycleState string         `json:"lifecycle_state"`
	Metadata       map[string]any `json:"metadata"`
}

type Result struct {
	WorkerName      string           `json:"worker_name"`
	Status          string           `json:"status"`
	ErrorMessage    string           `json:"error_message,omitempty"`
	Entries         []InventoryEntry `json:"entries,omitempty"`
	SyncedResources int              `json:"synced_resources"`
}

type ControlPlane interface {
	ClaimNextSync(context.Context, string) (*Claim, error)
	ReportSyncResult(context.Context, uint, Result) error
}
