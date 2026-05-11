package user

import (
	"errors"
	"fmt"
	"time"

	"backend-center/internal/infra/security"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type CreateLocalUserInput struct {
	Username    string `json:"username" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Email       string `json:"email"`
	Password    string `json:"password" binding:"required,min=8"`
	MFARequired bool   `json:"mfa_required"`
}

type UpdateUserInput struct {
	DisplayName string `json:"display_name" binding:"required"`
	Email       string `json:"email"`
	Status      string `json:"status"`
	MFARequired bool   `json:"mfa_required"`
	MFAEnabled  bool   `json:"mfa_enabled"`
}

type RoleSummary struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type ListItem struct {
	User
	Roles  []RoleSummary  `json:"roles"`
	Groups []GroupSummary `json:"groups"`
}

type GroupSummary struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

func (s *Service) List() ([]ListItem, error) {
	var users []User
	err := s.db.Order("id asc").Find(&users).Error
	if err != nil {
		return nil, err
	}

	items := make([]ListItem, 0, len(users))
	for _, item := range users {
		roles, err := s.rolesForUser(item.ID)
		if err != nil {
			return nil, err
		}
		groups, err := s.groupsForUser(item.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, ListItem{
			User:   item,
			Roles:  roles,
			Groups: groups,
		})
	}

	return items, nil
}

func (s *Service) CreateLocal(input CreateLocalUserInput) (*User, error) {
	hash, err := security.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	model := &User{
		Username:    input.Username,
		DisplayName: input.DisplayName,
		Email:       input.Email,
		Status:      "active",
		SourceType:  SourceLocal,
		IsExternal:  false,
		MFARequired: input.MFARequired,
		MFAEnabled:  false,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(model).Error; err != nil {
			return err
		}
		return tx.Create(&Credential{
			UserID:       model.ID,
			PasswordHash: hash,
		}).Error
	})
	return model, err
}

func (s *Service) FindByUsername(username string) (*User, error) {
	var model User
	if err := s.db.Where("username = ?", username).First(&model).Error; err != nil {
		return nil, err
	}
	return &model, nil
}

func (s *Service) VerifyLocalPassword(userID uint, password string) error {
	var credential Credential
	if err := s.db.Where("user_id = ?", userID).First(&credential).Error; err != nil {
		return err
	}
	return security.CheckPassword(credential.PasswordHash, password)
}

func (s *Service) TouchLogin(userID uint) error {
	return s.db.Model(&User{}).Where("id = ?", userID).Update("last_login_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
}

func (s *Service) Disable(userID uint) error {
	result := s.db.Model(&User{}).Where("id = ?", userID).Update("status", "disabled")
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return result.Error
}

func (s *Service) Get(userID uint) (*ListItem, error) {
	var model User
	if err := s.db.First(&model, userID).Error; err != nil {
		return nil, err
	}
	roles, err := s.rolesForUser(model.ID)
	if err != nil {
		return nil, err
	}
	groups, err := s.groupsForUser(model.ID)
	if err != nil {
		return nil, err
	}
	return &ListItem{User: model, Roles: roles, Groups: groups}, nil
}

func (s *Service) Update(userID uint, input UpdateUserInput) (*ListItem, error) {
	status := input.Status
	if status == "" {
		status = "active"
	}
	err := s.db.Model(&User{}).Where("id = ?", userID).Updates(map[string]any{
		"display_name": input.DisplayName,
		"email":        input.Email,
		"status":       status,
		"mfa_required": input.MFARequired,
		"mfa_enabled":  input.MFAEnabled,
	}).Error
	if err != nil {
		return nil, err
	}
	return s.Get(userID)
}

func (s *Service) ResetPassword(userID uint, password string) error {
	hash, err := security.HashPassword(password)
	if err != nil {
		return err
	}
	credential := Credential{UserID: userID}
	return s.db.Where(Credential{UserID: userID}).Assign(Credential{
		UserID:       userID,
		PasswordHash: hash,
	}).FirstOrCreate(&credential).Error
}

func (s *Service) Delete(userID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&Credential{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&ExternalIdentity{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&GroupMember{}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM subject_role_bindings WHERE subject_type = ? AND subject_id = ?", "user", fmt.Sprintf("%d", userID)).Error; err != nil {
			return err
		}
		return tx.Delete(&User{}, userID).Error
	})
}

func (s *Service) rolesForUser(userID uint) ([]RoleSummary, error) {
	var roles []RoleSummary
	err := s.db.Table("roles").
		Distinct("roles.id, roles.name, roles.code").
		Joins("join subject_role_bindings on subject_role_bindings.role_id = roles.id").
		Where("subject_role_bindings.subject_type = ? and subject_role_bindings.subject_id = ?", "user", fmt.Sprintf("%d", userID)).
		Order("roles.id desc").
		Scan(&roles).Error
	return roles, err
}

func (s *Service) groupsForUser(userID uint) ([]GroupSummary, error) {
	var groups []GroupSummary
	err := s.db.Table("groups").
		Distinct("groups.id, groups.name, groups.code").
		Joins("join group_members on group_members.group_id = groups.id").
		Where("group_members.user_id = ?", userID).
		Order("groups.id desc").
		Scan(&groups).Error
	return groups, err
}

type CreateGroupInput struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}

type UpdateGroupInput struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
}

type GroupMemberItem struct {
	ID          uint      `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	JoinedAt    time.Time `json:"joined_at"`
}

type GroupListItem struct {
	Group
	Members []GroupMemberItem `json:"members"`
}

func (s *Service) ListGroups() ([]GroupListItem, error) {
	var groups []Group
	if err := s.db.Order("id asc").Find(&groups).Error; err != nil {
		return nil, err
	}
	items := make([]GroupListItem, 0, len(groups))
	for _, group := range groups {
		members, err := s.membersForGroup(group.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, GroupListItem{Group: group, Members: members})
	}
	return items, nil
}

func (s *Service) CreateGroup(input CreateGroupInput) (*Group, error) {
	model := &Group{Name: input.Name, Code: input.Code, Description: input.Description}
	return model, s.db.Create(model).Error
}

func (s *Service) UpdateGroup(groupID uint, input UpdateGroupInput) (*Group, error) {
	if err := s.db.Model(&Group{}).Where("id = ?", groupID).Updates(map[string]any{
		"name": input.Name, "code": input.Code, "description": input.Description,
	}).Error; err != nil {
		return nil, err
	}
	var model Group
	if err := s.db.First(&model, groupID).Error; err != nil {
		return nil, err
	}
	return &model, nil
}

func (s *Service) DeleteGroup(groupID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_id = ?", groupID).Delete(&GroupMember{}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM subject_role_bindings WHERE subject_type = ? AND subject_id = ?", "group", fmt.Sprintf("%d", groupID)).Error; err != nil {
			return err
		}
		return tx.Delete(&Group{}, groupID).Error
	})
}

func (s *Service) AddGroupMember(groupID, userID uint) error {
	member := GroupMember{GroupID: groupID, UserID: userID}
	return s.db.Where(GroupMember{GroupID: groupID, UserID: userID}).FirstOrCreate(&member).Error
}

func (s *Service) RemoveGroupMember(groupID, userID uint) error {
	return s.db.Where("group_id = ? and user_id = ?", groupID, userID).Delete(&GroupMember{}).Error
}

func (s *Service) membersForGroup(groupID uint) ([]GroupMemberItem, error) {
	var items []GroupMemberItem
	err := s.db.Table("group_members").
		Select("users.id, users.username, users.display_name, users.email, group_members.created_at as joined_at").
		Joins("join users on users.id = group_members.user_id").
		Where("group_members.group_id = ?", groupID).
		Order("group_members.id desc").
		Scan(&items).Error
	return items, err
}
