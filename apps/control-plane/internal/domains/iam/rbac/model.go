package rbac

import "time"

type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Code        string    `gorm:"size:128;not null;uniqueIndex" json:"code"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Permission struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"size:128;not null;uniqueIndex" json:"code"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Resource    string    `gorm:"size:128;not null" json:"resource"`
	Action      string    `gorm:"size:128;not null" json:"action"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RolePermission struct {
	ID           uint `gorm:"primaryKey"`
	RoleID       uint `gorm:"uniqueIndex:idx_role_permission"`
	PermissionID uint `gorm:"uniqueIndex:idx_role_permission"`
}

type SubjectRoleBinding struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	SubjectType string     `gorm:"size:32;not null" json:"subject_type"`
	SubjectID   string     `gorm:"size:128;not null;index" json:"subject_id"`
	RoleID      uint       `gorm:"not null" json:"role_id"`
	SourceID    *uint      `json:"source_id"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
