package identitysource

import "time"

type IdentitySource struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Type         string    `gorm:"size:32;not null" json:"type"`
	Enabled      bool      `gorm:"not null;default:true" json:"enabled"`
	Host         string    `gorm:"size:255" json:"host"`
	Port         int       `json:"port"`
	BaseDN       string    `gorm:"size:255" json:"base_dn"`
	BindDN       string    `gorm:"size:255" json:"bind_dn"`
	BindPassword string    `gorm:"size:255" json:"bind_password,omitempty"`
	UserFilter   string    `gorm:"size:255" json:"user_filter"`
	GroupFilter  string    `gorm:"size:255" json:"group_filter"`
	ClientID     string    `gorm:"size:255" json:"client_id"`
	ClientSecret string    `gorm:"size:255" json:"client_secret,omitempty"`
	IssuerURL    string    `gorm:"size:255" json:"issuer_url"`
	RedirectURL  string    `gorm:"size:255" json:"redirect_url"`
	SyncMode     string    `gorm:"size:64;default:login_only" json:"sync_mode"`
	ConfigJSON   string    `gorm:"type:json" json:"config_json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
