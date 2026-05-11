package cloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ResourceService struct {
	base *baseService
}

func (s *ResourceService) List() ([]CloudResourceView, error) {
	var items []CloudResource
	if err := s.base.db.Order("updated_at desc, id desc").Find(&items).Error; err != nil {
		return nil, err
	}
	accountNames, _ := loadNameMap[CloudAccount](s.base.db)
	blueprintNames, _ := loadNameMap[DeploymentBlueprint](s.base.db)
	networkNames, _ := loadNameMap[NetworkPlan](s.base.db)
	result := make([]CloudResourceView, 0, len(items))
	for _, item := range items {
		result = append(result, cloudResourceToView(
			item,
			accountNames[item.AccountID],
			blueprintNames[item.BlueprintID],
			networkLabel(item.NetworkPlanID, networkNames),
		))
	}
	return result, nil
}

func (s *ResourceService) ClaimNextSync(input ResourceSyncClaimRequest) (*ResourceSyncClaimView, error) {
	workerName := strings.TrimSpace(input.WorkerName)
	if workerName == "" {
		return nil, errors.New("worker_name 不能为空")
	}

	var (
		job       DeploymentJob
		blueprint DeploymentBlueprint
		account   CloudAccount
	)

	err := s.base.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status IN ?", []string{"planned", "succeeded", "destroyed"}).
			Where("coalesce(resource_sync_status, '') IN ?", []string{"", "pending", "failed"}).
			Order("ended_at asc, id asc").
			First(&job).Error; err != nil {
			return err
		}
		if err := tx.First(&blueprint, job.BlueprintID).Error; err != nil {
			return err
		}
		if err := tx.First(&account, job.AccountID).Error; err != nil {
			return err
		}
		job.ResourceSyncStatus = "syncing"
		job.ResourceSyncWorker = workerName
		job.ResourceSyncError = ""
		return tx.Save(&job).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	networkNames, _ := loadNameMap[NetworkPlan](s.base.db)
	resources, _ := loadJobResources(s.base.db, job.ID)
	view := jobToView(job, account.Name, blueprint.Name, networkLabel(job.NetworkPlanID, networkNames), int64(len(resources)))

	region := account.Region
	if parsed := executionRegion(parseMap(view.Input), account.Region); strings.TrimSpace(parsed) != "" {
		region = parsed
	}
	return &ResourceSyncClaimView{
		ID:                job.ID,
		Name:              job.Name,
		Provider:          job.Provider,
		Status:            job.Status,
		Action:            job.Action,
		AccountID:         job.AccountID,
		AccountName:       account.Name,
		BlueprintID:       job.BlueprintID,
		BlueprintName:     blueprint.Name,
		BlueprintCode:     blueprint.Code,
		BlueprintCategory: blueprint.Category,
		NetworkPlanID:     job.NetworkPlanID,
		NetworkPlan:       networkLabel(job.NetworkPlanID, networkNames),
		Region:            region,
		Input:             view.Input,
		PlanSummary:       view.PlanSummary,
		Output:            view.Output,
		Resources:         resources,
	}, nil
}

func (s *ResourceService) ReportSyncResult(jobID uint, input ResourceSyncResultInput) error {
	workerName := strings.TrimSpace(input.WorkerName)
	if workerName == "" {
		return errors.New("worker_name 不能为空")
	}
	status := normalizeField(strings.ToLower(input.Status), "synced")
	if status != "synced" && status != "failed" {
		return errors.New("不支持的资源同步状态")
	}

	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var job DeploymentJob
		if err := tx.First(&job, jobID).Error; err != nil {
			return err
		}
		if strings.TrimSpace(job.ResourceSyncWorker) != "" && job.ResourceSyncWorker != workerName {
			return errors.New("resource sync worker 不匹配")
		}

		if status == "failed" {
			job.ResourceSyncStatus = "failed"
			job.ResourceSyncError = normalizeField(input.ErrorMessage, "资源回填失败")
			job.ResourceSyncWorker = workerName
			return tx.Save(&job).Error
		}

		if err := s.applyInventoryEntries(tx, job, input.Entries); err != nil {
			job.ResourceSyncStatus = "failed"
			job.ResourceSyncError = err.Error()
			job.ResourceSyncWorker = workerName
			_ = tx.Save(&job).Error
			return err
		}
		now := time.Now()
		job.ResourceSyncStatus = "synced"
		job.ResourceSyncError = ""
		job.ResourceSyncWorker = workerName
		job.ResourceSyncedAt = &now
		return tx.Save(&job).Error
	})
}

func (s *ResourceService) applyInventoryEntries(tx *gorm.DB, job DeploymentJob, entries []ResourceInventoryEntryInput) error {
	now := time.Now()
	if job.Action == "destroy" || job.Status == "destroyed" {
		if err := s.markScopeDestroyed(tx, job, now); err != nil {
			return err
		}
	}
	for _, entry := range entries {
		if strings.TrimSpace(entry.ResourceType) == "" || strings.TrimSpace(entry.ResourceName) == "" {
			continue
		}
		key := buildCloudResourceKey(job, entry)
		var item CloudResource
		err := tx.Where("sync_key = ?", key).First(&item).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		item.SyncKey = key
		item.Provider = job.Provider
		item.AccountID = job.AccountID
		item.BlueprintID = job.BlueprintID
		item.NetworkPlanID = job.NetworkPlanID
		item.ProjectID = job.ProjectID
		item.EnvironmentID = job.EnvironmentID
		item.StackID = job.StackID
		item.SourceJobID = job.ID
		item.Category = normalizeField(entry.Category, "network")
		item.Region = strings.TrimSpace(entry.Region)
		item.ResourceType = strings.TrimSpace(entry.ResourceType)
		item.ResourceName = strings.TrimSpace(entry.ResourceName)
		item.CloudID = strings.TrimSpace(entry.CloudID)
		item.ResourceRole = resolveResourceRole(entry)
		item.OwnershipMode = resolveOwnershipMode(job)
		item.LifecycleState = normalizeLifecycleState(entry.LifecycleState, job.Status)
		item.MachineEnrollmentStatus = resolveMachineEnrollmentStatus(item.ResourceRole, item.LifecycleState)
		item.ClusterEnrollmentStatus = resolveClusterEnrollmentStatus(item.ResourceRole, item.ResourceType, item.LifecycleState)
		item.MetadataJSON = mustJSONString(entry.Metadata)
		item.LastSyncedAt = &now

		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
			continue
		}
		if err := tx.Save(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *ResourceService) markScopeDestroyed(tx *gorm.DB, job DeploymentJob, syncedAt time.Time) error {
	updates := map[string]any{
		"source_job_id":   job.ID,
		"lifecycle_state": "destroyed",
		"last_synced_at":  syncedAt,
		"updated_at":      syncedAt,
	}
	query := tx.Model(&CloudResource{}).
		Where("provider = ?", job.Provider).
		Where("account_id = ?", job.AccountID).
		Where("blueprint_id = ?", job.BlueprintID)
	if job.NetworkPlanID != nil {
		query = query.Where("network_plan_id = ?", *job.NetworkPlanID)
	} else {
		query = query.Where("network_plan_id IS NULL")
	}
	return query.Updates(updates).Error
}

func buildCloudResourceKey(job DeploymentJob, entry ResourceInventoryEntryInput) string {
	networkPlanID := uint(0)
	if job.NetworkPlanID != nil {
		networkPlanID = *job.NetworkPlanID
	}
	region := strings.TrimSpace(entry.Region)
	cloudID := strings.TrimSpace(entry.CloudID)
	identity := strings.TrimSpace(entry.ResourceName)
	if cloudID != "" {
		identity = cloudID
	}
	return fmt.Sprintf("%s:%d:%d:%d:%s:%s:%s",
		strings.TrimSpace(job.Provider),
		job.AccountID,
		job.BlueprintID,
		networkPlanID,
		region,
		strings.TrimSpace(entry.ResourceType),
		identity,
	)
}

func normalizeLifecycleState(value, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "planned", "managed", "destroyed":
		return value
	}
	switch strings.TrimSpace(strings.ToLower(fallback)) {
	case "planned":
		return "planned"
	case "succeeded":
		return "managed"
	case "destroyed":
		return "destroyed"
	default:
		return "planned"
	}
}

func resolveResourceRole(entry ResourceInventoryEntryInput) string {
	resourceType := strings.TrimSpace(strings.ToLower(entry.ResourceType))
	switch {
	case strings.Contains(resourceType, "instance"):
		return "compute"
	case isClusterControlPlaneResourceType(resourceType):
		return "cluster"
	case strings.Contains(resourceType, "subnet") || strings.Contains(resourceType, "vswitch") || strings.Contains(resourceType, "vpc"):
		return "network"
	default:
		return normalizeField(entry.Category, "resource")
	}
}

func isClusterControlPlaneResourceType(resourceType string) bool {
	resourceType = strings.TrimSpace(strings.ToLower(resourceType))
	switch {
	case resourceType == "":
		return false
	case strings.Contains(resourceType, "ack_service"):
		return false
	case strings.Contains(resourceType, "node_group"), strings.Contains(resourceType, "nodegroup"), strings.Contains(resourceType, "fargate_profile"):
		return false
	case resourceType == "aws_eks_cluster", strings.Contains(resourceType, "eks_cluster"):
		return true
	case strings.Contains(resourceType, "ack"), strings.Contains(resourceType, "managed_kubernetes"):
		return true
	case strings.Contains(resourceType, "kubernetes_cluster"):
		return true
	default:
		return false
	}
}

func resolveOwnershipMode(job DeploymentJob) string {
	if normalizeJobType(job.JobType) == "operation" {
		return "hybrid"
	}
	return "terraform"
}

func resolveMachineEnrollmentStatus(resourceRole, lifecycleState string) string {
	if resourceRole != "compute" {
		return "not_applicable"
	}
	if lifecycleState == "destroyed" {
		return "not_applicable"
	}
	return "pending"
}

func resolveClusterEnrollmentStatus(resourceRole, resourceType, lifecycleState string) string {
	if resourceRole != "cluster" {
		return "not_applicable"
	}
	if !isClusterControlPlaneResourceType(resourceType) {
		return "not_applicable"
	}
	if lifecycleState == "destroyed" {
		return "not_applicable"
	}
	return "pending"
}

func parseMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if data, ok := value.(map[string]any); ok {
		return data
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return map[string]any{}
	}
	var result map[string]any
	_ = json.Unmarshal(raw, &result)
	return result
}
