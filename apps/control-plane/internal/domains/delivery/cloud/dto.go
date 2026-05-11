package cloud

import "time"

type CloudAccountInput struct {
	Name                 string   `json:"name"`
	Provider             string   `json:"provider"`
	AccessKey            string   `json:"access_key"`
	SecretKey            string   `json:"secret_key"`
	Region               string   `json:"region"`
	RoleARN              string   `json:"role_arn"`
	DefaultTags          []string `json:"default_tags"`
	DefaultZones         []string `json:"default_zones"`
	DefaultProjectID     *uint    `json:"default_project_id"`
	DefaultEnvironmentID *uint    `json:"default_environment_id"`
}

type CloudAccountView struct {
	ID                   uint       `json:"id"`
	Name                 string     `json:"name"`
	Provider             string     `json:"provider"`
	Region               string     `json:"region"`
	RoleARN              string     `json:"role_arn"`
	DefaultTags          []string   `json:"default_tags"`
	DefaultZones         []string   `json:"default_zones"`
	Status               string     `json:"status"`
	LastCheckedAt        *time.Time `json:"last_checked_at"`
	DefaultProjectID     *uint      `json:"default_project_id,omitempty"`
	DefaultEnvironmentID *uint      `json:"default_environment_id,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	HasAccessKey         bool       `json:"has_access_key"`
	HasSecretKey         bool       `json:"has_secret_key"`
}

type CloudInstanceTypeView struct {
	InstanceType string `json:"instance_type"`
	VCPU         int    `json:"vcpu"`
	MemoryMiB    int64  `json:"memory_mib"`
	MemoryGiB    string `json:"memory_gib"`
	Architecture string `json:"architecture"`
	Network      string `json:"network"`
}

type CloudKeyPairInput struct {
	Provider string `json:"provider"`
	Region   string `json:"region"`
	Name     string `json:"name"`
}

type CloudKeyPairView struct {
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	Region       string `json:"region"`
	Fingerprint  string `json:"fingerprint"`
	PrivateKey   string `json:"private_key"`
	KeyPairID    string `json:"key_pair_id"`
	CreatedByAPI bool   `json:"created_by_api"`
}

type NetworkPlanInput struct {
	Name              string                    `json:"name"`
	Provider          string                    `json:"provider"`
	AccountID         uint                      `json:"account_id"`
	ProjectID         *uint                     `json:"project_id"`
	EnvironmentID     *uint                     `json:"environment_id"`
	StackID           *uint                     `json:"stack_id"`
	Region            string                    `json:"region"`
	Environment       string                    `json:"environment"`
	VPCName           string                    `json:"vpc_name"`
	VPCCIDR           string                    `json:"vpc_cidr"`
	AvailabilityZones int                       `json:"availability_zones"`
	PublicSubnetCIDR  string                    `json:"public_subnet_cidr"`
	PrivateSubnetCIDR string                    `json:"private_subnet_cidr"`
	PublicSubnets     []string                  `json:"public_subnets"`
	PrivateSubnets    []string                  `json:"private_subnets"`
	SubnetGroups      []NetworkSubnetGroupInput `json:"subnet_groups"`
	BastionSubnetCIDR string                    `json:"bastion_subnet_cidr"`
	DefaultTags       []string                  `json:"default_tags"`
	NATGatewayCount   int                       `json:"nat_gateway_count"`
	CreateBastion     bool                      `json:"create_bastion_subnet"`
	SecurityBaseline  string                    `json:"security_baseline"`
}

type NetworkSubnetGroupInput struct {
	Role           string   `json:"role"`
	Tier           string   `json:"tier"`
	TrafficProfile string   `json:"traffic_profile"`
	CIDRs          []string `json:"cidrs"`
	Description    string   `json:"description"`
}

type NetworkPlanView struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	Provider      string    `json:"provider"`
	AccountID     uint      `json:"account_id"`
	AccountName   string    `json:"account_name"`
	ProjectID     *uint     `json:"project_id,omitempty"`
	EnvironmentID *uint     `json:"environment_id,omitempty"`
	StackID       *uint     `json:"stack_id,omitempty"`
	Region        string    `json:"region"`
	VPCCIDR       string    `json:"vpc_cidr"`
	Topology      any       `json:"topology"`
	Status        string    `json:"status"`
	ResourceBrief string    `json:"resource_brief"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type DeploymentJobInput struct {
	Name          string         `json:"name"`
	Provider      string         `json:"provider"`
	AccountID     uint           `json:"account_id"`
	BlueprintID   uint           `json:"blueprint_id"`
	NetworkPlanID *uint          `json:"network_plan_id"`
	ProjectID     *uint          `json:"project_id"`
	EnvironmentID *uint          `json:"environment_id"`
	StackID       *uint          `json:"stack_id"`
	SourceJobID   *uint          `json:"source_job_id,omitempty"`
	JobType       string         `json:"job_type"`
	Action        string         `json:"action"`
	Confirmed     bool           `json:"confirmed"`
	Input         map[string]any `json:"input"`
}

type DeploymentJobView struct {
	ID                 uint                     `json:"id"`
	Name               string                   `json:"name"`
	Provider           string                   `json:"provider"`
	AccountID          uint                     `json:"account_id"`
	AccountName        string                   `json:"account_name"`
	BlueprintID        uint                     `json:"blueprint_id"`
	BlueprintName      string                   `json:"blueprint_name"`
	NetworkPlanID      *uint                    `json:"network_plan_id,omitempty"`
	NetworkPlan        string                   `json:"network_plan"`
	ProjectID          *uint                    `json:"project_id,omitempty"`
	EnvironmentID      *uint                    `json:"environment_id,omitempty"`
	StackID            *uint                    `json:"stack_id,omitempty"`
	JobType            string                   `json:"job_type"`
	Status             string                   `json:"status"`
	Action             string                   `json:"action"`
	RunnerName         string                   `json:"runner_name"`
	WorkspacePath      string                   `json:"workspace_path"`
	RetryOfJobID       *uint                    `json:"retry_of_job_id,omitempty"`
	Input              any                      `json:"input"`
	PlanSummary        any                      `json:"plan_summary"`
	Output             any                      `json:"output"`
	ErrorMessage       string                   `json:"error_message"`
	LogExcerpt         string                   `json:"log_excerpt"`
	ResourceSyncStatus string                   `json:"resource_sync_status"`
	ResourceSyncWorker string                   `json:"resource_sync_worker"`
	ResourceSyncError  string                   `json:"resource_sync_error"`
	ResourceSyncedAt   *time.Time               `json:"resource_synced_at"`
	ResourceCount      int64                    `json:"resource_count"`
	Resources          []DeploymentResourceView `json:"resources,omitempty"`
	StartedAt          *time.Time               `json:"started_at"`
	EndedAt            *time.Time               `json:"ended_at"`
	HeartbeatAt        *time.Time               `json:"heartbeat_at"`
	CreatedAt          time.Time                `json:"created_at"`
	UpdatedAt          time.Time                `json:"updated_at"`
}

type DeploymentResourceView struct {
	ID           uint      `json:"id"`
	JobID        uint      `json:"job_id"`
	ResourceType string    `json:"resource_type"`
	ResourceName string    `json:"resource_name"`
	CloudID      string    `json:"cloud_id"`
	Metadata     any       `json:"metadata"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CloudResourceView struct {
	ID                      uint       `json:"id"`
	Provider                string     `json:"provider"`
	AccountID               uint       `json:"account_id"`
	AccountName             string     `json:"account_name"`
	BlueprintID             uint       `json:"blueprint_id"`
	BlueprintName           string     `json:"blueprint_name"`
	NetworkPlanID           *uint      `json:"network_plan_id,omitempty"`
	NetworkPlan             string     `json:"network_plan"`
	ProjectID               *uint      `json:"project_id,omitempty"`
	EnvironmentID           *uint      `json:"environment_id,omitempty"`
	StackID                 *uint      `json:"stack_id,omitempty"`
	SourceJobID             uint       `json:"source_job_id"`
	Category                string     `json:"category"`
	Region                  string     `json:"region"`
	ResourceType            string     `json:"resource_type"`
	ResourceName            string     `json:"resource_name"`
	CloudID                 string     `json:"cloud_id"`
	ResourceRole            string     `json:"resource_role"`
	OwnershipMode           string     `json:"ownership_mode"`
	MachineEnrollmentStatus string     `json:"machine_enrollment_status"`
	MachineEnrollmentWorker string     `json:"machine_enrollment_worker"`
	MachineEnrollmentError  string     `json:"machine_enrollment_error"`
	MachineEnrolledAt       *time.Time `json:"machine_enrolled_at"`
	ClusterEnrollmentStatus string     `json:"cluster_enrollment_status"`
	ClusterEnrollmentWorker string     `json:"cluster_enrollment_worker"`
	ClusterEnrollmentError  string     `json:"cluster_enrollment_error"`
	ClusterEnrolledAt       *time.Time `json:"cluster_enrolled_at"`
	LifecycleState          string     `json:"lifecycle_state"`
	Metadata                any        `json:"metadata"`
	LastSyncedAt            *time.Time `json:"last_synced_at"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type RetryResourceEnrollmentInput struct {
	Target string `json:"target"`
}

type Summary struct {
	Accounts   int64 `json:"accounts"`
	Networks   int64 `json:"networks"`
	Blueprints int64 `json:"blueprints"`
	Jobs       int64 `json:"jobs"`
	QueuedJobs int64 `json:"queued_jobs"`
	Resources  int64 `json:"resources"`
}

type JobClaimRequest struct {
	RunnerName string `json:"runner_name"`
}

type JobClaimView struct {
	ID            uint              `json:"id"`
	Name          string            `json:"name"`
	Provider      string            `json:"provider"`
	Action        string            `json:"action"`
	AccountID     uint              `json:"account_id"`
	BlueprintID   uint              `json:"blueprint_id"`
	NetworkPlanID *uint             `json:"network_plan_id,omitempty"`
	BlueprintCode string            `json:"blueprint_code"`
	TemplatePath  string            `json:"template_path"`
	Input         any               `json:"input"`
	Environment   map[string]string `json:"environment"`
	WorkspacePath string            `json:"workspace_path"`
	ClaimedAt     *time.Time        `json:"claimed_at"`
}

type BlueprintView struct {
	ID              uint      `json:"id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	Provider        string    `json:"provider"`
	Category        string    `json:"category"`
	Version         string    `json:"version"`
	TemplatePath    string    `json:"template_path"`
	SchemaJSON      string    `json:"schema_json"`
	Description     string    `json:"description"`
	Maturity        string    `json:"maturity"`
	Capability      string    `json:"capability"`
	SupportsApply   bool      `json:"supports_apply"`
	SupportsDestroy bool      `json:"supports_destroy"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type DeploymentJobLogView struct {
	ID        uint      `json:"id"`
	JobID     uint      `json:"job_id"`
	Stage     string    `json:"stage"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type JobHeartbeatInput struct {
	RunnerName string `json:"runner_name"`
	Status     string `json:"status"`
}

type JobLogInput struct {
	RunnerName string `json:"runner_name"`
	Stage      string `json:"stage"`
	Level      string `json:"level"`
	Message    string `json:"message"`
}

type JobResultInput struct {
	RunnerName   string         `json:"runner_name"`
	Status       string         `json:"status"`
	LogExcerpt   string         `json:"log_excerpt"`
	Output       map[string]any `json:"output"`
	PlanSummary  map[string]any `json:"plan_summary"`
	ErrorMessage string         `json:"error_message"`
}

type RetryJobInput struct {
	Name      string         `json:"name"`
	Action    string         `json:"action"`
	Confirmed bool           `json:"confirmed"`
	Input     map[string]any `json:"input"`
}

type ResourceSyncClaimRequest struct {
	WorkerName string `json:"worker_name"`
}

type ResourceSyncClaimView struct {
	ID                uint                     `json:"id"`
	Name              string                   `json:"name"`
	Provider          string                   `json:"provider"`
	Status            string                   `json:"status"`
	Action            string                   `json:"action"`
	AccountID         uint                     `json:"account_id"`
	AccountName       string                   `json:"account_name"`
	BlueprintID       uint                     `json:"blueprint_id"`
	BlueprintName     string                   `json:"blueprint_name"`
	BlueprintCode     string                   `json:"blueprint_code"`
	BlueprintCategory string                   `json:"blueprint_category"`
	NetworkPlanID     *uint                    `json:"network_plan_id,omitempty"`
	NetworkPlan       string                   `json:"network_plan"`
	Region            string                   `json:"region"`
	Input             any                      `json:"input"`
	PlanSummary       any                      `json:"plan_summary"`
	Output            any                      `json:"output"`
	Resources         []DeploymentResourceView `json:"resources"`
}

type ResourceInventoryEntryInput struct {
	Category       string         `json:"category"`
	Region         string         `json:"region"`
	ResourceType   string         `json:"resource_type"`
	ResourceName   string         `json:"resource_name"`
	CloudID        string         `json:"cloud_id"`
	LifecycleState string         `json:"lifecycle_state"`
	Metadata       map[string]any `json:"metadata"`
}

type ResourceSyncResultInput struct {
	WorkerName      string                        `json:"worker_name"`
	Status          string                        `json:"status"`
	ErrorMessage    string                        `json:"error_message"`
	Entries         []ResourceInventoryEntryInput `json:"entries"`
	SyncedResources int                           `json:"synced_resources"`
}

type MachineEnrollmentClaimRequest struct {
	WorkerName string `json:"worker_name"`
}

type MachineEnrollmentClaimView struct {
	ResourceID          uint           `json:"resource_id"`
	WorkerName          string         `json:"worker_name,omitempty"`
	Provider            string         `json:"provider"`
	AccountID           uint           `json:"account_id"`
	Region              string         `json:"region"`
	ProjectID           *uint          `json:"project_id,omitempty"`
	EnvironmentID       *uint          `json:"environment_id,omitempty"`
	StackID             *uint          `json:"stack_id,omitempty"`
	FoundationNetworkID *uint          `json:"foundation_network_id,omitempty"`
	ResourceType        string         `json:"resource_type"`
	ResourceName        string         `json:"resource_name"`
	CloudID             string         `json:"cloud_id"`
	Metadata            map[string]any `json:"metadata"`
}

type MachineEnrollmentResultInput struct {
	WorkerName      string `json:"worker_name"`
	Status          string `json:"status"`
	ErrorMessage    string `json:"error_message"`
	EnrolledAssetID *uint  `json:"enrolled_asset_id,omitempty"`
}

type ClusterEnrollmentClaimRequest struct {
	WorkerName string `json:"worker_name"`
}

type ClusterEnrollmentClaimView struct {
	ResourceID          uint              `json:"resource_id"`
	WorkerName          string            `json:"worker_name,omitempty"`
	Provider            string            `json:"provider"`
	AccountID           uint              `json:"account_id"`
	Region              string            `json:"region"`
	ProjectID           *uint             `json:"project_id,omitempty"`
	EnvironmentID       *uint             `json:"environment_id,omitempty"`
	StackID             *uint             `json:"stack_id,omitempty"`
	FoundationNetworkID *uint             `json:"foundation_network_id,omitempty"`
	ResourceType        string            `json:"resource_type"`
	ResourceName        string            `json:"resource_name"`
	CloudID             string            `json:"cloud_id"`
	Metadata            map[string]any    `json:"metadata"`
	Environment         map[string]string `json:"environment"`
}

type ClusterEnrollmentResultInput struct {
	WorkerName     string         `json:"worker_name"`
	Status         string         `json:"status"`
	ErrorMessage   string         `json:"error_message"`
	ClusterName    string         `json:"cluster_name"`
	Endpoint       string         `json:"endpoint"`
	Version        string         `json:"version"`
	VPCID          string         `json:"vpc_id"`
	SubnetRefs     []string       `json:"subnet_refs"`
	AccessMode     string         `json:"access_mode"`
	ProviderStatus string         `json:"provider_status"`
	Metadata       map[string]any `json:"metadata"`
}

type ClusterAddonExecutionView struct {
	ID            uint       `json:"id"`
	ClusterID     uint       `json:"cluster_id"`
	ClusterName   string     `json:"cluster_name"`
	SourceJobID   uint       `json:"source_job_id"`
	ResourceID    uint       `json:"resource_id"`
	Provider      string     `json:"provider"`
	AccountID     uint       `json:"account_id"`
	Region        string     `json:"region"`
	AddonKey      string     `json:"addon_key"`
	AddonType     string     `json:"addon_type"`
	InstallMode   string     `json:"install_mode"`
	ReleaseName   string     `json:"release_name"`
	Namespace     string     `json:"namespace"`
	Status        string     `json:"status"`
	WorkerName    string     `json:"worker_name"`
	ErrorMessage  string     `json:"error_message"`
	Contract      any        `json:"contract"`
	Result        any        `json:"result"`
	StartedAt     *time.Time `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at"`
	HeartbeatAt   *time.Time `json:"heartbeat_at"`
	LastSuccessAt *time.Time `json:"last_success_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type ClusterAddonClaimRequest struct {
	WorkerName string `json:"worker_name"`
}

type ClusterAddonClaimView struct {
	ExecutionID uint              `json:"execution_id"`
	ClusterID   uint              `json:"cluster_id"`
	ClusterName string            `json:"cluster_name"`
	Provider    string            `json:"provider"`
	AccountID   uint              `json:"account_id"`
	Region      string            `json:"region"`
	AddonKey    string            `json:"addon_key"`
	AddonType   string            `json:"addon_type"`
	InstallMode string            `json:"install_mode"`
	ReleaseName string            `json:"release_name"`
	Namespace   string            `json:"namespace"`
	Contract    map[string]any    `json:"contract"`
	Environment map[string]string `json:"environment"`
}

type ClusterAddonHeartbeatInput struct {
	WorkerName string `json:"worker_name"`
	Status     string `json:"status"`
}

type ClusterAddonResultInput struct {
	WorkerName   string         `json:"worker_name"`
	Status       string         `json:"status"`
	ErrorMessage string         `json:"error_message"`
	Result       map[string]any `json:"result"`
}

type RetryClusterAddonInput struct {
	AddonKey string `json:"addon_key"`
}
