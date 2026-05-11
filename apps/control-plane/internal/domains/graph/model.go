package graph

import "time"

type Node struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ScopeKind    string    `gorm:"size:32;not null;index" json:"scope_kind"`
	ScopeRef     string    `gorm:"size:64;not null;index" json:"scope_ref"`
	NodeKey      string    `gorm:"size:255;not null;uniqueIndex" json:"node_key"`
	NodeType     string    `gorm:"size:64;not null;index" json:"node_type"`
	RefID        string    `gorm:"size:64;not null;index" json:"ref_id"`
	Name         string    `gorm:"size:128;not null" json:"name"`
	Status       string    `gorm:"size:32;not null;default:active;index" json:"status"`
	MetadataJSON string    `gorm:"type:longtext" json:"metadata_json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Edge struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ScopeKind    string    `gorm:"size:32;not null;index" json:"scope_kind"`
	ScopeRef     string    `gorm:"size:64;not null;index" json:"scope_ref"`
	EdgeKey      string    `gorm:"size:255;not null;uniqueIndex" json:"edge_key"`
	FromNodeKey  string    `gorm:"size:255;not null;index" json:"from_node_key"`
	ToNodeKey    string    `gorm:"size:255;not null;index" json:"to_node_key"`
	Relation     string    `gorm:"size:64;not null;index" json:"relation"`
	MetadataJSON string    `gorm:"type:longtext" json:"metadata_json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
