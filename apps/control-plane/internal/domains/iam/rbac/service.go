package rbac

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type RolePermissionSummary struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

type RoleListItem struct {
	Role
	Permissions []RolePermissionSummary `json:"permissions"`
}

func (s *Service) ListRoles() ([]RoleListItem, error) {
	var roles []Role
	if err := s.db.Order("id asc").Find(&roles).Error; err != nil {
		return nil, err
	}
	items := make([]RoleListItem, 0, len(roles))
	for _, role := range roles {
		permissions, err := s.permissionsForRole(role.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, RoleListItem{
			Role:        role,
			Permissions: permissions,
		})
	}
	return items, nil
}

func (s *Service) CreateRole(input Role) (*Role, error) {
	if err := s.db.Create(&input).Error; err != nil {
		return nil, err
	}
	return &input, nil
}

func (s *Service) UpdateRole(id uint, input Role) (*Role, error) {
	var item Role
	if err := s.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	item.Name = input.Name
	item.Code = input.Code
	item.Description = input.Description
	if err := s.db.Save(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) DeleteRole(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&RolePermission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&SubjectRoleBinding{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Role{}, id).Error
	})
}

func (s *Service) Assign(binding SubjectRoleBinding) (*SubjectRoleBinding, error) {
	if err := s.db.Where(SubjectRoleBinding{
		SubjectType: binding.SubjectType,
		SubjectID:   binding.SubjectID,
		RoleID:      binding.RoleID,
	}).FirstOrCreate(&binding).Error; err != nil {
		return nil, err
	}
	return &binding, nil
}

func (s *Service) RemoveBinding(id uint) error {
	return s.db.Delete(&SubjectRoleBinding{}, id).Error
}

type BindingListItem struct {
	ID                 uint   `json:"id"`
	SubjectType        string `json:"subject_type"`
	SubjectID          string `json:"subject_id"`
	SubjectName        string `json:"subject_name"`
	SubjectDisplayName string `json:"subject_display_name"`
	RoleID             uint   `json:"role_id"`
	RoleName           string `json:"role_name"`
	RoleCode           string `json:"role_code"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (s *Service) ListBindings() ([]BindingListItem, error) {
	var items []BindingListItem
	err := s.db.Table("subject_role_bindings").
		Select("subject_role_bindings.id, subject_role_bindings.subject_type, subject_role_bindings.subject_id, subject_role_bindings.role_id, subject_role_bindings.created_at, subject_role_bindings.updated_at, roles.name as role_name, roles.code as role_code").
		Joins("join roles on roles.id = subject_role_bindings.role_id").
		Order("subject_role_bindings.id asc").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}

	for i := range items {
		item := &items[i]
		switch item.SubjectType {
		case "user":
			var result struct {
				Username    string
				DisplayName string
			}
			if err := s.db.Table("users").
				Select("username, display_name").
				Where("id = ?", item.SubjectID).
				Scan(&result).Error; err != nil {
				return nil, err
			}
			item.SubjectName = result.Username
			item.SubjectDisplayName = result.DisplayName
		case "group":
			var result struct {
				Name        string
				Description string
			}
			if err := s.db.Table("groups").
				Select("name, description").
				Where("id = ?", item.SubjectID).
				Scan(&result).Error; err != nil {
				return nil, err
			}
			item.SubjectName = result.Name
			item.SubjectDisplayName = result.Description
		default:
			item.SubjectName = item.SubjectID
		}
	}

	return items, nil
}

func (s *Service) PermissionsForUser(userID uint) ([]string, error) {
	var codes []string
	err := s.db.Raw(`
		SELECT DISTINCT permissions.code
		FROM permissions
		JOIN role_permissions ON role_permissions.permission_id = permissions.id
		JOIN subject_role_bindings ON subject_role_bindings.role_id = role_permissions.role_id
		WHERE (subject_role_bindings.subject_type = 'user' AND subject_role_bindings.subject_id = ?)
		   OR (
		       subject_role_bindings.subject_type = 'group'
		       AND subject_role_bindings.subject_id IN (
		           SELECT CAST(group_members.group_id AS CHAR)
		           FROM group_members
		           WHERE group_members.user_id = ?
		       )
		   )
	`, fmt.Sprintf("%d", userID), userID).
		Scan(&codes).Error
	return codes, err
}

func (s *Service) permissionsForRole(roleID uint) ([]RolePermissionSummary, error) {
	var permissions []RolePermissionSummary
	err := s.db.Table("permissions").
		Select("permissions.code, permissions.name, permissions.resource, permissions.action, permissions.description").
		Joins("join role_permissions on role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Order("permissions.resource asc, permissions.action asc, permissions.code asc").
		Scan(&permissions).Error
	return permissions, err
}
