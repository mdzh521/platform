package cloud

import (
	"encoding/json"
	"fmt"
	"strings"

	"backend-center/internal/domains/projects/project"

	"gorm.io/gorm"
)

func bindNetworkPlanScope(tx *gorm.DB, plan *NetworkPlan) error {
	if tx == nil || plan == nil || plan.ProjectID == nil || plan.EnvironmentID == nil {
		return nil
	}
	if plan.StackID != nil && *plan.StackID > 0 {
		return nil
	}
	stack, err := ensureStack(tx, stackBindingInput{
		ProjectID:     *plan.ProjectID,
		EnvironmentID: *plan.EnvironmentID,
		Provider:      plan.Provider,
		AccountID:     &plan.AccountID,
		Region:        plan.Region,
		StackType:     "foundation-network",
		Name:          plan.Name,
		LogicalName:   "main",
		FoundationID:  &plan.ID,
		Metadata: map[string]any{
			"network_plan_id": plan.ID,
			"vpc_cidr":        plan.VPCCIDR,
		},
	})
	if err != nil {
		return err
	}
	plan.StackID = &stack.ID
	return tx.Model(plan).Update("stack_id", stack.ID).Error
}

func bindJobScope(tx *gorm.DB, job *DeploymentJob, blueprint *DeploymentBlueprint, plan *NetworkPlan) error {
	if tx == nil || job == nil {
		return nil
	}
	if plan != nil {
		if job.ProjectID == nil {
			job.ProjectID = plan.ProjectID
		}
		if job.EnvironmentID == nil {
			job.EnvironmentID = plan.EnvironmentID
		}
	}
	if job.ProjectID == nil || job.EnvironmentID == nil {
		projectID, environmentID := extractScopeFromInput(job.InputJSON)
		if job.ProjectID == nil {
			job.ProjectID = projectID
		}
		if job.EnvironmentID == nil {
			job.EnvironmentID = environmentID
		}
	}
	if job.ProjectID == nil || job.EnvironmentID == nil {
		return nil
	}
	if job.StackID != nil && *job.StackID > 0 {
		return nil
	}
	stackType := resolveStackType(blueprint)
	stack, err := ensureStack(tx, stackBindingInput{
		ProjectID:     *job.ProjectID,
		EnvironmentID: *job.EnvironmentID,
		Provider:      job.Provider,
		AccountID:     &job.AccountID,
		Region:        resolveJobRegion(job, plan),
		StackType:     stackType,
		Name:          job.Name,
		LogicalName:   resolveStackLogicalName(job, blueprint),
		FoundationID:  job.NetworkPlanID,
		Metadata: map[string]any{
			"job_id":         job.ID,
			"blueprint_code": blueprint.Code,
		},
	})
	if err != nil {
		return err
	}
	job.StackID = &stack.ID
	return tx.Model(job).Updates(map[string]any{
		"project_id":     job.ProjectID,
		"environment_id": job.EnvironmentID,
		"stack_id":       stack.ID,
		"job_type":       normalizeJobType(job.JobType),
	}).Error
}

type stackBindingInput struct {
	ProjectID     uint
	EnvironmentID uint
	Provider      string
	AccountID     *uint
	Region        string
	StackType     string
	Name          string
	LogicalName   string
	FoundationID  *uint
	Metadata      map[string]any
}

func ensureStack(tx *gorm.DB, input stackBindingInput) (*project.Stack, error) {
	var projectRecord project.Project
	if err := tx.First(&projectRecord, input.ProjectID).Error; err != nil {
		return nil, err
	}
	var envRecord project.Environment
	if err := tx.Where("project_id = ?", input.ProjectID).First(&envRecord, input.EnvironmentID).Error; err != nil {
		return nil, err
	}
	stackCode := fmt.Sprintf("%s/%s/%s/%s/%s/%s",
		normalizeProvider(input.Provider),
		strings.TrimSpace(input.Region),
		strings.TrimSpace(envRecord.Code),
		strings.TrimSpace(projectRecord.Code),
		strings.TrimSpace(input.StackType),
		slugifyStackName(input.LogicalName),
	)
	var existing project.Stack
	err := tx.Where("stack_code = ?", stackCode).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		record := project.Stack{
			ProjectID:           input.ProjectID,
			EnvironmentID:       input.EnvironmentID,
			Provider:            normalizeProvider(input.Provider),
			AccountID:           input.AccountID,
			Region:              strings.TrimSpace(input.Region),
			StackType:           input.StackType,
			StackCode:           stackCode,
			Name:                strings.TrimSpace(input.Name),
			Status:              "active",
			FoundationNetworkID: input.FoundationID,
			OwnerMode:           "terraform",
			DriftStatus:         "clean",
			MetadataJSON:        encodeStackMetadata(input.Metadata),
		}
		if err := tx.Create(&record).Error; err != nil {
			return nil, err
		}
		return &record, nil
	}
	existing.AccountID = input.AccountID
	existing.Region = strings.TrimSpace(input.Region)
	existing.Name = strings.TrimSpace(input.Name)
	existing.FoundationNetworkID = input.FoundationID
	if existing.Status == "" || existing.Status == "draft" {
		existing.Status = "active"
	}
	existing.MetadataJSON = mergeStackMetadata(existing.MetadataJSON, input.Metadata)
	if err := tx.Save(&existing).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

func resolveStackType(blueprint *DeploymentBlueprint) string {
	if blueprint == nil {
		return "compute-batch"
	}
	code := strings.TrimSpace(strings.ToLower(blueprint.Code))
	category := strings.TrimSpace(strings.ToLower(blueprint.Category))
	switch {
	case category == "cluster":
		return "cluster"
	case strings.Contains(code, "bastion"):
		return "access-node"
	case strings.Contains(code, "server"):
		return "compute-batch"
	default:
		return "shared-service"
	}
}

func resolveStackLogicalName(job *DeploymentJob, blueprint *DeploymentBlueprint) string {
	if job == nil {
		return "main"
	}
	input := parseNestedJobInput(job.InputJSON)
	switch resolveStackType(blueprint) {
	case "cluster":
		return firstNonEmpty(fmt.Sprint(input["cluster_name"]), job.Name, "main")
	case "access-node":
		return firstNonEmpty(fmt.Sprint(input["server_name"]), job.Name, "main")
	default:
		return firstNonEmpty(fmt.Sprint(input["server_name"]), job.Name, "main")
	}
}

func resolveJobRegion(job *DeploymentJob, plan *NetworkPlan) string {
	if job == nil {
		return ""
	}
	if plan != nil && strings.TrimSpace(plan.Region) != "" {
		return plan.Region
	}
	input := parseNestedJobInput(job.InputJSON)
	return strings.TrimSpace(fmt.Sprint(input["region"]))
}

func extractScopeFromInput(raw string) (*uint, *uint) {
	input := parseNestedJobInput(raw)
	return optionalUint(input["project_id"]), optionalUint(input["environment_id"])
}

func slugifyStackName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	builder := strings.Builder{}
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	if result := strings.Trim(builder.String(), "-"); result != "" {
		return result
	}
	return "main"
}

func encodeStackMetadata(value map[string]any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func mergeStackMetadata(raw string, incoming map[string]any) string {
	current := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &current)
	for key, value := range incoming {
		current[key] = value
	}
	return encodeStackMetadata(current)
}
