package machine

import "time"

type Asset struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	Name                string     `gorm:"size:128;not null" json:"name"`
	Address             string     `gorm:"size:255;not null;index" json:"address"`
	PrivateIP           string     `gorm:"size:255;index" json:"private_ip"`
	Platform            string     `gorm:"size:32;not null;index" json:"platform"`
	Protocol            string     `gorm:"size:32;not null;index" json:"protocol"`
	ProjectID           *uint      `gorm:"index" json:"project_id,omitempty"`
	EnvironmentID       *uint      `gorm:"index" json:"environment_id,omitempty"`
	StackID             *uint      `gorm:"index" json:"stack_id,omitempty"`
	FoundationNetworkID *uint      `gorm:"index" json:"foundation_network_id,omitempty"`
	AccessMode          string     `gorm:"size:32;not null;default:direct;index" json:"access_mode"`
	EnrollmentStatus    string     `gorm:"size:32;not null;default:pending;index" json:"enrollment_status"`
	EnrollmentError     string     `gorm:"type:text" json:"enrollment_error"`
	GatewayAssetID      *uint      `gorm:"index" json:"gateway_asset_id,omitempty"`
	IsGatewayNode       bool       `gorm:"not null;default:false;index" json:"is_gateway_node"`
	SourceType          string     `gorm:"size:32;not null;default:manual;index" json:"source_type"`
	SourceProvider      string     `gorm:"size:32;index" json:"source_provider"`
	SourceResourceID    string     `gorm:"size:255;index" json:"source_resource_id"`
	SourceJobID         *uint      `gorm:"index" json:"source_job_id,omitempty"`
	GroupName           string     `gorm:"size:128;index" json:"group_name"`
	LoginPolicy         string     `gorm:"size:32;not null;default:managed_first;index" json:"login_policy"`
	Port                int        `gorm:"not null;default:22" json:"port"`
	Account             string     `gorm:"size:128" json:"account"`
	Status              string     `gorm:"size:32;not null;default:online;index" json:"status"`
	Tags                string     `gorm:"type:text" json:"tags"`
	Description         string     `gorm:"size:255" json:"description"`
	LastSeenAt          *time.Time `json:"last_seen_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type AssetGroup struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Name               string    `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Code               string    `gorm:"size:128;not null;uniqueIndex" json:"code"`
	ParentID           *uint     `gorm:"index" json:"parent_id,omitempty"`
	DefaultLoginPolicy string    `gorm:"size:32;not null;default:managed_first" json:"default_login_policy"`
	Description        string    `gorm:"size:255" json:"description"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type Session struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	AssetID   *uint      `gorm:"index" json:"asset_id"`
	AssetName string     `gorm:"size:128;not null" json:"asset_name"`
	Address   string     `gorm:"size:255;not null" json:"address"`
	Protocol  string     `gorm:"size:32;not null;index" json:"protocol"`
	Account   string     `gorm:"size:128" json:"account"`
	Status    string     `gorm:"size:32;not null;default:prepared;index" json:"status"`
	Detail    string     `gorm:"size:255" json:"detail"`
	StartedAt time.Time  `gorm:"autoCreateTime" json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type SessionEvent struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	AssetID    uint      `gorm:"not null;index" json:"asset_id"`
	SessionID  *uint     `gorm:"index" json:"session_id"`
	AssetName  string    `gorm:"size:128;not null" json:"asset_name"`
	EventType  string    `gorm:"size:64;not null;index" json:"event_type"`
	EventLevel string    `gorm:"size:32;not null;default:info;index" json:"event_level"`
	Summary    string    `gorm:"size:255;not null" json:"summary"`
	Detail     string    `gorm:"type:text" json:"detail"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type SSHHostTrust struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	Address           string     `gorm:"size:255;not null;uniqueIndex:idx_ssh_host_trust" json:"address"`
	Port              int        `gorm:"not null;uniqueIndex:idx_ssh_host_trust" json:"port"`
	Algorithm         string     `gorm:"size:64;not null" json:"algorithm"`
	FingerprintSHA256 string     `gorm:"size:255;not null" json:"fingerprint_sha256"`
	FingerprintMD5    string     `gorm:"size:255;not null" json:"fingerprint_md5"`
	FirstSeenAt       time.Time  `json:"first_seen_at"`
	LastVerifiedAt    *time.Time `json:"last_verified_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type Account struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	AssetID             uint      `gorm:"not null;index" json:"asset_id"`
	Name                string    `gorm:"size:128;not null" json:"name"`
	Username            string    `gorm:"size:128;not null" json:"username"`
	AuthType            string    `gorm:"size:32;not null;default:password" json:"auth_type"`
	PasswordEncrypted   string    `gorm:"type:text" json:"-"`
	PrivateKeyEncrypted string    `gorm:"type:longtext" json:"-"`
	PassphraseEncrypted string    `gorm:"type:text" json:"-"`
	Description         string    `gorm:"size:255" json:"description"`
	IsDefault           bool      `gorm:"not null;default:false" json:"is_default"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type CredentialLibrary struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	Name                string    `gorm:"size:128;not null" json:"name"`
	Username            string    `gorm:"size:128;not null" json:"username"`
	AuthType            string    `gorm:"size:32;not null;default:password" json:"auth_type"`
	PasswordEncrypted   string    `gorm:"type:text" json:"-"`
	PrivateKeyEncrypted string    `gorm:"type:longtext" json:"-"`
	PassphraseEncrypted string    `gorm:"type:text" json:"-"`
	SourceType          string    `gorm:"size:64;index" json:"source_type"`
	Provider            string    `gorm:"size:32;index" json:"provider"`
	ScopeKind           string    `gorm:"size:64;index" json:"scope_kind"`
	ScopeRef            string    `gorm:"size:128;index" json:"scope_ref"`
	ScopeName           string    `gorm:"size:128" json:"scope_name"`
	AssetID             *uint     `gorm:"index" json:"asset_id,omitempty"`
	AssetName           string    `gorm:"size:128" json:"asset_name"`
	Description         string    `gorm:"size:255" json:"description"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type QuickCommand struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Kind        string    `gorm:"size:32;not null;default:command" json:"kind"`
	Content     string    `gorm:"type:longtext;not null" json:"content"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
