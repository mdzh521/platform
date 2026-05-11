package project

import "time"

type ProjectInput struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	OwnerUserID  *uint  `json:"owner_user_id"`
	OwnerTeam    string `json:"owner_team"`
	BusinessLine string `json:"business_line"`
	Status       string `json:"status"`
}

type ProjectView struct {
	ID               uint              `json:"id"`
	Code             string            `json:"code"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	OwnerUserID      *uint             `json:"owner_user_id,omitempty"`
	OwnerTeam        string            `json:"owner_team"`
	BusinessLine     string            `json:"business_line"`
	Status           string            `json:"status"`
	EnvironmentCount int64             `json:"environment_count"`
	StackCount       int64             `json:"stack_count"`
	Environments     []EnvironmentView `json:"environments,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type EnvironmentInput struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

type EnvironmentView struct {
	ID          uint      `json:"id"`
	ProjectID   uint      `json:"project_id"`
	ProjectCode string    `json:"project_code,omitempty"`
	ProjectName string    `json:"project_name,omitempty"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	StackCount  int64     `json:"stack_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StackInput struct {
	ProjectID           uint           `json:"project_id"`
	EnvironmentID       uint           `json:"environment_id"`
	Provider            string         `json:"provider"`
	AccountID           *uint          `json:"account_id"`
	Region              string         `json:"region"`
	StackType           string         `json:"stack_type"`
	StackCode           string         `json:"stack_code"`
	Name                string         `json:"name"`
	Status              string         `json:"status"`
	FoundationNetworkID *uint          `json:"foundation_network_id"`
	OwnerMode           string         `json:"owner_mode"`
	BackendType         string         `json:"backend_type"`
	BackendBucket       string         `json:"backend_bucket"`
	BackendKey          string         `json:"backend_key"`
	BackendLockTable    string         `json:"backend_lock_table"`
	StateVersion        string         `json:"state_version"`
	CurrentJobID        *uint          `json:"current_job_id"`
	LastApplyJobID      *uint          `json:"last_apply_job_id"`
	LastDestroyJobID    *uint          `json:"last_destroy_job_id"`
	DriftStatus         string         `json:"drift_status"`
	Metadata            map[string]any `json:"metadata"`
}

type StackView struct {
	ID                  uint           `json:"id"`
	ProjectID           uint           `json:"project_id"`
	ProjectCode         string         `json:"project_code,omitempty"`
	ProjectName         string         `json:"project_name,omitempty"`
	EnvironmentID       uint           `json:"environment_id"`
	EnvironmentCode     string         `json:"environment_code,omitempty"`
	EnvironmentName     string         `json:"environment_name,omitempty"`
	Provider            string         `json:"provider"`
	AccountID           *uint          `json:"account_id,omitempty"`
	Region              string         `json:"region"`
	StackType           string         `json:"stack_type"`
	StackCode           string         `json:"stack_code"`
	Name                string         `json:"name"`
	Status              string         `json:"status"`
	FoundationNetworkID *uint          `json:"foundation_network_id,omitempty"`
	OwnerMode           string         `json:"owner_mode"`
	BackendType         string         `json:"backend_type"`
	BackendBucket       string         `json:"backend_bucket"`
	BackendKey          string         `json:"backend_key"`
	BackendLockTable    string         `json:"backend_lock_table"`
	StateVersion        string         `json:"state_version"`
	CurrentJobID        *uint          `json:"current_job_id,omitempty"`
	LastApplyJobID      *uint          `json:"last_apply_job_id,omitempty"`
	LastDestroyJobID    *uint          `json:"last_destroy_job_id,omitempty"`
	DriftStatus         string         `json:"drift_status"`
	Metadata            map[string]any `json:"metadata"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}
