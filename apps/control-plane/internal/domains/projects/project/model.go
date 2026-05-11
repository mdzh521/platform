package project

import "time"

type Project struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Code         string    `gorm:"size:64;not null;uniqueIndex" json:"code"`
	Name         string    `gorm:"size:128;not null" json:"name"`
	Description  string    `gorm:"size:255" json:"description"`
	OwnerUserID  *uint     `gorm:"index" json:"owner_user_id,omitempty"`
	OwnerTeam    string    `gorm:"size:128" json:"owner_team"`
	BusinessLine string    `gorm:"size:128" json:"business_line"`
	Status       string    `gorm:"size:32;not null;default:active;index" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Environment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"not null;index;uniqueIndex:idx_project_environment_code" json:"project_id"`
	Code      string    `gorm:"size:64;not null;uniqueIndex:idx_project_environment_code" json:"code"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Kind      string    `gorm:"size:32;not null;default:dev;index" json:"kind"`
	Status    string    `gorm:"size:32;not null;default:active;index" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Stack struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	ProjectID           uint      `gorm:"not null;index" json:"project_id"`
	EnvironmentID       uint      `gorm:"not null;index" json:"environment_id"`
	Provider            string    `gorm:"size:32;not null;index" json:"provider"`
	AccountID           *uint     `gorm:"index" json:"account_id,omitempty"`
	Region              string    `gorm:"size:64;not null;index" json:"region"`
	StackType           string    `gorm:"size:64;not null;index" json:"stack_type"`
	StackCode           string    `gorm:"size:255;not null;uniqueIndex" json:"stack_code"`
	Name                string    `gorm:"size:128;not null" json:"name"`
	Status              string    `gorm:"size:32;not null;default:draft;index" json:"status"`
	FoundationNetworkID *uint     `gorm:"index" json:"foundation_network_id,omitempty"`
	OwnerMode           string    `gorm:"size:32;not null;default:terraform;index" json:"owner_mode"`
	BackendType         string    `gorm:"size:64" json:"backend_type"`
	BackendBucket       string    `gorm:"size:255" json:"backend_bucket"`
	BackendKey          string    `gorm:"size:255" json:"backend_key"`
	BackendLockTable    string    `gorm:"size:255" json:"backend_lock_table"`
	StateVersion        string    `gorm:"size:64" json:"state_version"`
	CurrentJobID        *uint     `gorm:"index" json:"current_job_id,omitempty"`
	LastApplyJobID      *uint     `gorm:"index" json:"last_apply_job_id,omitempty"`
	LastDestroyJobID    *uint     `gorm:"index" json:"last_destroy_job_id,omitempty"`
	DriftStatus         string    `gorm:"size:32;not null;default:clean;index" json:"drift_status"`
	MetadataJSON        string    `gorm:"type:longtext" json:"metadata_json"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
