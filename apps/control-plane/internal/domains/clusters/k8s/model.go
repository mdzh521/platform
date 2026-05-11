package k8s

import "time"

type Cluster struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	Name                string     `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Code                string     `gorm:"size:128;not null;uniqueIndex" json:"code"`
	Environment         string     `gorm:"size:64;not null" json:"environment"`
	ProjectID           *uint      `gorm:"index" json:"project_id,omitempty"`
	EnvironmentID       *uint      `gorm:"index" json:"environment_id,omitempty"`
	StackID             *uint      `gorm:"index" json:"stack_id,omitempty"`
	FoundationNetworkID *uint      `gorm:"index" json:"foundation_network_id,omitempty"`
	Provider            string     `gorm:"size:64;not null" json:"provider"`
	APIEndpoint         string     `gorm:"size:255;not null" json:"api_endpoint"`
	AuthType            string     `gorm:"size:64;not null" json:"auth_type"`
	CredentialJSON      string     `gorm:"type:text" json:"credential_json,omitempty"`
	SourceResourceID    string     `gorm:"size:255;index" json:"source_resource_id"`
	ClusterAccessMode   string     `gorm:"size:32;not null;default:direct;index" json:"cluster_access_mode"`
	Version             string     `gorm:"size:64" json:"version"`
	VPCID               string     `gorm:"size:255" json:"vpc_id"`
	SubnetRefsJSON      string     `gorm:"type:text" json:"subnet_refs_json"`
	Description         string     `gorm:"size:255" json:"description"`
	Status              string     `gorm:"size:32;not null;default:draft" json:"status"`
	LastCheckedAt       *time.Time `json:"last_checked_at"`
	LastSyncedAt        *time.Time `json:"last_synced_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type Namespace struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ClusterID   uint      `gorm:"not null;index;uniqueIndex:idx_cluster_namespace" json:"cluster_id"`
	Name        string    `gorm:"size:128;not null;uniqueIndex:idx_cluster_namespace" json:"name"`
	DisplayName string    `gorm:"size:128" json:"display_name"`
	Description string    `gorm:"size:255" json:"description"`
	LabelsJSON  string    `gorm:"type:text" json:"labels_json"`
	SourceType  string    `gorm:"size:32;not null;default:manual" json:"source_type"`
	Status      string    `gorm:"size:32;not null;default:active" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Workload struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ClusterID     uint      `gorm:"not null;index;uniqueIndex:idx_cluster_workload" json:"cluster_id"`
	NamespaceID   *uint     `gorm:"index" json:"namespace_id"`
	NamespaceName string    `gorm:"size:128;not null;index;uniqueIndex:idx_cluster_workload" json:"namespace_name"`
	Kind          string    `gorm:"size:64;not null;index;uniqueIndex:idx_cluster_workload" json:"kind"`
	Name          string    `gorm:"size:128;not null;index;uniqueIndex:idx_cluster_workload" json:"name"`
	ReadyReplicas int       `json:"ready_replicas"`
	Replicas      int       `json:"replicas"`
	Image         string    `gorm:"size:255" json:"image"`
	Status        string    `gorm:"size:32;not null;default:unknown" json:"status"`
	SourceType    string    `gorm:"size:32;not null;default:sync" json:"source_type"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
