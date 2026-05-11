package scopeutil

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type ScopeInput struct {
	ProjectID           *uint
	EnvironmentID       *uint
	StackID             *uint
	FoundationNetworkID *uint
	AccountID           *uint
	Provider            string
}

type projectRow struct {
	ID uint
}

type environmentRow struct {
	ID        uint
	ProjectID uint
}

type stackRow struct {
	ID            uint
	ProjectID     uint
	EnvironmentID uint
	AccountID     *uint
	Provider      string
	StackType     string
}

type accountRow struct {
	ID       uint
	Provider string
}

type networkPlanRow struct {
	ID            uint
	AccountID     uint
	ProjectID     *uint
	EnvironmentID *uint
	StackID       *uint
	Provider      string
}

func ValidateAccountDefaults(db *gorm.DB, projectID, environmentID *uint) error {
	if present(environmentID) && !present(projectID) {
		return errors.New("default_environment_id 必须与 default_project_id 一起提交")
	}
	return Validate(db, ScopeInput{
		ProjectID:     projectID,
		EnvironmentID: environmentID,
	})
}

func Validate(db *gorm.DB, input ScopeInput) error {
	if db == nil {
		return errors.New("scope validate requires db")
	}
	if present(input.EnvironmentID) && !present(input.ProjectID) {
		return errors.New("environment_id 必须与 project_id 一起提交")
	}

	if present(input.ProjectID) {
		var project projectRow
		if err := db.Table("projects").Select("id").First(&project, *input.ProjectID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("project_id=%d 不存在", *input.ProjectID)
			}
			return err
		}
	}

	if present(input.AccountID) {
		var account accountRow
		if err := db.Table("cloud_accounts").Select("id, provider").First(&account, *input.AccountID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("account_id=%d 不存在", *input.AccountID)
			}
			return err
		}
		if provider := strings.TrimSpace(strings.ToLower(input.Provider)); provider != "" && strings.TrimSpace(strings.ToLower(account.Provider)) != provider {
			return fmt.Errorf("account_id=%d 的 provider 为 %s，与请求中的 %s 不一致", *input.AccountID, account.Provider, provider)
		}
	}

	if present(input.EnvironmentID) {
		var env environmentRow
		if err := db.Table("environments").Select("id, project_id").First(&env, *input.EnvironmentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("environment_id=%d 不存在", *input.EnvironmentID)
			}
			return err
		}
		if present(input.ProjectID) && env.ProjectID != *input.ProjectID {
			return fmt.Errorf("environment_id=%d 不属于 project_id=%d", *input.EnvironmentID, *input.ProjectID)
		}
	}

	if present(input.StackID) {
		var stack stackRow
		if err := db.Table("stacks").Select("id, project_id, environment_id, account_id, provider, stack_type").First(&stack, *input.StackID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("stack_id=%d 不存在", *input.StackID)
			}
			return err
		}
		if present(input.ProjectID) && stack.ProjectID != *input.ProjectID {
			return fmt.Errorf("stack_id=%d 不属于 project_id=%d", *input.StackID, *input.ProjectID)
		}
		if present(input.EnvironmentID) && stack.EnvironmentID != *input.EnvironmentID {
			return fmt.Errorf("stack_id=%d 不属于 environment_id=%d", *input.StackID, *input.EnvironmentID)
		}
		if present(input.AccountID) && stack.AccountID != nil && *stack.AccountID != *input.AccountID {
			return fmt.Errorf("stack_id=%d 绑定的 account_id 与当前请求不一致", *input.StackID)
		}
		if provider := strings.TrimSpace(strings.ToLower(input.Provider)); provider != "" && strings.TrimSpace(strings.ToLower(stack.Provider)) != "" && strings.TrimSpace(strings.ToLower(stack.Provider)) != provider {
			return fmt.Errorf("stack_id=%d 的 provider 为 %s，与请求中的 %s 不一致", *input.StackID, stack.Provider, provider)
		}
	}

	if present(input.FoundationNetworkID) {
		var plan networkPlanRow
		if err := db.Table("network_plans").Select("id, account_id, project_id, environment_id, stack_id, provider").First(&plan, *input.FoundationNetworkID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("foundation_network_id=%d 不存在", *input.FoundationNetworkID)
			}
			return err
		}
		if present(input.AccountID) && plan.AccountID != *input.AccountID {
			return fmt.Errorf("foundation_network_id=%d 绑定的 account_id 与当前请求不一致", *input.FoundationNetworkID)
		}
		if present(input.ProjectID) && plan.ProjectID != nil && *plan.ProjectID != *input.ProjectID {
			return fmt.Errorf("foundation_network_id=%d 不属于 project_id=%d", *input.FoundationNetworkID, *input.ProjectID)
		}
		if present(input.EnvironmentID) && plan.EnvironmentID != nil && *plan.EnvironmentID != *input.EnvironmentID {
			return fmt.Errorf("foundation_network_id=%d 不属于 environment_id=%d", *input.FoundationNetworkID, *input.EnvironmentID)
		}
		if present(input.StackID) {
			var stack stackRow
			if err := db.Table("stacks").Select("id, project_id, environment_id, account_id, provider, stack_type").First(&stack, *input.StackID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("stack_id=%d 不存在", *input.StackID)
				}
				return err
			}
			if strings.TrimSpace(strings.ToLower(stack.StackType)) == "foundation-network" {
				if plan.StackID != nil && *plan.StackID != *input.StackID {
					return fmt.Errorf("foundation_network_id=%d 不属于 stack_id=%d", *input.FoundationNetworkID, *input.StackID)
				}
			}
		}
		if provider := strings.TrimSpace(strings.ToLower(input.Provider)); provider != "" && strings.TrimSpace(strings.ToLower(plan.Provider)) != provider {
			return fmt.Errorf("foundation_network_id=%d 的 provider 为 %s，与请求中的 %s 不一致", *input.FoundationNetworkID, plan.Provider, provider)
		}
	}

	return nil
}

func present(value *uint) bool {
	return value != nil && *value > 0
}
