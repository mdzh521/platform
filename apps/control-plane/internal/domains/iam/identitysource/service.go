package identitysource

import (
	"crypto/tls"
	"fmt"
	"time"

	ldap "github.com/go-ldap/ldap/v3"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type ListItem struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Enabled     bool      `json:"enabled"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	BaseDN      string    `json:"base_dn"`
	BindDN      string    `json:"bind_dn"`
	UserFilter  string    `json:"user_filter"`
	GroupFilter string    `json:"group_filter"`
	ClientID    string    `json:"client_id"`
	IssuerURL   string    `json:"issuer_url"`
	RedirectURL string    `json:"redirect_url"`
	SyncMode    string    `json:"sync_mode"`
	HasSecret   bool      `json:"has_secret"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (s *Service) List() ([]ListItem, error) {
	var items []IdentitySource
	err := s.db.Order("id asc").Find(&items).Error
	if err != nil {
		return nil, err
	}
	result := make([]ListItem, 0, len(items))
	for _, item := range items {
		result = append(result, sanitize(item))
	}
	return result, nil
}

func (s *Service) Create(input IdentitySource) (*IdentitySource, error) {
	if input.SyncMode == "" {
		input.SyncMode = "login_only"
	}
	if err := s.db.Create(&input).Error; err != nil {
		return nil, err
	}
	return &input, nil
}

func (s *Service) Update(id uint, input IdentitySource) (*ListItem, error) {
	var current IdentitySource
	if err := s.db.First(&current, id).Error; err != nil {
		return nil, err
	}
	if input.BindPassword == "" {
		input.BindPassword = current.BindPassword
	}
	if input.ClientSecret == "" {
		input.ClientSecret = current.ClientSecret
	}
	input.ID = current.ID
	if err := s.db.Model(&current).Updates(input).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&current, id).Error; err != nil {
		return nil, err
	}
	item := sanitize(current)
	return &item, nil
}

func (s *Service) Delete(id uint) error {
	return s.db.Delete(&IdentitySource{}, id).Error
}

func (s *Service) Test(id uint) (map[string]any, error) {
	var source IdentitySource
	if err := s.db.First(&source, id).Error; err != nil {
		return nil, err
	}

	if source.Type != "ldap" {
		return map[string]any{
			"source_id": id,
			"result":    "saved",
			"message":   "non-LDAP source stored successfully; runtime handshake will be implemented in the next stage",
		}, nil
	}

	address := fmt.Sprintf("%s:%d", source.Host, source.Port)
	conn, err := ldap.DialURL("ldap://" + address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_ = conn.StartTLS(&tls.Config{InsecureSkipVerify: true})
	conn.SetTimeout(5 * time.Second)
	if source.BindDN != "" {
		if err := conn.Bind(source.BindDN, source.BindPassword); err != nil {
			return nil, err
		}
	}

	return map[string]any{
		"source_id": id,
		"result":    "ok",
		"message":   "ldap connection succeeded",
	}, nil
}

func sanitize(item IdentitySource) ListItem {
	return ListItem{
		ID:          item.ID,
		Name:        item.Name,
		Type:        item.Type,
		Enabled:     item.Enabled,
		Host:        item.Host,
		Port:        item.Port,
		BaseDN:      item.BaseDN,
		BindDN:      item.BindDN,
		UserFilter:  item.UserFilter,
		GroupFilter: item.GroupFilter,
		ClientID:    item.ClientID,
		IssuerURL:   item.IssuerURL,
		RedirectURL: item.RedirectURL,
		SyncMode:    item.SyncMode,
		HasSecret:   item.BindPassword != "" || item.ClientSecret != "",
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}
