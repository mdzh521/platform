package k8s

import "time"

type ClusterListItem struct {
	ID                  uint       `json:"id"`
	Name                string     `json:"name"`
	Code                string     `json:"code"`
	Environment         string     `json:"environment"`
	ProjectID           *uint      `json:"project_id,omitempty"`
	EnvironmentID       *uint      `json:"environment_id,omitempty"`
	StackID             *uint      `json:"stack_id,omitempty"`
	FoundationNetworkID *uint      `json:"foundation_network_id,omitempty"`
	Provider            string     `json:"provider"`
	APIEndpoint         string     `json:"api_endpoint"`
	AuthType            string     `json:"auth_type"`
	SourceResourceID    string     `json:"source_resource_id"`
	AccessMode          string     `json:"access_mode"`
	Version             string     `json:"version"`
	VPCID               string     `json:"vpc_id"`
	SubnetRefs          []string   `json:"subnet_refs"`
	Description         string     `json:"description"`
	Status              string     `json:"status"`
	HasCredential       bool       `json:"has_credential"`
	NamespaceCount      int64      `json:"namespace_count"`
	LastCheckedAt       *time.Time `json:"last_checked_at"`
	LastSyncedAt        *time.Time `json:"last_synced_at"`
	ServerVersion       string     `json:"server_version"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type ClusterInput struct {
	Name                string   `json:"name" binding:"required"`
	Code                string   `json:"code" binding:"required"`
	Environment         string   `json:"environment" binding:"required"`
	ProjectID           *uint    `json:"project_id"`
	EnvironmentID       *uint    `json:"environment_id"`
	StackID             *uint    `json:"stack_id"`
	FoundationNetworkID *uint    `json:"foundation_network_id"`
	Provider            string   `json:"provider" binding:"required"`
	APIEndpoint         string   `json:"api_endpoint" binding:"required"`
	AuthType            string   `json:"auth_type" binding:"required"`
	Credential          string   `json:"credential"`
	SourceResourceID    string   `json:"source_resource_id"`
	AccessMode          string   `json:"access_mode"`
	Version             string   `json:"version"`
	VPCID               string   `json:"vpc_id"`
	SubnetRefs          []string `json:"subnet_refs"`
	Description         string   `json:"description"`
}

type NamespaceListItem struct {
	ID          uint      `json:"id"`
	ClusterID   uint      `json:"cluster_id"`
	ClusterName string    `json:"cluster_name"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	SourceType  string    `json:"source_type"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type NamespaceInput struct {
	ClusterID   uint   `json:"cluster_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	SourceType  string `json:"source_type"`
	Status      string `json:"status"`
}

type ScaleWorkloadInput struct {
	Replicas int `json:"replicas" binding:"min=0"`
}

type UpdatePrimaryImageInput struct {
	Image string `json:"image" binding:"required"`
}

type WorkloadContainerImageInput struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type UpdateWorkloadImagesInput struct {
	Containers []WorkloadContainerImageInput `json:"containers"`
}

type ExecWorkloadCommandInput struct {
	PodName        string `json:"pod_name"`
	ContainerName  string `json:"container_name"`
	Command        string `json:"command" binding:"required"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type OpenShellSessionInput struct {
	PodName        string `json:"pod_name"`
	ContainerName  string `json:"container_name"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type OpenInteractiveTerminalInput struct {
	PodName        string `json:"pod_name"`
	ContainerName  string `json:"container_name"`
	Shell          string `json:"shell"`
	Cols           int    `json:"cols"`
	Rows           int    `json:"rows"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type ShellSessionInput struct {
	Input string `json:"input" binding:"required"`
}

type UpdateStatefulSetSettingsInput struct {
	ServiceMode         string `json:"service_mode"`
	PodManagementPolicy string `json:"pod_management_policy"`
	RollingPartition    int    `json:"rolling_partition"`
}

type WorkloadContainerInput struct {
	Name                string `json:"name"`
	Image               string `json:"image"`
	PortsText           string `json:"ports_text"`
	CommandText         string `json:"command_text"`
	ArgsText            string `json:"args_text"`
	EnvText             string `json:"env_text"`
	EnvFromText         string `json:"env_from_text"`
	EnvValueFromText    string `json:"env_value_from_text"`
	ImagePullPolicy     string `json:"image_pull_policy"`
	WorkingDir          string `json:"working_dir"`
	Stdin               bool   `json:"stdin"`
	TTY                 bool   `json:"tty"`
	LifecyclePostStart  string `json:"lifecycle_post_start_text"`
	LifecyclePreStop    string `json:"lifecycle_pre_stop_text"`
	VolumeMountsText    string `json:"volume_mounts_text"`
	CPURequest          string `json:"cpu_request"`
	MemoryRequest       string `json:"memory_request"`
	CPULimit            string `json:"cpu_limit"`
	MemoryLimit         string `json:"memory_limit"`
	LivenessProbeYAML   string `json:"liveness_probe_yaml"`
	ReadinessProbeYAML  string `json:"readiness_probe_yaml"`
	StartupProbeYAML    string `json:"startup_probe_yaml"`
	SecurityContextYAML string `json:"security_context_yaml"`
}

type WorkloadVolumeInput struct {
	Name                string `json:"name"`
	Type                string `json:"type"`
	ConfigMapName       string `json:"config_map_name"`
	ConfigMapData       string `json:"config_map_data"`
	PVCName             string `json:"pvc_name"`
	PVCSize             string `json:"pvc_size"`
	PVCStorageClassName string `json:"pvc_storage_class_name"`
	SecretName          string `json:"secret_name"`
	HostPath            string `json:"host_path"`
	NFSServer           string `json:"nfs_server"`
	NFSPath             string `json:"nfs_path"`
	EmptyDirMedium      string `json:"empty_dir_medium"`
}

type WorkloadClaimTemplateInput struct {
	Name             string `json:"name"`
	StorageClassName string `json:"storage_class_name"`
	Size             string `json:"size"`
	AccessModesText  string `json:"access_modes_text"`
	LabelsText       string `json:"labels_text"`
	AnnotationsText  string `json:"annotations_text"`
}

type CreateWorkloadBundleInput struct {
	ClusterID               uint                         `json:"cluster_id" binding:"required"`
	Namespace               string                       `json:"namespace" binding:"required"`
	WorkloadType            string                       `json:"workload_type"`
	Name                    string                       `json:"name" binding:"required"`
	LabelsText              string                       `json:"labels_text"`
	AnnotationsText         string                       `json:"annotations_text"`
	Image                   string                       `json:"image"`
	Replicas                int                          `json:"replicas"`
	ServiceName             string                       `json:"service_name"`
	PodManagementPolicy     string                       `json:"pod_management_policy"`
	StatefulSetServiceMode  string                       `json:"statefulset_service_mode"`
	StatefulSetPartition    int                          `json:"statefulset_partition"`
	Schedule                string                       `json:"schedule"`
	Suspend                 bool                         `json:"suspend"`
	ConcurrencyPolicy       string                       `json:"concurrency_policy"`
	StartingDeadlineSeconds int                          `json:"starting_deadline_seconds"`
	SuccessfulJobsHistory   int                          `json:"successful_jobs_history_limit"`
	FailedJobsHistory       int                          `json:"failed_jobs_history_limit"`
	ContainerPort           int                          `json:"container_port"`
	InitContainers          []WorkloadContainerInput     `json:"init_containers"`
	Containers              []WorkloadContainerInput     `json:"containers"`
	Volumes                 []WorkloadVolumeInput        `json:"volumes"`
	ClaimTemplates          []WorkloadClaimTemplateInput `json:"claim_templates"`
	UpdateStrategy          string                       `json:"update_strategy"`
	MaxSurge                string                       `json:"max_surge"`
	MaxUnavailable          string                       `json:"max_unavailable"`
	NodeSelectorText        string                       `json:"node_selector_text"`
	TolerationsYAML         string                       `json:"tolerations_yaml"`
	AffinityYAML            string                       `json:"affinity_yaml"`
	DNSPolicy               string                       `json:"dns_policy"`
	DNSConfigYAML           string                       `json:"dns_config_yaml"`
	HostAliasesYAML         string                       `json:"host_aliases_yaml"`
	PodSecurityContextYAML  string                       `json:"pod_security_context_yaml"`
	HostNetwork             bool                         `json:"host_network"`
	HostPID                 bool                         `json:"host_pid"`
	HostIPC                 bool                         `json:"host_ipc"`
	ImagePullSecretsText    string                       `json:"image_pull_secrets_text"`
	TopologySpreadYAML      string                       `json:"topology_spread_yaml"`
	TerminationGracePeriod  int                          `json:"termination_grace_period_seconds"`
	CreateService           bool                         `json:"create_service"`
	ServiceType             string                       `json:"service_type"`
	ServicePort             int                          `json:"service_port"`
	CreateIngress           bool                         `json:"create_ingress"`
	IngressHost             string                       `json:"ingress_host"`
	IngressPath             string                       `json:"ingress_path"`
	IngressClassName        string                       `json:"ingress_class_name"`
	CreateServiceAccount    bool                         `json:"create_service_account"`
	ServiceAccountName      string                       `json:"service_account_name"`
	CreateConfigMap         bool                         `json:"create_config_map"`
	ConfigMapName           string                       `json:"config_map_name"`
	ConfigMapData           string                       `json:"config_map_data"`
	MountConfigMap          bool                         `json:"mount_config_map"`
	ConfigMountPath         string                       `json:"config_mount_path"`
	CreatePVC               bool                         `json:"create_pvc"`
	PVCName                 string                       `json:"pvc_name"`
	PVCStorageClassName     string                       `json:"pvc_storage_class_name"`
	PVCSize                 string                       `json:"pvc_size"`
	MountPVCPath            string                       `json:"mount_pvc_path"`
}

type ApplyManifestInput struct {
	ClusterID    uint   `json:"cluster_id" binding:"required"`
	Namespace    string `json:"namespace"`
	ManifestYAML string `json:"manifest_yaml" binding:"required"`
}

type UpdateNamespaceResourceInput struct {
	ConfigMap      *UpdateConfigMapResourceInput      `json:"config_map"`
	Secret         *UpdateSecretResourceInput         `json:"secret"`
	Service        *UpdateServiceResourceInput        `json:"service"`
	Ingress        *UpdateIngressResourceInput        `json:"ingress"`
	ResourceQuota  *UpdateResourceQuotaResourceInput  `json:"resource_quota"`
	LimitRange     *UpdateLimitRangeResourceInput     `json:"limit_range"`
	ServiceAccount *UpdateServiceAccountResourceInput `json:"service_account"`
	PVC            *UpdatePersistentVolumeClaimInput  `json:"persistent_volume_claim"`
}

type UpdateConfigMapResourceInput struct {
	EntriesText string `json:"entries_text"`
}

type UpdateSecretResourceInput struct {
	EntriesText     string `json:"entries_text"`
	RemoveKeysText  string `json:"remove_keys_text"`
	AnnotationsText string `json:"annotations_text"`
}

type UpdateServiceResourceInput struct {
	Type            string `json:"type"`
	SessionAffinity string `json:"session_affinity"`
	SelectorText    string `json:"selector_text"`
	ExternalIPsText string `json:"external_ips_text"`
	PortsText       string `json:"ports_text"`
}

type UpdateIngressResourceInput struct {
	IngressClass    string `json:"ingress_class"`
	AnnotationsText string `json:"annotations_text"`
	RulesText       string `json:"rules_text"`
	TLSText         string `json:"tls_text"`
}

type UpdateResourceQuotaResourceInput struct {
	HardText   string `json:"hard_text"`
	ScopesText string `json:"scopes_text"`
}

type UpdateLimitRangeResourceInput struct {
	LimitsText string `json:"limits_text"`
}

type UpdateServiceAccountResourceInput struct {
	ImagePullSecretsText string `json:"image_pull_secrets_text"`
	AnnotationsText      string `json:"annotations_text"`
}

type UpdatePersistentVolumeClaimInput struct {
	RequestedStorage string `json:"requested_storage"`
	AnnotationsText  string `json:"annotations_text"`
}

type ResourceListItem struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Summary   string `json:"summary"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type WorkloadListItem struct {
	ID            uint      `json:"id"`
	ClusterID     uint      `json:"cluster_id"`
	NamespaceID   *uint     `json:"namespace_id"`
	NamespaceName string    `json:"namespace_name"`
	Name          string    `json:"name"`
	Kind          string    `json:"kind"`
	Image         string    `json:"image"`
	Replicas      int       `json:"replicas"`
	ReadyReplicas int       `json:"ready_replicas"`
	Status        string    `json:"status"`
	SourceType    string    `json:"source_type"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type WorkloadDetail struct {
	ID            uint      `json:"id"`
	ClusterID     uint      `json:"cluster_id"`
	NamespaceID   *uint     `json:"namespace_id"`
	NamespaceName string    `json:"namespace_name"`
	Name          string    `json:"name"`
	Kind          string    `json:"kind"`
	Image         string    `json:"image"`
	Replicas      int       `json:"replicas"`
	ReadyReplicas int       `json:"ready_replicas"`
	Status        string    `json:"status"`
	Labels        string    `json:"labels"`
	Annotations   string    `json:"annotations"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type RelatedPod struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	NodeName   string `json:"node_name"`
	PodIP      string `json:"pod_ip"`
	Containers int    `json:"containers"`
}

type RolloutReplicaSet struct {
	Name          string `json:"name"`
	Desired       int    `json:"desired"`
	Ready         int    `json:"ready"`
	Available     int    `json:"available"`
	Revision      string `json:"revision"`
	CreationStamp string `json:"creation_stamp"`
}

type RelatedService struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	ClusterIP  string   `json:"cluster_ip"`
	Ports      []string `json:"ports"`
	Selector   []string `json:"selector"`
	Endpoints  []string `json:"endpoints"`
	ExternalIP []string `json:"external_ips"`
}

type RelatedIngress struct {
	Name      string   `json:"name"`
	ClassName string   `json:"class_name"`
	Hosts     []string `json:"hosts"`
	Paths     []string `json:"paths"`
	Backends  []string `json:"backends"`
	Addresses []string `json:"addresses"`
}

type RelatedPersistentVolumeClaim struct {
	Name         string   `json:"name"`
	Status       string   `json:"status"`
	Volume       string   `json:"volume"`
	StorageClass string   `json:"storage_class"`
	AccessModes  []string `json:"access_modes"`
	Requested    string   `json:"requested"`
	Capacity     string   `json:"capacity"`
	TemplateName string   `json:"template_name"`
}

type StatefulSetClaimTemplate struct {
	Name         string            `json:"name"`
	StorageClass string            `json:"storage_class"`
	AccessModes  []string          `json:"access_modes"`
	Requested    string            `json:"requested"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
}

type ServiceDetail struct {
	Name            string            `json:"name"`
	Type            string            `json:"type"`
	ClusterIP       string            `json:"cluster_ip"`
	SessionAffinity string            `json:"session_affinity"`
	Selector        []string          `json:"selector"`
	ExternalIPs     []string          `json:"external_ips"`
	Ports           []map[string]any  `json:"ports"`
	Endpoints       []string          `json:"endpoints"`
	SelectedBy      []map[string]any  `json:"selected_by"`
	Annotations     map[string]string `json:"annotations"`
	CreatedAt       string            `json:"created_at"`
}

type IngressDetail struct {
	Name            string            `json:"name"`
	IngressClass    string            `json:"ingress_class"`
	DefaultBackend  string            `json:"default_backend"`
	Addresses       []string          `json:"addresses"`
	Rules           []map[string]any  `json:"rules"`
	TLS             []map[string]any  `json:"tls"`
	BackendServices []map[string]any  `json:"backend_services"`
	Annotations     map[string]string `json:"annotations"`
	CreatedAt       string            `json:"created_at"`
}

type ConfigMapDetail struct {
	Name         string            `json:"name"`
	DataCount    int               `json:"data_count"`
	Entries      []map[string]any  `json:"entries"`
	ReferencedBy []map[string]any  `json:"referenced_by"`
	Annotations  map[string]string `json:"annotations"`
	CreatedAt    string            `json:"created_at"`
}

type SecretDetail struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	DataCount    int               `json:"data_count"`
	Keys         []string          `json:"keys"`
	KeySizes     []map[string]any  `json:"key_sizes"`
	ReferencedBy []map[string]any  `json:"referenced_by"`
	Annotations  map[string]string `json:"annotations"`
	CreatedAt    string            `json:"created_at"`
}

type ServiceAccountDetail struct {
	Name         string            `json:"name"`
	Secrets      []string          `json:"secrets"`
	ImagePullRef []string          `json:"image_pull_secrets"`
	ReferencedBy []map[string]any  `json:"referenced_by"`
	Annotations  map[string]string `json:"annotations"`
	CreatedAt    string            `json:"created_at"`
}

type PersistentVolumeClaimDetail struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	Volume       string            `json:"volume"`
	StorageClass string            `json:"storage_class"`
	AccessModes  []string          `json:"access_modes"`
	Requested    string            `json:"requested"`
	Capacity     string            `json:"capacity"`
	MountedBy    []map[string]any  `json:"mounted_by"`
	Annotations  map[string]string `json:"annotations"`
	CreatedAt    string            `json:"created_at"`
}

type ResourceQuotaDetail struct {
	Name              string            `json:"name"`
	Hard              map[string]string `json:"hard"`
	Used              map[string]string `json:"used"`
	Scopes            []string          `json:"scopes"`
	ImpactedWorkloads []map[string]any  `json:"impacted_workloads"`
	Annotations       map[string]string `json:"annotations"`
	CreatedAt         string            `json:"created_at"`
}

type LimitRangeDetail struct {
	Name              string            `json:"name"`
	Limits            []map[string]any  `json:"limits"`
	ImpactedWorkloads []map[string]any  `json:"impacted_workloads"`
	Annotations       map[string]string `json:"annotations"`
	CreatedAt         string            `json:"created_at"`
}

type CreatedResource struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ClusterOverviewNode struct {
	Name           string   `json:"name"`
	Ready          bool     `json:"ready"`
	Roles          []string `json:"roles"`
	InternalIP     string   `json:"internal_ip"`
	PodCIDR        string   `json:"pod_cidr"`
	KubeletVersion string   `json:"kubelet_version"`
	OSImage        string   `json:"os_image"`
	CreatedAt      string   `json:"created_at"`
}

type ClusterOverviewEvent struct {
	Type         string `json:"type"`
	Namespace    string `json:"namespace"`
	InvolvedKind string `json:"involved_kind"`
	InvolvedName string `json:"involved_name"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
	Component    string `json:"component"`
	Count        int    `json:"count"`
	Timestamp    string `json:"timestamp"`
}

type NamespaceGovernanceSummary struct {
	Name               string `json:"name"`
	Status             string `json:"status"`
	ResourceQuotaCount int    `json:"resource_quota_count"`
	LimitRangeCount    int    `json:"limit_range_count"`
}
