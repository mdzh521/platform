package user

import "time"

type SourceType string

const (
	SourceLocal SourceType = "local"
	SourceLDAP  SourceType = "ldap"
	SourceSSO   SourceType = "sso"
)

type User struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Username    string     `gorm:"size:128;uniqueIndex;not null" json:"username"`
	DisplayName string     `gorm:"size:128;not null" json:"display_name"`
	Email       string     `gorm:"size:255" json:"email"`
	Status      string     `gorm:"size:32;not null;default:active" json:"status"`
	SourceType  SourceType `gorm:"size:32;not null" json:"source_type"`
	IsExternal  bool       `gorm:"not null;default:false" json:"is_external"`
	MFARequired bool       `gorm:"not null;default:false" json:"mfa_required"`
	MFAEnabled  bool       `gorm:"not null;default:false" json:"mfa_enabled"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type Credential struct {
	UserID            uint      `gorm:"primaryKey"`
	PasswordHash      string    `gorm:"size:255;not null"`
	PasswordUpdatedAt time.Time `gorm:"autoCreateTime"`
}

type ExternalIdentity struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"index;not null" json:"user_id"`
	SourceID         uint      `gorm:"index;not null" json:"source_id"`
	ExternalUID      string    `gorm:"size:255;not null" json:"external_uid"`
	ExternalUsername string    `gorm:"size:255" json:"external_username"`
	ExternalEmail    string    `gorm:"size:255" json:"external_email"`
	RawProfileJSON   string    `gorm:"type:json" json:"raw_profile_json"`
	LastSyncedAt     time.Time `gorm:"autoCreateTime" json:"last_synced_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Group struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Code        string    `gorm:"size:128;not null;uniqueIndex" json:"code"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GroupMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GroupID   uint      `gorm:"not null;index" json:"group_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
