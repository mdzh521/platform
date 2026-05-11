package cloud

import "time"

type CloudAccount struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	Name                 string     `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Provider             string     `gorm:"size:32;not null;index" json:"provider"`
	AccessKeyEncrypted   string     `gorm:"type:text" json:"-"`
	SecretKeyEncrypted   string     `gorm:"type:longtext" json:"-"`
	RoleARN              string     `gorm:"size:255" json:"role_arn"`
	ExternalID           string     `gorm:"size:255" json:"external_id"`
	Region               string     `gorm:"size:64;not null" json:"region"`
	DefaultTagsJSON      string     `gorm:"type:longtext" json:"default_tags_json"`
	DefaultZonesJSON     string     `gorm:"type:text" json:"default_zones_json"`
	DefaultProjectID     *uint      `gorm:"index" json:"default_project_id,omitempty"`
	DefaultEnvironmentID *uint      `gorm:"index" json:"default_environment_id,omitempty"`
	Status               string     `gorm:"size:32;not null;default:active;index" json:"status"`
	LastCheckedAt        *time.Time `json:"last_checked_at"`
	LastCheckMessage     string     `gorm:"type:text" json:"last_check_message"`
	Description          string     `gorm:"size:255" json:"description"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type NetworkPlan struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Name              string    `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Provider          string    `gorm:"size:32;not null;index" json:"provider"`
	AccountID         uint      `gorm:"not null;index" json:"account_id"`
	ProjectID         *uint     `gorm:"index" json:"project_id,omitempty"`
	EnvironmentID     *uint     `gorm:"index" json:"environment_id,omitempty"`
	StackID           *uint     `gorm:"index" json:"stack_id,omitempty"`
	Region            string    `gorm:"size:64;not null" json:"region"`
	VPCCIDR           string    `gorm:"size:64;not null" json:"vpc_cidr"`
	TopologyJSON      string    `gorm:"type:longtext;not null" json:"topology_json"`
	ValidationStatus  string    `gorm:"size:32;not null;default:draft" json:"validation_status"`
	ValidationMessage string    `gorm:"type:text" json:"validation_message"`
	Status            string    `gorm:"size:32;not null;default:draft;index" json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type DeploymentBlueprint struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Code         string    `gorm:"size:128;not null;uniqueIndex" json:"code"`
	Name         string    `gorm:"size:128;not null" json:"name"`
	Provider     string    `gorm:"size:32;not null;index" json:"provider"`
	Category     string    `gorm:"size:64;not null;index" json:"category"`
	Version      string    `gorm:"size:32;not null" json:"version"`
	TemplatePath string    `gorm:"size:255;not null" json:"template_path"`
	SchemaJSON   string    `gorm:"type:longtext;not null" json:"schema_json"`
	Description  string    `gorm:"size:255" json:"description"`
	Maturity     string    `gorm:"size:32;not null;default:experimental;index" json:"maturity"`
	Capability   string    `gorm:"size:64;not null;default:plan_only;index" json:"capability"`
	Enabled      bool      `gorm:"not null;default:true;index" json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type DeploymentJob struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	Name               string     `gorm:"size:128;not null" json:"name"`
	Provider           string     `gorm:"size:32;not null;index" json:"provider"`
	AccountID          uint       `gorm:"not null;index" json:"account_id"`
	BlueprintID        uint       `gorm:"not null;index" json:"blueprint_id"`
	NetworkPlanID      *uint      `gorm:"index" json:"network_plan_id,omitempty"`
	ProjectID          *uint      `gorm:"index" json:"project_id,omitempty"`
	EnvironmentID      *uint      `gorm:"index" json:"environment_id,omitempty"`
	StackID            *uint      `gorm:"index" json:"stack_id,omitempty"`
	JobType            string     `gorm:"size:32;not null;default:provision;index" json:"job_type"`
	Status             string     `gorm:"size:32;not null;default:queued;index" json:"status"`
	Action             string     `gorm:"size:32;not null;index" json:"action"`
	RunnerName         string     `gorm:"size:128" json:"runner_name"`
	WorkspacePath      string     `gorm:"size:255" json:"workspace_path"`
	RetryOfJobID       *uint      `gorm:"index" json:"retry_of_job_id,omitempty"`
	InputJSON          string     `gorm:"type:longtext;not null" json:"input_json"`
	PlanSummaryJSON    string     `gorm:"type:longtext" json:"plan_summary_json"`
	OutputJSON         string     `gorm:"type:longtext" json:"output_json"`
	ErrorMessage       string     `gorm:"type:text" json:"error_message"`
	LogExcerpt         string     `gorm:"type:text" json:"log_excerpt"`
	ResourceSyncStatus string     `gorm:"size:32;not null;default:pending;index" json:"resource_sync_status"`
	ResourceSyncWorker string     `gorm:"size:128" json:"resource_sync_worker"`
	ResourceSyncError  string     `gorm:"type:text" json:"resource_sync_error"`
	ResourceSyncedAt   *time.Time `json:"resource_synced_at"`
	StartedAt          *time.Time `json:"started_at"`
	EndedAt            *time.Time `json:"ended_at"`
	HeartbeatAt        *time.Time `json:"heartbeat_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type DeploymentJobLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	JobID     uint      `gorm:"not null;index" json:"job_id"`
	Stage     string    `gorm:"size:32;not null;index" json:"stage"`
	Level     string    `gorm:"size:16;not null;index" json:"level"`
	Message   string    `gorm:"type:longtext;not null" json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type DeploymentResource struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	JobID        uint      `gorm:"not null;index" json:"job_id"`
	ResourceType string    `gorm:"size:128;not null" json:"resource_type"`
	ResourceName string    `gorm:"size:255;not null" json:"resource_name"`
	CloudID      string    `gorm:"size:255" json:"cloud_id"`
	MetadataJSON string    `gorm:"type:longtext" json:"metadata_json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CloudResource struct {
	ID                      uint       `gorm:"primaryKey" json:"id"`
	SyncKey                 string     `gorm:"size:255;not null;uniqueIndex" json:"sync_key"`
	Provider                string     `gorm:"size:32;not null;index" json:"provider"`
	AccountID               uint       `gorm:"not null;index" json:"account_id"`
	BlueprintID             uint       `gorm:"not null;index" json:"blueprint_id"`
	NetworkPlanID           *uint      `gorm:"index" json:"network_plan_id,omitempty"`
	ProjectID               *uint      `gorm:"index" json:"project_id,omitempty"`
	EnvironmentID           *uint      `gorm:"index" json:"environment_id,omitempty"`
	StackID                 *uint      `gorm:"index" json:"stack_id,omitempty"`
	SourceJobID             uint       `gorm:"not null;index" json:"source_job_id"`
	Category                string     `gorm:"size:64;not null;index" json:"category"`
	Region                  string     `gorm:"size:64" json:"region"`
	ResourceType            string     `gorm:"size:128;not null;index" json:"resource_type"`
	ResourceName            string     `gorm:"size:255;not null" json:"resource_name"`
	CloudID                 string     `gorm:"size:255" json:"cloud_id"`
	ResourceRole            string     `gorm:"size:64;index" json:"resource_role"`
	OwnershipMode           string     `gorm:"size:32;not null;default:terraform;index" json:"ownership_mode"`
	MachineEnrollmentStatus string     `gorm:"size:32;not null;default:not_applicable;index" json:"machine_enrollment_status"`
	MachineEnrollmentWorker string     `gorm:"size:128" json:"machine_enrollment_worker"`
	MachineEnrollmentError  string     `gorm:"type:text" json:"machine_enrollment_error"`
	MachineEnrolledAt       *time.Time `json:"machine_enrolled_at"`
	ClusterEnrollmentStatus string     `gorm:"size:32;not null;default:not_applicable;index" json:"cluster_enrollment_status"`
	ClusterEnrollmentWorker string     `gorm:"size:128" json:"cluster_enrollment_worker"`
	ClusterEnrollmentError  string     `gorm:"type:text" json:"cluster_enrollment_error"`
	ClusterEnrolledAt       *time.Time `json:"cluster_enrolled_at"`
	LifecycleState          string     `gorm:"size:32;not null;index" json:"lifecycle_state"`
	MetadataJSON            string     `gorm:"type:longtext" json:"metadata_json"`
	LastSyncedAt            *time.Time `json:"last_synced_at"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type ClusterAddonExecution struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ClusterID     uint       `gorm:"not null;index;uniqueIndex:idx_cluster_addon_execution" json:"cluster_id"`
	SourceJobID   uint       `gorm:"not null;index" json:"source_job_id"`
	ResourceID    uint       `gorm:"not null;index" json:"resource_id"`
	Provider      string     `gorm:"size:32;not null;index" json:"provider"`
	AccountID     uint       `gorm:"not null;index" json:"account_id"`
	Region        string     `gorm:"size:64;not null" json:"region"`
	AddonKey      string     `gorm:"size:128;not null;uniqueIndex:idx_cluster_addon_execution" json:"addon_key"`
	AddonType     string     `gorm:"size:64;not null;index" json:"addon_type"`
	InstallMode   string     `gorm:"size:64;not null;index" json:"install_mode"`
	ReleaseName   string     `gorm:"size:128;not null" json:"release_name"`
	Namespace     string     `gorm:"size:128;not null" json:"namespace"`
	Status        string     `gorm:"size:32;not null;default:queued;index" json:"status"`
	WorkerName    string     `gorm:"size:128" json:"worker_name"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message"`
	ContractJSON  string     `gorm:"type:longtext;not null" json:"contract_json"`
	ResultJSON    string     `gorm:"type:longtext" json:"result_json"`
	StartedAt     *time.Time `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at"`
	HeartbeatAt   *time.Time `json:"heartbeat_at"`
	LastSuccessAt *time.Time `json:"last_success_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
