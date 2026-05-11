package seed

import (
	"encoding/json"
	"fmt"
	"strings"

	"backend-center/internal/config"
	"backend-center/internal/domains/delivery/cloud"
	"backend-center/internal/domains/iam/rbac"
	"backend-center/internal/domains/platform/systemsetting"
	"backend-center/internal/domains/iam/user"
	"backend-center/internal/infra/secure"
	"backend-center/internal/infra/security"

	"gorm.io/gorm"
)

func Run(db *gorm.DB, cfg config.Config) error {
	return db.Transaction(func(tx *gorm.DB) error {
		permissions := []rbac.Permission{
			{Code: "user.read", Name: "Read users", Resource: "user", Action: "read"},
			{Code: "user.create", Name: "Create users", Resource: "user", Action: "create"},
			{Code: "user.disable", Name: "Disable users", Resource: "user", Action: "disable"},
			{Code: "identity_source.manage", Name: "Manage identity sources", Resource: "identity_source", Action: "manage"},
			{Code: "rbac.manage", Name: "Manage RBAC", Resource: "rbac", Action: "manage"},
			{Code: "project.read", Name: "Read projects", Resource: "project", Action: "read"},
			{Code: "project.manage", Name: "Manage projects", Resource: "project", Action: "manage"},
			{Code: "stack.read", Name: "Read stacks", Resource: "stack", Action: "read"},
			{Code: "stack.manage", Name: "Manage stacks", Resource: "stack", Action: "manage"},
			{Code: "cluster.read", Name: "Read clusters", Resource: "cluster", Action: "read"},
			{Code: "cluster.manage", Name: "Manage clusters", Resource: "cluster", Action: "manage"},
			{Code: "namespace.read", Name: "Read namespaces", Resource: "namespace", Action: "read"},
			{Code: "namespace.manage", Name: "Manage namespaces", Resource: "namespace", Action: "manage"},
		}
		for _, permission := range permissions {
			if err := tx.Where(rbac.Permission{Code: permission.Code}).FirstOrCreate(&permission).Error; err != nil {
				return err
			}
		}

		adminRole := rbac.Role{Name: "Administrator", Code: "admin", Description: "System administrator"}
		if err := tx.Where(rbac.Role{Code: adminRole.Code}).FirstOrCreate(&adminRole).Error; err != nil {
			return err
		}

		var savedPermissions []rbac.Permission
		if err := tx.Find(&savedPermissions).Error; err != nil {
			return err
		}
		for _, permission := range savedPermissions {
			link := rbac.RolePermission{RoleID: adminRole.ID, PermissionID: permission.ID}
			if err := tx.Where(link).FirstOrCreate(&link).Error; err != nil {
				return err
			}
		}

		var admin user.User
		err := tx.Where(user.User{Username: cfg.DefaultAdmin}).First(&admin).Error
		if err == gorm.ErrRecordNotFound {
			admin = user.User{
				Username:    cfg.DefaultAdmin,
				DisplayName: "Default Admin",
				Email:       "admin@example.com",
				Status:      "active",
				SourceType:  user.SourceLocal,
			}
			if err := tx.Create(&admin).Error; err != nil {
				return err
			}
			hash, err := security.HashPassword(cfg.DefaultPassword)
			if err != nil {
				return err
			}
			if err := tx.Create(&user.Credential{UserID: admin.ID, PasswordHash: hash}).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		binding := rbac.SubjectRoleBinding{
			SubjectType: "user",
			SubjectID:   fmt.Sprintf("%d", admin.ID),
			RoleID:      adminRole.ID,
		}
		if err := tx.Where(binding).FirstOrCreate(&binding).Error; err != nil {
			return err
		}

		settings := []systemsetting.SystemSetting{
			{Key: "k8s.sync.interval_seconds", Value: "300", Description: "Default kubernetes metadata sync interval"},
			{Key: "k8s.default_namespace_source", Value: "manual", Description: "Default namespace source type for manual records"},
			{Key: "k8s.audit.enabled", Value: "true", Description: "Enable kubernetes audit tracking in future modules"},
		}
		for _, item := range settings {
			record := item
			if err := tx.Where(systemsetting.SystemSetting{Key: item.Key}).Assign(systemsetting.SystemSetting{
				Value:       item.Value,
				Description: item.Description,
			}).FirstOrCreate(&record).Error; err != nil {
				return err
			}
		}

		cipher := secure.New(cfg.JWTSecret)
		for _, item := range cfg.CloudAccounts {
			if err := upsertCloudAccount(tx, cipher, item); err != nil {
				return err
			}
		}

		return nil
	})
}

func upsertCloudAccount(tx *gorm.DB, cipher *secure.Cipher, input config.CloudAccountConfig) error {
	name := strings.TrimSpace(input.Name)
	provider := strings.ToLower(strings.TrimSpace(input.Provider))
	region := strings.TrimSpace(input.Region)
	if name == "" || provider == "" || region == "" {
		return nil
	}

	model := cloud.CloudAccount{}
	err := tx.Where(cloud.CloudAccount{Name: name}).First(&model).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	model.Name = name
	model.Provider = provider
	model.Region = region
	model.RoleARN = strings.TrimSpace(input.RoleARN)
	model.DefaultTagsJSON = mustJSONString(cleanStringList(input.DefaultTags))
	model.DefaultZonesJSON = mustJSONString(cleanStringList(input.DefaultZones))
	model.Status = "configured"

	if accessKey := strings.TrimSpace(input.AccessKey); accessKey != "" {
		encrypted, err := cipher.Encrypt(accessKey)
		if err != nil {
			return err
		}
		model.AccessKeyEncrypted = encrypted
	}
	if secretKey := strings.TrimSpace(input.SecretKey); secretKey != "" {
		encrypted, err := cipher.Encrypt(secretKey)
		if err != nil {
			return err
		}
		model.SecretKeyEncrypted = encrypted
	}

	if err == gorm.ErrRecordNotFound {
		return tx.Create(&model).Error
	}
	return tx.Save(&model).Error
}

func cleanStringList(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func mustJSONString(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
