package machine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	platformcache "backend-center/internal/infra/cache"
	"backend-center/internal/infra/secure"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	cache  platformcache.Store
	cipher *secure.Cipher
}

type Summary struct {
	TotalAssets     int64            `json:"total_assets"`
	TotalGroups     int64            `json:"total_groups"`
	OnlineAssets    int64            `json:"online_assets"`
	WarningAssets   int64            `json:"warning_assets"`
	OfflineAssets   int64            `json:"offline_assets"`
	RecentSessions  int64            `json:"recent_sessions"`
	GroupBreakup    map[string]int64 `json:"group_breakup"`
	PlatformBreakup map[string]int64 `json:"platform_breakup"`
	ProtocolBreakup map[string]int64 `json:"protocol_breakup"`
}

type AssetView struct {
	ID                     uint       `json:"id"`
	Name                   string     `json:"name"`
	Address                string     `json:"address"`
	PrivateIP              string     `json:"private_ip"`
	Platform               string     `json:"platform"`
	Protocol               string     `json:"protocol"`
	ProjectID              *uint      `json:"project_id,omitempty"`
	EnvironmentID          *uint      `json:"environment_id,omitempty"`
	StackID                *uint      `json:"stack_id,omitempty"`
	FoundationNetworkID    *uint      `json:"foundation_network_id,omitempty"`
	AccessMode             string     `json:"access_mode"`
	GatewayAssetID         *uint      `json:"gateway_asset_id,omitempty"`
	GatewayAssetName       string     `json:"gateway_asset_name,omitempty"`
	GatewayAddress         string     `json:"gateway_address,omitempty"`
	IsGatewayNode          bool       `json:"is_gateway_node"`
	SourceType             string     `json:"source_type"`
	SourceProvider         string     `json:"source_provider"`
	SourceResourceID       string     `json:"source_resource_id"`
	SourceJobID            *uint      `json:"source_job_id,omitempty"`
	EnrollmentStatus       string     `json:"enrollment_status"`
	EnrollmentError        string     `json:"enrollment_error"`
	GroupName              string     `json:"group_name"`
	LoginPolicy            string     `json:"login_policy"`
	Port                   int        `json:"port"`
	Account                string     `json:"account"`
	Status                 string     `json:"status"`
	Tags                   []string   `json:"tags"`
	Description            string     `json:"description"`
	LastSeenAt             *time.Time `json:"last_seen_at"`
	HasAccounts            bool       `json:"has_accounts"`
	AccountCount           int        `json:"account_count"`
	DefaultAccountID       *uint      `json:"default_account_id,omitempty"`
	DefaultAccountName     string     `json:"default_account_name,omitempty"`
	DefaultAccountUsername string     `json:"default_account_username,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type AssetListInput struct {
	Search   string
	Group    string
	Status   string
	Page     int
	PageSize int
	SortBy   string
	Order    string
}

type PaginatedAssetView struct {
	Items      []AssetView `json:"items"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalItems int64       `json:"total_items"`
	TotalPages int         `json:"total_pages"`
}

type AssetGroupView struct {
	ID                 uint             `json:"id"`
	Name               string           `json:"name"`
	Code               string           `json:"code"`
	ParentID           *uint            `json:"parent_id,omitempty"`
	DefaultLoginPolicy string           `json:"default_login_policy"`
	Description        string           `json:"description"`
	AssetCount         int64            `json:"asset_count"`
	Depth              int              `json:"depth"`
	Children           []AssetGroupView `json:"children,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}

type SessionView struct {
	ID        uint       `json:"id"`
	AssetID   *uint      `json:"asset_id"`
	AssetName string     `json:"asset_name"`
	Address   string     `json:"address"`
	Protocol  string     `json:"protocol"`
	Account   string     `json:"account"`
	Status    string     `json:"status"`
	Detail    string     `json:"detail"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
}

type SessionEventView struct {
	ID         uint      `json:"id"`
	AssetID    uint      `json:"asset_id"`
	SessionID  *uint     `json:"session_id"`
	AssetName  string    `json:"asset_name"`
	EventType  string    `json:"event_type"`
	EventLevel string    `json:"event_level"`
	Summary    string    `json:"summary"`
	Detail     string    `json:"detail"`
	CreatedAt  time.Time `json:"created_at"`
}

type AccountView struct {
	ID          uint      `json:"id"`
	AssetID     uint      `json:"asset_id"`
	Name        string    `json:"name"`
	Username    string    `json:"username"`
	AuthType    string    `json:"auth_type"`
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CredentialView struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Username    string    `json:"username"`
	AuthType    string    `json:"auth_type"`
	SourceType  string    `json:"source_type"`
	Provider    string    `json:"provider"`
	ScopeKind   string    `json:"scope_kind"`
	ScopeRef    string    `json:"scope_ref"`
	ScopeName   string    `json:"scope_name"`
	AssetID     *uint     `json:"asset_id,omitempty"`
	AssetName   string    `json:"asset_name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CredentialDetailView struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Username    string    `json:"username"`
	AuthType    string    `json:"auth_type"`
	SourceType  string    `json:"source_type"`
	Provider    string    `json:"provider"`
	ScopeKind   string    `json:"scope_kind"`
	ScopeRef    string    `json:"scope_ref"`
	ScopeName   string    `json:"scope_name"`
	AssetID     *uint     `json:"asset_id,omitempty"`
	AssetName   string    `json:"asset_name"`
	Password    string    `json:"password"`
	PrivateKey  string    `json:"private_key"`
	Passphrase  string    `json:"passphrase"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type QuickCommandView struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Content     string    `json:"content"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AccountDetailView struct {
	ID          uint      `json:"id"`
	AssetID     uint      `json:"asset_id"`
	Name        string    `json:"name"`
	Username    string    `json:"username"`
	AuthType    string    `json:"auth_type"`
	Password    string    `json:"password"`
	PrivateKey  string    `json:"private_key"`
	Passphrase  string    `json:"passphrase"`
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateAssetInput struct {
	Name           string `json:"name"`
	Address        string `json:"address"`
	Platform       string `json:"platform"`
	Protocol       string `json:"protocol"`
	AccessMode     string `json:"access_mode"`
	GatewayAssetID *uint  `json:"gateway_asset_id"`
	IsGatewayNode  bool   `json:"is_gateway_node"`
	GroupName      string `json:"group_name"`
	LoginPolicy    string `json:"login_policy"`
	Port           int    `json:"port"`
	Account        string `json:"account"`
	Status         string `json:"status"`
	Tags           string `json:"tags"`
	Description    string `json:"description"`
}

type UpdateAssetInput struct {
	Name           string `json:"name"`
	Address        string `json:"address"`
	Platform       string `json:"platform"`
	Protocol       string `json:"protocol"`
	AccessMode     string `json:"access_mode"`
	GatewayAssetID *uint  `json:"gateway_asset_id"`
	IsGatewayNode  bool   `json:"is_gateway_node"`
	GroupName      string `json:"group_name"`
	LoginPolicy    string `json:"login_policy"`
	Port           int    `json:"port"`
	Account        string `json:"account"`
	Status         string `json:"status"`
	Tags           string `json:"tags"`
	Description    string `json:"description"`
}

type BatchAssetCreateInput struct {
	Items          []BatchAssetCreateItem `json:"items"`
	Platform       string                 `json:"platform"`
	Protocol       string                 `json:"protocol"`
	AccessMode     string                 `json:"access_mode"`
	GatewayAssetID *uint                  `json:"gateway_asset_id"`
	IsGatewayNode  bool                   `json:"is_gateway_node"`
	GroupName      string                 `json:"group_name"`
	LoginPolicy    string                 `json:"login_policy"`
	Port           int                    `json:"port"`
	Account        string                 `json:"account"`
	Status         string                 `json:"status"`
	Tags           string                 `json:"tags"`
	Description    string                 `json:"description"`
}

type BatchAssetCreateItem struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
	Account string `json:"account"`
}

type BatchAssetUpdateInput struct {
	IDs                   []uint `json:"ids"`
	GroupName             string `json:"group_name"`
	LoginPolicy           string `json:"login_policy"`
	Port                  int    `json:"port"`
	Account               string `json:"account"`
	Status                string `json:"status"`
	Tags                  string `json:"tags"`
	Description           string `json:"description"`
	CredentialName        string `json:"credential_name"`
	CredentialUsername    string `json:"credential_username"`
	CredentialAuthType    string `json:"credential_auth_type"`
	CredentialPassword    string `json:"credential_password"`
	CredentialPrivateKey  string `json:"credential_private_key"`
	CredentialPassphrase  string `json:"credential_passphrase"`
	CredentialDescription string `json:"credential_description"`
}

type BatchAssetDeleteInput struct {
	IDs []uint `json:"ids"`
}

type BatchAssetResult struct {
	Count int `json:"count"`
}

type CreateAssetGroupInput struct {
	Name               string `json:"name"`
	Code               string `json:"code"`
	ParentID           *uint  `json:"parent_id"`
	DefaultLoginPolicy string `json:"default_login_policy"`
	Description        string `json:"description"`
}

type UpdateAssetGroupInput struct {
	Name               string `json:"name"`
	Code               string `json:"code"`
	ParentID           *uint  `json:"parent_id"`
	DefaultLoginPolicy string `json:"default_login_policy"`
	Description        string `json:"description"`
}

type CreateAccountInput struct {
	Name        string `json:"name"`
	Username    string `json:"username"`
	AuthType    string `json:"auth_type"`
	Password    string `json:"password"`
	PrivateKey  string `json:"private_key"`
	Passphrase  string `json:"passphrase"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

type CreateCredentialInput struct {
	Name        string `json:"name"`
	Username    string `json:"username"`
	AuthType    string `json:"auth_type"`
	Password    string `json:"password"`
	PrivateKey  string `json:"private_key"`
	Passphrase  string `json:"passphrase"`
	SourceType  string `json:"source_type"`
	Provider    string `json:"provider"`
	ScopeKind   string `json:"scope_kind"`
	ScopeRef    string `json:"scope_ref"`
	ScopeName   string `json:"scope_name"`
	AssetID     *uint  `json:"asset_id"`
	AssetName   string `json:"asset_name"`
	Description string `json:"description"`
}

type CreateQuickCommandInput struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Content     string `json:"content"`
	Description string `json:"description"`
}

type UpdateQuickCommandInput struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Content     string `json:"content"`
	Description string `json:"description"`
}

type ImportCredentialInput struct {
	CredentialID uint `json:"credential_id"`
	IsDefault    bool `json:"is_default"`
}

type QuickConnectInput struct {
	AssetID      uint   `json:"asset_id"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	Platform     string `json:"platform"`
	Protocol     string `json:"protocol"`
	GroupName    string `json:"group_name"`
	LoginPolicy  string `json:"login_policy"`
	Port         int    `json:"port"`
	Account      string `json:"account"`
	Description  string `json:"description"`
	SaveToAssets bool   `json:"save_to_assets"`
}

type QuickConnectResult struct {
	Asset   *AssetView  `json:"asset,omitempty"`
	Session SessionView `json:"session"`
	Message string      `json:"message"`
}

type TerminalTicketInput struct {
	AccountID  uint   `json:"account_id"`
	AuthType   string `json:"auth_type"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
	Passphrase string `json:"passphrase"`
}

type TerminalTicketResult struct {
	Ticket       string      `json:"ticket"`
	Session      SessionView `json:"session"`
	Asset        AssetView   `json:"asset"`
	TerminalMode string      `json:"terminal_mode"`
	Message      string      `json:"message"`
}

type SFTPEntryView struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Type    string    `json:"type"`
	Size    int64     `json:"size"`
	Mode    string    `json:"mode"`
	Owner   string    `json:"owner"`
	Group   string    `json:"group"`
	ModTime time.Time `json:"mod_time"`
}

type SFTPListResult struct {
	Path    string          `json:"path"`
	Parent  string          `json:"parent"`
	Entries []SFTPEntryView `json:"entries"`
}

type terminalTicketPayload struct {
	AssetID           uint      `json:"asset_id"`
	SessionID         uint      `json:"session_id"`
	Address           string    `json:"address"`
	Port              int       `json:"port"`
	Protocol          string    `json:"protocol"`
	Username          string    `json:"username"`
	AuthType          string    `json:"auth_type"`
	Password          string    `json:"password"`
	PrivateKey        string    `json:"private_key"`
	Passphrase        string    `json:"passphrase"`
	AccessMode        string    `json:"access_mode"`
	GatewayAssetID    uint      `json:"gateway_asset_id,omitempty"`
	GatewayName       string    `json:"gateway_name,omitempty"`
	GatewayAddress    string    `json:"gateway_address,omitempty"`
	GatewayPort       int       `json:"gateway_port,omitempty"`
	GatewayUsername   string    `json:"gateway_username,omitempty"`
	GatewayAuthType   string    `json:"gateway_auth_type,omitempty"`
	GatewayPassword   string    `json:"gateway_password,omitempty"`
	GatewayPrivateKey string    `json:"gateway_private_key,omitempty"`
	GatewayPassphrase string    `json:"gateway_passphrase,omitempty"`
	IssuedAt          time.Time `json:"issued_at"`
}

func NewService(db *gorm.DB, cache platformcache.Store, secret string) *Service {
	service := &Service{db: db, cache: cache, cipher: secure.New(secret)}
	service.recoverStaleSessions(35 * time.Minute)
	return service
}

func (s *Service) Summary() (Summary, error) {
	summary := Summary{
		GroupBreakup:    map[string]int64{},
		PlatformBreakup: map[string]int64{},
		ProtocolBreakup: map[string]int64{},
	}

	if err := s.db.Model(&Asset{}).Count(&summary.TotalAssets).Error; err != nil {
		return summary, err
	}
	if err := s.db.Model(&AssetGroup{}).Count(&summary.TotalGroups).Error; err != nil {
		return summary, err
	}
	if err := s.db.Model(&Asset{}).Where("status = ?", "online").Count(&summary.OnlineAssets).Error; err != nil {
		return summary, err
	}
	if err := s.db.Model(&Asset{}).Where("status = ?", "warning").Count(&summary.WarningAssets).Error; err != nil {
		return summary, err
	}
	if err := s.db.Model(&Asset{}).Where("status = ?", "offline").Count(&summary.OfflineAssets).Error; err != nil {
		return summary, err
	}
	if err := s.db.Model(&Session{}).Where("started_at >= ?", time.Now().Add(-24*time.Hour)).Count(&summary.RecentSessions).Error; err != nil {
		return summary, err
	}

	var platformRows []struct {
		Platform string
		Count    int64
	}
	if err := s.db.Model(&Asset{}).Select("platform, count(*) as count").Group("platform").Scan(&platformRows).Error; err != nil {
		return summary, err
	}
	for _, row := range platformRows {
		summary.PlatformBreakup[row.Platform] = row.Count
	}

	var groupRows []struct {
		GroupName string
		Count     int64
	}
	if err := s.db.Model(&Asset{}).Select("group_name, count(*) as count").Group("group_name").Scan(&groupRows).Error; err != nil {
		return summary, err
	}
	for _, row := range groupRows {
		key := strings.TrimSpace(row.GroupName)
		if key == "" {
			key = "未分组"
		}
		summary.GroupBreakup[key] = row.Count
	}

	var protocolRows []struct {
		Protocol string
		Count    int64
	}
	if err := s.db.Model(&Asset{}).Select("protocol, count(*) as count").Group("protocol").Scan(&protocolRows).Error; err != nil {
		return summary, err
	}
	for _, row := range protocolRows {
		summary.ProtocolBreakup[row.Protocol] = row.Count
	}

	return summary, nil
}

func (s *Service) ListGroups() ([]AssetGroupView, error) {
	var groups []AssetGroup
	if err := s.db.Order("name asc").Find(&groups).Error; err != nil {
		return nil, err
	}
	return s.buildAssetGroupTree(groups), nil
}

func (s *Service) CreateGroup(input CreateAssetGroupInput) (AssetGroupView, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return AssetGroupView{}, errors.New("资产组名称不能为空")
	}
	code := strings.TrimSpace(input.Code)
	if code == "" {
		code = slugifyMachineValue(name)
	}
	group := AssetGroup{
		Name:               name,
		Code:               code,
		ParentID:           input.ParentID,
		DefaultLoginPolicy: normalizeLoginPolicy(input.DefaultLoginPolicy),
		Description:        strings.TrimSpace(input.Description),
	}
	if err := s.validateAssetGroupParent(0, group.ParentID); err != nil {
		return AssetGroupView{}, err
	}
	if err := s.db.Create(&group).Error; err != nil {
		return AssetGroupView{}, err
	}
	return s.assetGroupToView(group, 0), nil
}

func (s *Service) UpdateGroup(id uint, input UpdateAssetGroupInput) (AssetGroupView, error) {
	var group AssetGroup
	if err := s.db.First(&group, id).Error; err != nil {
		return AssetGroupView{}, err
	}
	previousName := group.Name
	name := firstNonEmpty(strings.TrimSpace(input.Name), group.Name)
	code := firstNonEmpty(strings.TrimSpace(input.Code), group.Code)
	parentID := input.ParentID
	if err := s.validateAssetGroupParent(id, parentID); err != nil {
		return AssetGroupView{}, err
	}
	updates := map[string]any{
		"name":                 name,
		"code":                 code,
		"parent_id":            parentID,
		"default_login_policy": normalizeLoginPolicy(firstNonEmpty(input.DefaultLoginPolicy, group.DefaultLoginPolicy)),
		"description":          strings.TrimSpace(input.Description),
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&group).Updates(updates).Error; err != nil {
			return err
		}
		if previousName != name {
			if err := tx.Model(&Asset{}).Where("group_name = ?", previousName).Update("group_name", name).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return AssetGroupView{}, err
	}
	if err := s.db.First(&group, id).Error; err != nil {
		return AssetGroupView{}, err
	}
	return s.assetGroupToView(group, 0), nil
}

func (s *Service) DeleteGroup(id uint) error {
	var group AssetGroup
	if err := s.db.First(&group, id).Error; err != nil {
		return err
	}
	var childCount int64
	if err := s.db.Model(&AssetGroup{}).Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
		return err
	}
	if childCount > 0 {
		return errors.New("请先删除或迁移下级资产组")
	}
	if err := s.db.Model(&Asset{}).Where("group_name = ?", group.Name).Update("group_name", "").Error; err != nil {
		return err
	}
	return s.db.Delete(&group).Error
}

func (s *Service) ListAssets(input AssetListInput) (PaginatedAssetView, error) {
	query := s.db.Model(&Asset{})
	search := strings.TrimSpace(input.Search)
	group := strings.TrimSpace(input.Group)
	status := strings.TrimSpace(input.Status)
	if search != "" {
		like := "%" + search + "%"
		query = query.Where(
			"name LIKE ? OR address LIKE ? OR group_name LIKE ? OR account LIKE ? OR description LIKE ? OR tags LIKE ?",
			like, like, like, like, like, like,
		)
	}
	if group != "" {
		groupNames, err := s.expandAssetGroupNames(group)
		if err != nil {
			return PaginatedAssetView{}, err
		}
		if len(groupNames) > 0 {
			query = query.Where("group_name IN ?", groupNames)
		}
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var totalItems int64
	if err := query.Count(&totalItems).Error; err != nil {
		return PaginatedAssetView{}, err
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := input.Page
	if page <= 0 {
		page = 1
	}
	totalPages := 1
	if totalItems > 0 {
		totalPages = int((totalItems + int64(pageSize) - 1) / int64(pageSize))
	}
	if page > totalPages {
		page = totalPages
	}
	sortBy := "updated_at"
	switch strings.ToLower(strings.TrimSpace(input.SortBy)) {
	case "name":
		sortBy = "name"
	case "address":
		sortBy = "address"
	case "group_name":
		sortBy = "group_name"
	case "account":
		sortBy = "account"
	case "status":
		sortBy = "status"
	case "last_seen_at":
		sortBy = "last_seen_at"
	case "created_at":
		sortBy = "created_at"
	}
	order := "desc"
	if strings.ToLower(strings.TrimSpace(input.Order)) == "asc" {
		order = "asc"
	}
	var assets []Asset
	if err := query.Order(sortBy + " " + order).Offset((page - 1) * pageSize).Limit(pageSize).Find(&assets).Error; err != nil {
		return PaginatedAssetView{}, err
	}
	result := make([]AssetView, 0, len(assets))
	for _, asset := range assets {
		result = append(result, assetToView(asset))
	}
	if err := s.enrichAssetViews(result); err != nil {
		return PaginatedAssetView{}, err
	}
	return PaginatedAssetView{
		Items:      result,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) ListSessions() ([]SessionView, error) {
	var sessions []Session
	if err := s.db.Order("started_at desc").Limit(12).Find(&sessions).Error; err != nil {
		return nil, err
	}
	result := make([]SessionView, 0, len(sessions))
	for _, session := range sessions {
		result = append(result, sessionToView(session))
	}
	return result, nil
}

func (s *Service) ListRecentEvents(limit int) ([]SessionEventView, error) {
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	var events []SessionEvent
	if err := s.db.Order("created_at desc").Limit(limit).Find(&events).Error; err != nil {
		return nil, err
	}
	result := make([]SessionEventView, 0, len(events))
	for _, event := range events {
		result = append(result, sessionEventToView(event))
	}
	return result, nil
}

func (s *Service) GetAsset(id uint) (AssetView, error) {
	var asset Asset
	if err := s.db.First(&asset, id).Error; err != nil {
		return AssetView{}, err
	}
	view := assetToView(asset)
	views := []AssetView{view}
	if err := s.enrichAssetViews(views); err != nil {
		return AssetView{}, err
	}
	return views[0], nil
}

func (s *Service) ListAssetSessions(id uint) ([]SessionView, error) {
	var sessions []Session
	if err := s.db.Where("asset_id = ?", id).Order("started_at desc").Limit(20).Find(&sessions).Error; err != nil {
		return nil, err
	}
	result := make([]SessionView, 0, len(sessions))
	for _, session := range sessions {
		result = append(result, sessionToView(session))
	}
	return result, nil
}

func (s *Service) ListAssetEvents(id uint) ([]SessionEventView, error) {
	var events []SessionEvent
	if err := s.db.Where("asset_id = ?", id).Order("created_at desc").Limit(40).Find(&events).Error; err != nil {
		return nil, err
	}
	result := make([]SessionEventView, 0, len(events))
	for _, event := range events {
		result = append(result, sessionEventToView(event))
	}
	return result, nil
}

func (s *Service) ListAccounts(assetID uint) ([]AccountView, error) {
	var accounts []Account
	if err := s.db.Where("asset_id = ?", assetID).Order("is_default desc, updated_at desc").Find(&accounts).Error; err != nil {
		return nil, err
	}
	result := make([]AccountView, 0, len(accounts))
	for _, account := range accounts {
		result = append(result, accountToView(account))
	}
	return result, nil
}

func (s *Service) ListCredentials() ([]CredentialView, error) {
	var credentials []CredentialLibrary
	if err := s.db.Order("updated_at desc").Find(&credentials).Error; err != nil {
		return nil, err
	}
	result := make([]CredentialView, 0, len(credentials))
	for _, credential := range credentials {
		result = append(result, credentialToView(credential))
	}
	return result, nil
}

func (s *Service) ListQuickCommands() ([]QuickCommandView, error) {
	var commands []QuickCommand
	if err := s.db.Order("updated_at desc, name asc").Find(&commands).Error; err != nil {
		return nil, err
	}
	result := make([]QuickCommandView, 0, len(commands))
	for _, command := range commands {
		result = append(result, quickCommandToView(command))
	}
	return result, nil
}

func (s *Service) GetCredential(id uint) (CredentialDetailView, error) {
	var credential CredentialLibrary
	if err := s.db.First(&credential, id).Error; err != nil {
		return CredentialDetailView{}, err
	}
	return s.credentialToDetailView(credential)
}

func (s *Service) CreateCredential(input CreateCredentialInput) (CredentialView, error) {
	credential, err := s.buildCredentialLibrary(input)
	if err != nil {
		return CredentialView{}, err
	}
	if err := s.db.Create(&credential).Error; err != nil {
		return CredentialView{}, err
	}
	return credentialToView(credential), nil
}

func (s *Service) CreateQuickCommand(input CreateQuickCommandInput) (QuickCommandView, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return QuickCommandView{}, errors.New("快捷命令名称不能为空")
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return QuickCommandView{}, errors.New("快捷命令内容不能为空")
	}
	kind := normalizeField(input.Kind, "command")
	if kind != "command" && kind != "script" {
		kind = "command"
	}
	command := QuickCommand{
		Name:        name,
		Kind:        kind,
		Content:     content,
		Description: strings.TrimSpace(input.Description),
	}
	if err := s.db.Create(&command).Error; err != nil {
		return QuickCommandView{}, err
	}
	return quickCommandToView(command), nil
}

func (s *Service) UpdateQuickCommand(id uint, input UpdateQuickCommandInput) (QuickCommandView, error) {
	var command QuickCommand
	if err := s.db.First(&command, id).Error; err != nil {
		return QuickCommandView{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return QuickCommandView{}, errors.New("快捷命令名称不能为空")
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return QuickCommandView{}, errors.New("快捷命令内容不能为空")
	}
	kind := normalizeField(input.Kind, "command")
	if kind != "command" && kind != "script" {
		kind = "command"
	}
	command.Name = name
	command.Kind = kind
	command.Content = content
	command.Description = strings.TrimSpace(input.Description)
	if err := s.db.Save(&command).Error; err != nil {
		return QuickCommandView{}, err
	}
	return quickCommandToView(command), nil
}

func (s *Service) DeleteCredential(id uint) error {
	return s.db.Delete(&CredentialLibrary{}, id).Error
}

func (s *Service) DeleteQuickCommand(id uint) error {
	return s.db.Delete(&QuickCommand{}, id).Error
}

func (s *Service) CreateAccount(assetID uint, input CreateAccountInput) (AccountView, error) {
	var asset Asset
	if err := s.db.First(&asset, assetID).Error; err != nil {
		return AccountView{}, err
	}
	account, err := s.buildAccount(assetID, input)
	if err != nil {
		return AccountView{}, err
	}
	if account.IsDefault {
		if err := s.db.Model(&Account{}).Where("asset_id = ?", assetID).Update("is_default", false).Error; err != nil {
			return AccountView{}, err
		}
	}
	if err := s.db.Create(&account).Error; err != nil {
		return AccountView{}, err
	}
	if account.IsDefault {
		_ = s.db.Model(&Asset{}).Where("id = ?", assetID).Update("account", account.Username).Error
		s.recordSessionEvent(assetID, 0, "account_default", "info", "更新默认托管账号", "默认登录账号已切换为 "+account.Username+"。")
	}
	return accountToView(account), nil
}

func (s *Service) GetAccount(assetID, accountID uint) (AccountDetailView, error) {
	var account Account
	if err := s.db.Where("asset_id = ? AND id = ?", assetID, accountID).First(&account).Error; err != nil {
		return AccountDetailView{}, err
	}
	return s.accountToDetailView(account)
}

func (s *Service) ImportCredentialToAsset(assetID uint, input ImportCredentialInput) (AccountView, error) {
	var asset Asset
	if err := s.db.First(&asset, assetID).Error; err != nil {
		return AccountView{}, err
	}
	var credential CredentialLibrary
	if err := s.db.First(&credential, input.CredentialID).Error; err != nil {
		return AccountView{}, err
	}
	account := Account{
		AssetID:             assetID,
		Name:                credential.Name,
		Username:            credential.Username,
		AuthType:            credential.AuthType,
		PasswordEncrypted:   credential.PasswordEncrypted,
		PrivateKeyEncrypted: credential.PrivateKeyEncrypted,
		PassphraseEncrypted: credential.PassphraseEncrypted,
		Description:         credential.Description,
		IsDefault:           input.IsDefault,
	}
	if account.IsDefault {
		if err := s.db.Model(&Account{}).Where("asset_id = ?", assetID).Update("is_default", false).Error; err != nil {
			return AccountView{}, err
		}
	}
	if err := s.db.Create(&account).Error; err != nil {
		return AccountView{}, err
	}
	if account.IsDefault {
		_ = s.db.Model(&Asset{}).Where("id = ?", assetID).Update("account", account.Username).Error
	}
	s.recordSessionEvent(assetID, 0, "credential_imported", "info", "导入登录凭据", "已从凭据库导入登录凭据 "+credential.Name+"。")
	return accountToView(account), nil
}

func (s *Service) DeleteAccount(assetID, accountID uint) error {
	var account Account
	if err := s.db.Where("asset_id = ? AND id = ?", assetID, accountID).First(&account).Error; err != nil {
		return err
	}
	if err := s.db.Where("asset_id = ? AND id = ?", assetID, accountID).Delete(&Account{}).Error; err != nil {
		return err
	}
	if account.IsDefault {
		if err := s.applyFallbackDefaultAccount(assetID); err != nil {
			return err
		}
		s.recordSessionEvent(assetID, 0, "account_default", "warning", "默认托管账号已删除", "默认托管账号被删除，已尝试回退到其它账号。")
	}
	return nil
}

func (s *Service) CreateAsset(input CreateAssetInput) (AssetView, error) {
	asset, err := s.buildAssetModel(input)
	if err != nil {
		return AssetView{}, err
	}
	if err := s.db.Create(&asset).Error; err != nil {
		return AssetView{}, err
	}
	return assetToView(asset), nil
}

func (s *Service) buildAssetModel(input CreateAssetInput) (Asset, error) {
	name := strings.TrimSpace(input.Name)
	address := strings.TrimSpace(input.Address)
	if address == "" {
		return Asset{}, errors.New("机器地址不能为空")
	}
	if name == "" {
		name = address
	}
	platform := normalizeField(input.Platform, "linux")
	protocol := normalizeField(input.Protocol, "ssh")
	status := normalizeField(input.Status, "online")
	accessMode := normalizeAssetAccessMode(input.AccessMode)
	gatewayAssetID := input.GatewayAssetID
	if accessMode == "direct" {
		gatewayAssetID = nil
	}
	port := input.Port
	if port <= 0 {
		port = defaultPort(protocol)
	}
	now := time.Now()
	asset := Asset{
		Name:           name,
		Address:        address,
		Platform:       platform,
		Protocol:       protocol,
		AccessMode:     accessMode,
		GatewayAssetID: gatewayAssetID,
		IsGatewayNode:  input.IsGatewayNode,
		GroupName:      strings.TrimSpace(input.GroupName),
		LoginPolicy:    normalizeLoginPolicy(input.LoginPolicy),
		Port:           port,
		Account:        strings.TrimSpace(input.Account),
		Status:         status,
		Tags:           normalizeTags(input.Tags),
		Description:    strings.TrimSpace(input.Description),
		LastSeenAt:     &now,
	}
	if err := s.validateAssetGateway(asset.ID, asset.AccessMode, asset.GatewayAssetID, asset.Protocol); err != nil {
		return Asset{}, err
	}
	return asset, nil
}

func (s *Service) UpdateAsset(id uint, input UpdateAssetInput) (AssetView, error) {
	var asset Asset
	if err := s.db.First(&asset, id).Error; err != nil {
		return AssetView{}, err
	}
	address := strings.TrimSpace(input.Address)
	if address == "" {
		address = asset.Address
	}
	if address == "" {
		return AssetView{}, errors.New("机器地址不能为空")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = address
	}
	protocol := normalizeField(input.Protocol, asset.Protocol)
	port := input.Port
	if port <= 0 {
		port = defaultPort(protocol)
	}
	updates := map[string]any{
		"name":            name,
		"address":         address,
		"platform":        normalizeField(input.Platform, asset.Platform),
		"protocol":        protocol,
		"access_mode":     normalizeAssetAccessMode(firstNonEmpty(input.AccessMode, asset.AccessMode)),
		"group_name":      strings.TrimSpace(input.GroupName),
		"login_policy":    normalizeLoginPolicy(firstNonEmpty(input.LoginPolicy, asset.LoginPolicy)),
		"port":            port,
		"account":         strings.TrimSpace(input.Account),
		"status":          normalizeField(input.Status, asset.Status),
		"tags":            normalizeTags(input.Tags),
		"description":     strings.TrimSpace(input.Description),
		"is_gateway_node": input.IsGatewayNode,
	}
	normalizedGatewayID := input.GatewayAssetID
	if accessMode, _ := updates["access_mode"].(string); accessMode == "direct" {
		normalizedGatewayID = nil
	}
	updates["gateway_asset_id"] = normalizedGatewayID
	if err := s.validateAssetGateway(id, updates["access_mode"].(string), normalizedGatewayID, protocol); err != nil {
		return AssetView{}, err
	}
	if err := s.db.Model(&asset).Updates(updates).Error; err != nil {
		return AssetView{}, err
	}
	if err := s.db.First(&asset, id).Error; err != nil {
		return AssetView{}, err
	}
	s.recordSessionEvent(asset.ID, 0, "asset_updated", "info", "资产信息已更新", "资产基础信息、协议或默认登录信息已更新。")
	return assetToView(asset), nil
}

func (s *Service) DeleteAsset(id uint) error {
	var asset Asset
	if err := s.db.First(&asset, id).Error; err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("asset_id = ?", id).Delete(&Account{}).Error; err != nil {
			return err
		}
		if err := tx.Where("asset_id = ?", id).Delete(&SessionEvent{}).Error; err != nil {
			return err
		}
		if err := tx.Where("asset_id = ?", id).Delete(&Session{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&asset).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *Service) BatchCreateAssets(input BatchAssetCreateInput) (BatchAssetResult, error) {
	items := make([]BatchAssetCreateItem, 0, len(input.Items))
	for _, item := range input.Items {
		address := strings.TrimSpace(item.Address)
		if address == "" {
			continue
		}
		items = append(items, BatchAssetCreateItem{
			Name:    strings.TrimSpace(item.Name),
			Address: address,
			Port:    item.Port,
			Account: strings.TrimSpace(item.Account),
		})
	}
	if len(items) == 0 {
		return BatchAssetResult{}, errors.New("至少提供一台服务器地址")
	}
	count := 0
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			assetInput := CreateAssetInput{
				Name:           firstNonEmpty(item.Name, item.Address),
				Address:        item.Address,
				Platform:       input.Platform,
				Protocol:       input.Protocol,
				AccessMode:     input.AccessMode,
				GatewayAssetID: input.GatewayAssetID,
				IsGatewayNode:  input.IsGatewayNode,
				GroupName:      input.GroupName,
				LoginPolicy:    input.LoginPolicy,
				Port:           firstPositive(item.Port, input.Port),
				Account:        firstNonEmpty(item.Account, input.Account),
				Status:         input.Status,
				Tags:           input.Tags,
				Description:    input.Description,
			}
			asset, err := s.buildAssetModel(assetInput)
			if err != nil {
				return err
			}
			if err := tx.Create(&asset).Error; err != nil {
				return err
			}
			count++
		}
		return nil
	})
	if err != nil {
		return BatchAssetResult{}, err
	}
	return BatchAssetResult{Count: count}, nil
}

func (s *Service) BatchUpdateAssets(input BatchAssetUpdateInput) (BatchAssetResult, error) {
	ids := uniqueUintIDs(input.IDs)
	if len(ids) == 0 {
		return BatchAssetResult{}, errors.New("请至少选择一台服务器")
	}
	applyCredential := strings.TrimSpace(input.CredentialUsername) != "" ||
		strings.TrimSpace(input.CredentialPassword) != "" ||
		strings.TrimSpace(input.CredentialPrivateKey) != ""
	count := 0
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			var asset Asset
			if err := tx.First(&asset, id).Error; err != nil {
				return err
			}
			updates := map[string]any{}
			if strings.TrimSpace(input.GroupName) != "" {
				updates["group_name"] = strings.TrimSpace(input.GroupName)
			}
			if strings.TrimSpace(input.LoginPolicy) != "" {
				updates["login_policy"] = normalizeLoginPolicy(input.LoginPolicy)
			}
			if input.Port > 0 {
				updates["port"] = input.Port
			}
			if strings.TrimSpace(input.Account) != "" {
				updates["account"] = strings.TrimSpace(input.Account)
			}
			if strings.TrimSpace(input.Status) != "" {
				updates["status"] = normalizeField(input.Status, asset.Status)
			}
			if strings.TrimSpace(input.Tags) != "" {
				updates["tags"] = normalizeTags(input.Tags)
			}
			if strings.TrimSpace(input.Description) != "" {
				updates["description"] = strings.TrimSpace(input.Description)
			}
			if len(updates) > 0 {
				if err := tx.Model(&asset).Updates(updates).Error; err != nil {
					return err
				}
			}
			if applyCredential {
				accountInput := CreateAccountInput{
					Name:        input.CredentialName,
					Username:    input.CredentialUsername,
					AuthType:    firstNonEmpty(input.CredentialAuthType, "password"),
					Password:    input.CredentialPassword,
					PrivateKey:  input.CredentialPrivateKey,
					Passphrase:  input.CredentialPassphrase,
					Description: input.CredentialDescription,
					IsDefault:   true,
				}
				if err := s.upsertDefaultAccountTx(tx, asset.ID, accountInput); err != nil {
					return err
				}
				if strings.TrimSpace(input.CredentialUsername) != "" {
					if err := tx.Model(&Asset{}).Where("id = ?", asset.ID).Update("account", strings.TrimSpace(input.CredentialUsername)).Error; err != nil {
						return err
					}
				}
			}
			count++
		}
		return nil
	})
	if err != nil {
		return BatchAssetResult{}, err
	}
	return BatchAssetResult{Count: count}, nil
}

func (s *Service) BatchDeleteAssets(input BatchAssetDeleteInput) (BatchAssetResult, error) {
	ids := uniqueUintIDs(input.IDs)
	if len(ids) == 0 {
		return BatchAssetResult{}, errors.New("请至少选择一台服务器")
	}
	count := 0
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			var asset Asset
			if err := tx.First(&asset, id).Error; err != nil {
				return err
			}
			if err := tx.Where("asset_id = ?", id).Delete(&Account{}).Error; err != nil {
				return err
			}
			if err := tx.Where("asset_id = ?", id).Delete(&SessionEvent{}).Error; err != nil {
				return err
			}
			if err := tx.Where("asset_id = ?", id).Delete(&Session{}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&asset).Error; err != nil {
				return err
			}
			count++
		}
		return nil
	})
	if err != nil {
		return BatchAssetResult{}, err
	}
	return BatchAssetResult{Count: count}, nil
}

func (s *Service) SetDefaultAccount(assetID, accountID uint) (AccountView, error) {
	var account Account
	if err := s.db.Where("asset_id = ? AND id = ?", assetID, accountID).First(&account).Error; err != nil {
		return AccountView{}, err
	}
	if err := s.db.Model(&Account{}).Where("asset_id = ?", assetID).Update("is_default", false).Error; err != nil {
		return AccountView{}, err
	}
	if err := s.db.Model(&Account{}).Where("id = ?", account.ID).Update("is_default", true).Error; err != nil {
		return AccountView{}, err
	}
	if err := s.db.Model(&Asset{}).Where("id = ?", assetID).Update("account", account.Username).Error; err != nil {
		return AccountView{}, err
	}
	s.recordSessionEvent(assetID, 0, "account_default", "info", "设置默认托管账号", "默认登录账号已切换为 "+account.Username+"。")
	account.IsDefault = true
	return accountToView(account), nil
}

func (s *Service) QuickConnect(input QuickConnectInput) (QuickConnectResult, error) {
	var asset Asset
	var assetView *AssetView

	if input.AssetID > 0 {
		if err := s.db.First(&asset, input.AssetID).Error; err != nil {
			return QuickConnectResult{}, err
		}
	} else {
		address := strings.TrimSpace(input.Address)
		if address == "" {
			return QuickConnectResult{}, errors.New("机器地址不能为空")
		}
		name := strings.TrimSpace(input.Name)
		if name == "" {
			name = address
		}
		asset = Asset{
			Name:        name,
			Address:     address,
			Platform:    normalizeField(input.Platform, "linux"),
			Protocol:    normalizeField(input.Protocol, "ssh"),
			GroupName:   strings.TrimSpace(input.GroupName),
			LoginPolicy: normalizeLoginPolicy(input.LoginPolicy),
			Port:        defaultPort(normalizeField(input.Protocol, "ssh")),
			Account:     strings.TrimSpace(input.Account),
			Status:      "online",
			Description: strings.TrimSpace(input.Description),
		}
		if input.Port > 0 {
			asset.Port = input.Port
		}
		if input.SaveToAssets || input.AssetID == 0 {
			now := time.Now()
			asset.LastSeenAt = &now
			if err := s.db.Create(&asset).Error; err != nil {
				return QuickConnectResult{}, err
			}
			view := assetToView(asset)
			assetView = &view
		}
	}

	if asset.ID > 0 {
		now := time.Now()
		asset.LastSeenAt = &now
		asset.Status = "online"
		if err := s.db.Model(&asset).Updates(map[string]any{
			"last_seen_at": asset.LastSeenAt,
			"status":       asset.Status,
			"account":      asset.Account,
		}).Error; err != nil {
			return QuickConnectResult{}, err
		}
		view := assetToView(asset)
		assetView = &view
	}

	session := Session{
		AssetName: pickAssetName(asset, input),
		Address:   pickAddress(asset, input),
		Protocol:  normalizeField(pickProtocol(asset, input), "ssh"),
		Account:   strings.TrimSpace(pickAccount(asset, input)),
		Status:    "prepared",
		Detail:    "连接入口已准备，SSH/RDP 在线代理会在下一阶段接入。",
	}
	if asset.ID > 0 {
		session.AssetID = &asset.ID
	}
	if err := s.db.Create(&session).Error; err != nil {
		return QuickConnectResult{}, err
	}
	s.recordSessionEvent(asset.ID, session.ID, "quick_connect", "info", "创建快速登录入口", "快速登录入口已准备，等待后续进入终端或接入代理。")

	return QuickConnectResult{
		Asset:   assetView,
		Session: sessionToView(session),
		Message: "快速登录入口已准备，当前阶段先完成资产和会话工作台。",
	}, nil
}

func (s *Service) CreateTerminalTicket(id uint, input TerminalTicketInput) (TerminalTicketResult, error) {
	if s.cache == nil {
		return TerminalTicketResult{}, errors.New("terminal cache is not configured")
	}
	var asset Asset
	if err := s.db.First(&asset, id).Error; err != nil {
		return TerminalTicketResult{}, err
	}
	if strings.ToLower(asset.Protocol) != "ssh" {
		return TerminalTicketResult{}, errors.New("当前阶段仅支持 SSH 资产进入在线终端")
	}
	username := strings.TrimSpace(input.Username)
	password := strings.TrimSpace(input.Password)
	privateKey := strings.TrimSpace(input.PrivateKey)
	passphrase := strings.TrimSpace(input.Passphrase)
	authType := "password"
	if input.AccountID > 0 {
		var account Account
		if err := s.db.Where("asset_id = ? AND id = ?", id, input.AccountID).First(&account).Error; err != nil {
			return TerminalTicketResult{}, err
		}
		username = account.Username
		authType = account.AuthType
		switch account.AuthType {
		case "password":
			decryptedPassword, err := s.cipher.Decrypt(account.PasswordEncrypted)
			if err != nil {
				return TerminalTicketResult{}, err
			}
			password = decryptedPassword
		case "ssh_key":
			decryptedKey, err := s.cipher.Decrypt(account.PrivateKeyEncrypted)
			if err != nil {
				return TerminalTicketResult{}, err
			}
			privateKey = decryptedKey
			if strings.TrimSpace(account.PassphraseEncrypted) != "" {
				decryptedPassphrase, err := s.cipher.Decrypt(account.PassphraseEncrypted)
				if err != nil {
					return TerminalTicketResult{}, err
				}
				passphrase = decryptedPassphrase
			}
		default:
			return TerminalTicketResult{}, errors.New("不支持的账号认证方式")
		}
	} else {
		if username == "" {
			username = strings.TrimSpace(asset.Account)
		}
		if username == "" {
			return TerminalTicketResult{}, errors.New("请输入登录账号")
		}
		authType = normalizeField(input.AuthType, "password")
		if authType == "password" && password == "" {
			return TerminalTicketResult{}, errors.New("请输入登录密码")
		}
		if authType == "ssh_key" && privateKey == "" {
			return TerminalTicketResult{}, errors.New("请输入 SSH 私钥")
		}
	}

	session := Session{
		AssetID:   &asset.ID,
		AssetName: asset.Name,
		Address:   asset.Address,
		Protocol:  asset.Protocol,
		Account:   username,
		Status:    "prepared",
		Detail:    "SSH 在线终端已准备，等待建立连接。",
	}
	if err := s.db.Create(&session).Error; err != nil {
		return TerminalTicketResult{}, err
	}
	s.recordSessionEvent(asset.ID, session.ID, "ticket_issued", "info", "生成 SSH 终端票据", "SSH 在线终端票据已生成，等待浏览器建立连接。")

	ticket := generateTicketID()
	payload := terminalTicketPayload{
		AssetID:    asset.ID,
		SessionID:  session.ID,
		Address:    asset.Address,
		Port:       asset.Port,
		Protocol:   asset.Protocol,
		Username:   username,
		AuthType:   authType,
		Password:   password,
		PrivateKey: privateKey,
		Passphrase: passphrase,
		AccessMode: normalizeAssetAccessMode(asset.AccessMode),
		IssuedAt:   time.Now(),
	}
	if payload.AccessMode == "via_gateway" && asset.GatewayAssetID != nil {
		gatewayPayload, err := s.buildGatewayTerminalPayload(asset, *asset.GatewayAssetID)
		if err != nil {
			return TerminalTicketResult{}, err
		}
		payload.GatewayAssetID = gatewayPayload.AssetID
		payload.GatewayName = gatewayPayload.Name
		payload.GatewayAddress = gatewayPayload.Address
		payload.GatewayPort = gatewayPayload.Port
		payload.GatewayUsername = gatewayPayload.Username
		payload.GatewayAuthType = gatewayPayload.AuthType
		payload.GatewayPassword = gatewayPayload.Password
		payload.GatewayPrivateKey = gatewayPayload.PrivateKey
		payload.GatewayPassphrase = gatewayPayload.Passphrase
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return TerminalTicketResult{}, err
	}
	if err := s.cache.Set(context.Background(), machineTerminalTicketKey(ticket), string(raw), 2*time.Minute); err != nil {
		return TerminalTicketResult{}, err
	}
	if err := s.cache.Set(context.Background(), machineSessionAccessKey(session.ID), string(raw), 30*time.Minute); err != nil {
		return TerminalTicketResult{}, err
	}

	return TerminalTicketResult{
		Ticket:       ticket,
		Session:      sessionToView(session),
		Asset:        assetToView(asset),
		TerminalMode: "ssh-websocket",
		Message:      "SSH 终端票据已生成，请在 2 分钟内完成连接。",
	}, nil
}

func (s *Service) buildAccount(assetID uint, input CreateAccountInput) (Account, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" {
		return Account{}, errors.New("登录账号不能为空")
	}
	authType := normalizeField(input.AuthType, "password")
	passwordEncrypted, privateKeyEncrypted, passphraseEncrypted, err := s.encryptCredentialMaterial(authType, input.Password, input.PrivateKey, input.Passphrase)
	if err != nil {
		return Account{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = username
	}
	return Account{
		AssetID:             assetID,
		Name:                name,
		Username:            username,
		AuthType:            authType,
		PasswordEncrypted:   passwordEncrypted,
		PrivateKeyEncrypted: privateKeyEncrypted,
		PassphraseEncrypted: passphraseEncrypted,
		Description:         strings.TrimSpace(input.Description),
		IsDefault:           input.IsDefault,
	}, nil
}

func (s *Service) upsertDefaultAccountTx(tx *gorm.DB, assetID uint, input CreateAccountInput) error {
	var existing Account
	err := tx.Where("asset_id = ? AND is_default = ?", assetID, true).First(&existing).Error
	account, buildErr := s.buildAccount(assetID, input)
	if buildErr != nil {
		return buildErr
	}
	account.IsDefault = true
	if err == nil {
		updates := map[string]any{
			"name":                  account.Name,
			"username":              account.Username,
			"auth_type":             account.AuthType,
			"password_encrypted":    account.PasswordEncrypted,
			"private_key_encrypted": account.PrivateKeyEncrypted,
			"passphrase_encrypted":  account.PassphraseEncrypted,
			"description":           account.Description,
			"is_default":            true,
		}
		return tx.Model(&existing).Updates(updates).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := tx.Model(&Account{}).Where("asset_id = ?", assetID).Update("is_default", false).Error; err != nil {
		return err
	}
	return tx.Create(&account).Error
}

func uniqueUintIDs(ids []uint) []uint {
	seen := map[uint]struct{}{}
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func (s *Service) buildCredentialLibrary(input CreateCredentialInput) (CredentialLibrary, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" {
		return CredentialLibrary{}, errors.New("登录账号不能为空")
	}
	authType := normalizeField(input.AuthType, "password")
	passwordEncrypted, privateKeyEncrypted, passphraseEncrypted, err := s.encryptCredentialMaterial(authType, input.Password, input.PrivateKey, input.Passphrase)
	if err != nil {
		return CredentialLibrary{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = username
	}
	return CredentialLibrary{
		Name:                name,
		Username:            username,
		AuthType:            authType,
		PasswordEncrypted:   passwordEncrypted,
		PrivateKeyEncrypted: privateKeyEncrypted,
		PassphraseEncrypted: passphraseEncrypted,
		SourceType:          strings.TrimSpace(input.SourceType),
		Provider:            strings.TrimSpace(strings.ToLower(input.Provider)),
		ScopeKind:           strings.TrimSpace(input.ScopeKind),
		ScopeRef:            strings.TrimSpace(input.ScopeRef),
		ScopeName:           strings.TrimSpace(input.ScopeName),
		AssetID:             input.AssetID,
		AssetName:           strings.TrimSpace(input.AssetName),
		Description:         strings.TrimSpace(input.Description),
	}, nil
}

func (s *Service) encryptCredentialMaterial(authType, password, privateKey, passphrase string) (string, string, string, error) {
	var passwordEncrypted string
	var privateKeyEncrypted string
	var passphraseEncrypted string
	switch authType {
	case "password":
		password = strings.TrimSpace(password)
		if password == "" {
			return "", "", "", errors.New("登录密码不能为空")
		}
		encrypted, err := s.cipher.Encrypt(password)
		if err != nil {
			return "", "", "", err
		}
		passwordEncrypted = encrypted
	case "ssh_key":
		privateKey = strings.TrimSpace(privateKey)
		if privateKey == "" {
			return "", "", "", errors.New("SSH 私钥不能为空")
		}
		encrypted, err := s.cipher.Encrypt(privateKey)
		if err != nil {
			return "", "", "", err
		}
		privateKeyEncrypted = encrypted
		if strings.TrimSpace(passphrase) != "" {
			encryptedPassphrase, err := s.cipher.Encrypt(strings.TrimSpace(passphrase))
			if err != nil {
				return "", "", "", err
			}
			passphraseEncrypted = encryptedPassphrase
		}
	default:
		return "", "", "", errors.New("不支持的认证方式")
	}
	return passwordEncrypted, privateKeyEncrypted, passphraseEncrypted, nil
}

func (s *Service) consumeTerminalTicket(ticket string) (terminalTicketPayload, error) {
	if s.cache == nil {
		return terminalTicketPayload{}, errors.New("terminal cache is not configured")
	}
	key := machineTerminalTicketKey(ticket)
	raw, ok, err := s.cache.Get(context.Background(), key)
	if err != nil {
		return terminalTicketPayload{}, err
	}
	if !ok {
		return terminalTicketPayload{}, errors.New("terminal ticket not found or expired")
	}
	_ = s.cache.Delete(context.Background(), key)
	var payload terminalTicketPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return terminalTicketPayload{}, err
	}
	return payload, nil
}

func (s *Service) markSessionActive(id uint) {
	var session Session
	if err := s.db.First(&session, id).Error; err != nil {
		return
	}
	_ = s.db.Model(&Session{}).Where("id = ?", id).Updates(map[string]any{
		"status": "active",
		"detail": "SSH 在线终端已连接。",
	}).Error
	s.recordSessionEvent(session.AssetIDValue(), session.ID, "terminal_connected", "info", "终端已连接", "浏览器 SSH 终端已建立连接。")
}

func (s *Service) markSessionClosed(id uint, detail string) {
	var session Session
	if err := s.db.First(&session, id).Error; err != nil {
		return
	}
	now := time.Now()
	_ = s.db.Model(&Session{}).Where("id = ?", id).Updates(map[string]any{
		"status":     "closed",
		"detail":     detail,
		"ended_at":   &now,
		"updated_at": now,
	}).Error
	s.recordSessionEvent(session.AssetIDValue(), session.ID, "terminal_closed", "info", "终端已关闭", detail)
}

func assetToView(asset Asset) AssetView {
	return AssetView{
		ID:                  asset.ID,
		Name:                asset.Name,
		Address:             asset.Address,
		PrivateIP:           asset.PrivateIP,
		Platform:            asset.Platform,
		Protocol:            asset.Protocol,
		ProjectID:           asset.ProjectID,
		EnvironmentID:       asset.EnvironmentID,
		StackID:             asset.StackID,
		FoundationNetworkID: asset.FoundationNetworkID,
		AccessMode:          normalizeAssetAccessMode(asset.AccessMode),
		GatewayAssetID:      asset.GatewayAssetID,
		IsGatewayNode:       asset.IsGatewayNode,
		SourceType:          asset.SourceType,
		SourceProvider:      asset.SourceProvider,
		SourceResourceID:    asset.SourceResourceID,
		SourceJobID:         asset.SourceJobID,
		EnrollmentStatus:    asset.EnrollmentStatus,
		EnrollmentError:     asset.EnrollmentError,
		GroupName:           asset.GroupName,
		LoginPolicy:         normalizeLoginPolicy(asset.LoginPolicy),
		Port:                asset.Port,
		Account:             asset.Account,
		Status:              asset.Status,
		Tags:                splitTags(asset.Tags),
		Description:         asset.Description,
		LastSeenAt:          asset.LastSeenAt,
		CreatedAt:           asset.CreatedAt,
		UpdatedAt:           asset.UpdatedAt,
	}
}

func (s *Service) enrichAssetViews(views []AssetView) error {
	if len(views) == 0 {
		return nil
	}
	assetIDs := make([]uint, 0, len(views))
	indexByID := make(map[uint]int, len(views))
	for index, view := range views {
		assetIDs = append(assetIDs, view.ID)
		indexByID[view.ID] = index
	}
	var accounts []Account
	if err := s.db.Where("asset_id IN ?", assetIDs).Order("is_default desc, updated_at desc").Find(&accounts).Error; err != nil {
		return err
	}
	accountCount := make(map[uint]int, len(assetIDs))
	for _, account := range accounts {
		accountCount[account.AssetID]++
		index, ok := indexByID[account.AssetID]
		if !ok {
			continue
		}
		views[index].HasAccounts = true
		if account.IsDefault && views[index].DefaultAccountID == nil {
			accountID := account.ID
			views[index].DefaultAccountID = &accountID
			views[index].DefaultAccountName = account.Name
			views[index].DefaultAccountUsername = account.Username
		}
	}
	for assetID, index := range indexByID {
		views[index].AccountCount = accountCount[assetID]
	}
	gatewayIDs := make([]uint, 0)
	for _, view := range views {
		if view.GatewayAssetID != nil && *view.GatewayAssetID > 0 {
			gatewayIDs = append(gatewayIDs, *view.GatewayAssetID)
		}
	}
	gatewayIDs = uniqueUintIDs(gatewayIDs)
	if len(gatewayIDs) == 0 {
		return nil
	}
	var gatewayAssets []Asset
	if err := s.db.Where("id IN ?", gatewayIDs).Find(&gatewayAssets).Error; err != nil {
		return err
	}
	gatewayByID := make(map[uint]Asset, len(gatewayAssets))
	for _, asset := range gatewayAssets {
		gatewayByID[asset.ID] = asset
	}
	for index := range views {
		if views[index].GatewayAssetID == nil {
			continue
		}
		gateway, ok := gatewayByID[*views[index].GatewayAssetID]
		if !ok {
			continue
		}
		views[index].GatewayAssetName = gateway.Name
		views[index].GatewayAddress = gateway.Address
	}
	return nil
}

type gatewayTerminalPayload struct {
	AssetID    uint
	Name       string
	Address    string
	Port       int
	Username   string
	AuthType   string
	Password   string
	PrivateKey string
	Passphrase string
}

func (s *Service) buildGatewayTerminalPayload(asset Asset, gatewayAssetID uint) (gatewayTerminalPayload, error) {
	var gateway Asset
	if err := s.db.First(&gateway, gatewayAssetID).Error; err != nil {
		return gatewayTerminalPayload{}, errors.New("跳板机不存在")
	}
	if strings.ToLower(gateway.Protocol) != "ssh" {
		return gatewayTerminalPayload{}, errors.New("跳板机必须是 SSH 资产")
	}
	var account Account
	if err := s.db.Where("asset_id = ? AND is_default = ?", gateway.ID, true).First(&account).Error; err != nil {
		return gatewayTerminalPayload{}, errors.New("跳板机缺少默认托管账号")
	}
	password, privateKey, passphrase, err := s.decryptAccountMaterial(account)
	if err != nil {
		return gatewayTerminalPayload{}, err
	}
	return gatewayTerminalPayload{
		AssetID:    gateway.ID,
		Name:       gateway.Name,
		Address:    gateway.Address,
		Port:       gateway.Port,
		Username:   account.Username,
		AuthType:   account.AuthType,
		Password:   password,
		PrivateKey: privateKey,
		Passphrase: passphrase,
	}, nil
}

func (s *Service) decryptAccountMaterial(account Account) (string, string, string, error) {
	switch account.AuthType {
	case "password":
		password, err := s.cipher.Decrypt(account.PasswordEncrypted)
		return password, "", "", err
	case "ssh_key":
		privateKey, err := s.cipher.Decrypt(account.PrivateKeyEncrypted)
		if err != nil {
			return "", "", "", err
		}
		passphrase := ""
		if strings.TrimSpace(account.PassphraseEncrypted) != "" {
			passphrase, err = s.cipher.Decrypt(account.PassphraseEncrypted)
			if err != nil {
				return "", "", "", err
			}
		}
		return "", privateKey, passphrase, nil
	default:
		return "", "", "", errors.New("不支持的跳板机认证方式")
	}
}

func (s *Service) validateAssetGateway(assetID uint, accessMode string, gatewayAssetID *uint, protocol string) error {
	if normalizeAssetAccessMode(accessMode) != "via_gateway" {
		return nil
	}
	if strings.ToLower(strings.TrimSpace(protocol)) != "ssh" {
		return errors.New("当前仅 SSH 资产支持通过跳板机访问")
	}
	if gatewayAssetID == nil || *gatewayAssetID == 0 {
		return errors.New("通过跳板机访问时，必须选择入口节点")
	}
	if assetID > 0 && *gatewayAssetID == assetID {
		return errors.New("入口节点不能指向自己")
	}
	var gateway Asset
	if err := s.db.First(&gateway, *gatewayAssetID).Error; err != nil {
		return errors.New("入口节点不存在")
	}
	if strings.ToLower(gateway.Protocol) != "ssh" {
		return errors.New("入口节点必须是 SSH 资产")
	}
	return nil
}

func normalizeAssetAccessMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "via_gateway":
		return "via_gateway"
	default:
		return "direct"
	}
}

func sessionToView(session Session) SessionView {
	return SessionView{
		ID:        session.ID,
		AssetID:   session.AssetID,
		AssetName: session.AssetName,
		Address:   session.Address,
		Protocol:  session.Protocol,
		Account:   session.Account,
		Status:    session.Status,
		Detail:    session.Detail,
		StartedAt: session.StartedAt,
		EndedAt:   session.EndedAt,
	}
}

func sessionEventToView(event SessionEvent) SessionEventView {
	return SessionEventView{
		ID:         event.ID,
		AssetID:    event.AssetID,
		SessionID:  event.SessionID,
		AssetName:  event.AssetName,
		EventType:  event.EventType,
		EventLevel: event.EventLevel,
		Summary:    event.Summary,
		Detail:     event.Detail,
		CreatedAt:  event.CreatedAt,
	}
}

func accountToView(account Account) AccountView {
	return AccountView{
		ID:          account.ID,
		AssetID:     account.AssetID,
		Name:        account.Name,
		Username:    account.Username,
		AuthType:    account.AuthType,
		Description: account.Description,
		IsDefault:   account.IsDefault,
		CreatedAt:   account.CreatedAt,
		UpdatedAt:   account.UpdatedAt,
	}
}

func credentialToView(credential CredentialLibrary) CredentialView {
	return CredentialView{
		ID:          credential.ID,
		Name:        credential.Name,
		Username:    credential.Username,
		AuthType:    credential.AuthType,
		SourceType:  credential.SourceType,
		Provider:    credential.Provider,
		ScopeKind:   credential.ScopeKind,
		ScopeRef:    credential.ScopeRef,
		ScopeName:   credential.ScopeName,
		AssetID:     credential.AssetID,
		AssetName:   credential.AssetName,
		Description: credential.Description,
		CreatedAt:   credential.CreatedAt,
		UpdatedAt:   credential.UpdatedAt,
	}
}

func quickCommandToView(command QuickCommand) QuickCommandView {
	return QuickCommandView{
		ID:          command.ID,
		Name:        command.Name,
		Kind:        command.Kind,
		Content:     command.Content,
		Description: command.Description,
		CreatedAt:   command.CreatedAt,
		UpdatedAt:   command.UpdatedAt,
	}
}

func (s *Service) credentialToDetailView(credential CredentialLibrary) (CredentialDetailView, error) {
	password, err := s.decryptOptional(credential.PasswordEncrypted)
	if err != nil {
		return CredentialDetailView{}, err
	}
	privateKey, err := s.decryptOptional(credential.PrivateKeyEncrypted)
	if err != nil {
		return CredentialDetailView{}, err
	}
	passphrase, err := s.decryptOptional(credential.PassphraseEncrypted)
	if err != nil {
		return CredentialDetailView{}, err
	}
	return CredentialDetailView{
		ID:          credential.ID,
		Name:        credential.Name,
		Username:    credential.Username,
		AuthType:    credential.AuthType,
		SourceType:  credential.SourceType,
		Provider:    credential.Provider,
		ScopeKind:   credential.ScopeKind,
		ScopeRef:    credential.ScopeRef,
		ScopeName:   credential.ScopeName,
		AssetID:     credential.AssetID,
		AssetName:   credential.AssetName,
		Password:    password,
		PrivateKey:  privateKey,
		Passphrase:  passphrase,
		Description: credential.Description,
		CreatedAt:   credential.CreatedAt,
		UpdatedAt:   credential.UpdatedAt,
	}, nil
}

func (s *Service) accountToDetailView(account Account) (AccountDetailView, error) {
	password, err := s.decryptOptional(account.PasswordEncrypted)
	if err != nil {
		return AccountDetailView{}, err
	}
	privateKey, err := s.decryptOptional(account.PrivateKeyEncrypted)
	if err != nil {
		return AccountDetailView{}, err
	}
	passphrase, err := s.decryptOptional(account.PassphraseEncrypted)
	if err != nil {
		return AccountDetailView{}, err
	}
	return AccountDetailView{
		ID:          account.ID,
		AssetID:     account.AssetID,
		Name:        account.Name,
		Username:    account.Username,
		AuthType:    account.AuthType,
		Password:    password,
		PrivateKey:  privateKey,
		Passphrase:  passphrase,
		Description: account.Description,
		IsDefault:   account.IsDefault,
		CreatedAt:   account.CreatedAt,
		UpdatedAt:   account.UpdatedAt,
	}, nil
}

func (s *Service) decryptOptional(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	return s.cipher.Decrypt(value)
}

func (s *Service) assetGroupToView(group AssetGroup, depth int) AssetGroupView {
	var assetCount int64
	_ = s.db.Model(&Asset{}).Where("group_name = ?", group.Name).Count(&assetCount).Error
	return AssetGroupView{
		ID:                 group.ID,
		Name:               group.Name,
		Code:               group.Code,
		ParentID:           group.ParentID,
		DefaultLoginPolicy: normalizeLoginPolicy(group.DefaultLoginPolicy),
		Description:        group.Description,
		AssetCount:         assetCount,
		Depth:              depth,
		CreatedAt:          group.CreatedAt,
		UpdatedAt:          group.UpdatedAt,
	}
}

func (s *Service) buildAssetGroupTree(groups []AssetGroup) []AssetGroupView {
	if len(groups) == 0 {
		return nil
	}
	childrenByParent := make(map[uint][]AssetGroup)
	roots := make([]AssetGroup, 0)
	for _, group := range groups {
		if group.ParentID == nil {
			roots = append(roots, group)
			continue
		}
		childrenByParent[*group.ParentID] = append(childrenByParent[*group.ParentID], group)
	}
	var walk func(items []AssetGroup, depth int) []AssetGroupView
	walk = func(items []AssetGroup, depth int) []AssetGroupView {
		result := make([]AssetGroupView, 0, len(items))
		for _, group := range items {
			view := s.assetGroupToView(group, depth)
			if children := childrenByParent[group.ID]; len(children) > 0 {
				view.Children = walk(children, depth+1)
			}
			result = append(result, view)
		}
		return result
	}
	tree := walk(roots, 0)
	var accumulate func(item *AssetGroupView) int64
	accumulate = func(item *AssetGroupView) int64 {
		total := item.AssetCount
		for index := range item.Children {
			total += accumulate(&item.Children[index])
		}
		item.AssetCount = total
		return total
	}
	for index := range tree {
		accumulate(&tree[index])
	}
	return tree
}

func (s *Service) validateAssetGroupParent(groupID uint, parentID *uint) error {
	if parentID == nil {
		return nil
	}
	if groupID != 0 && *parentID == groupID {
		return errors.New("资产组不能挂到自己下面")
	}
	var parent AssetGroup
	if err := s.db.First(&parent, *parentID).Error; err != nil {
		return errors.New("上级资产组不存在")
	}
	if groupID == 0 {
		return nil
	}
	var groups []AssetGroup
	if err := s.db.Find(&groups).Error; err != nil {
		return err
	}
	parentMap := make(map[uint]*uint, len(groups))
	for _, group := range groups {
		parentMap[group.ID] = group.ParentID
	}
	for current := parent.ParentID; current != nil; {
		if *current == groupID {
			return errors.New("不能把资产组移动到自己的下级节点")
		}
		next, ok := parentMap[*current]
		if !ok {
			break
		}
		current = next
	}
	return nil
}

func (s *Service) expandAssetGroupNames(groupName string) ([]string, error) {
	target := strings.TrimSpace(groupName)
	if target == "" {
		return nil, nil
	}
	var groups []AssetGroup
	if err := s.db.Order("name asc").Find(&groups).Error; err != nil {
		return nil, err
	}
	groupByName := make(map[string]AssetGroup, len(groups))
	childrenByParent := make(map[uint][]AssetGroup)
	for _, group := range groups {
		groupByName[group.Name] = group
		if group.ParentID != nil {
			childrenByParent[*group.ParentID] = append(childrenByParent[*group.ParentID], group)
		}
	}
	root, ok := groupByName[target]
	if !ok {
		return []string{target}, nil
	}
	names := []string{root.Name}
	queue := []AssetGroup{root}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, child := range childrenByParent[current.ID] {
			names = append(names, child.Name)
			queue = append(queue, child)
		}
	}
	return names, nil
}

func normalizeField(value string, fallback string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func normalizeLoginPolicy(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "inherit_group":
		return "inherit_group"
	case "manual_only":
		return "manual_only"
	case "managed_only":
		return "managed_only"
	default:
		return "managed_first"
	}
}

func normalizeTags(value string) string {
	parts := splitTags(value)
	return strings.Join(parts, ",")
}

func splitTags(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	raw := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	})
	tags := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		tags = append(tags, trimmed)
	}
	return tags
}

func defaultPort(protocol string) int {
	switch strings.ToLower(protocol) {
	case "rdp":
		return 3389
	case "vnc":
		return 5900
	case "mysql":
		return 3306
	default:
		return 22
	}
}

func machineTerminalTicketKey(ticket string) string {
	return "machine:terminal:ticket:" + ticket
}

func machineSessionAccessKey(sessionID uint) string {
	return "machine:terminal:session:" + strconv.FormatUint(uint64(sessionID), 10)
}

func generateTicketID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(buf)
}

func pickAssetName(asset Asset, input QuickConnectInput) string {
	if asset.Name != "" {
		return asset.Name
	}
	if strings.TrimSpace(input.Name) != "" {
		return strings.TrimSpace(input.Name)
	}
	return strings.TrimSpace(input.Address)
}

func pickAddress(asset Asset, input QuickConnectInput) string {
	if asset.Address != "" {
		return asset.Address
	}
	return strings.TrimSpace(input.Address)
}

func pickProtocol(asset Asset, input QuickConnectInput) string {
	if asset.Protocol != "" {
		return asset.Protocol
	}
	return input.Protocol
}

func pickAccount(asset Asset, input QuickConnectInput) string {
	if asset.Account != "" {
		return asset.Account
	}
	return input.Account
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func slugifyMachineValue(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "asset-group"
	}
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", ".", "-", ":", "-", "@", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return "asset-group"
	}
	return value
}

func (s *Service) applyFallbackDefaultAccount(assetID uint) error {
	var next Account
	err := s.db.Where("asset_id = ?", assetID).Order("updated_at desc").First(&next).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.db.Model(&Asset{}).Where("id = ?", assetID).Update("account", "").Error
		}
		return err
	}
	if err := s.db.Model(&Account{}).Where("id = ?", next.ID).Update("is_default", true).Error; err != nil {
		return err
	}
	return s.db.Model(&Asset{}).Where("id = ?", assetID).Update("account", next.Username).Error
}

func (s *Service) recordSessionEvent(assetID uint, sessionID uint, eventType string, eventLevel string, summary string, detail string) {
	if assetID == 0 || strings.TrimSpace(summary) == "" {
		return
	}
	var asset Asset
	if err := s.db.Select("id", "name").First(&asset, assetID).Error; err != nil {
		return
	}
	event := SessionEvent{
		AssetID:    assetID,
		AssetName:  asset.Name,
		EventType:  normalizeField(eventType, "event"),
		EventLevel: normalizeField(eventLevel, "info"),
		Summary:    strings.TrimSpace(summary),
		Detail:     strings.TrimSpace(detail),
	}
	if sessionID > 0 {
		event.SessionID = &sessionID
	}
	_ = s.db.Create(&event).Error
}

func (s Session) AssetIDValue() uint {
	if s.AssetID == nil {
		return 0
	}
	return *s.AssetID
}
