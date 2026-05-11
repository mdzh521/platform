package cloud

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend-center/internal/domains/clusters/k8s"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const clusterAddonHeartbeatTimeout = 5 * time.Minute

type AddonService struct {
	base *baseService
}

func (s *AddonService) RecoverTimedOutExecutions() error {
	cutoff := time.Now().Add(-clusterAddonHeartbeatTimeout)
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var items []ClusterAddonExecution
		if err := tx.
			Where("status IN ?", []string{"claimed", "installing"}).
			Where("ended_at IS NULL").
			Where("heartbeat_at IS NULL OR heartbeat_at < ?", cutoff).
			Find(&items).Error; err != nil {
			return err
		}
		for _, item := range items {
			now := time.Now()
			item.Status = "failed"
			item.EndedAt = &now
			item.ErrorMessage = fmt.Sprintf("addon execution heartbeat timeout after %s", clusterAddonHeartbeatTimeout)
			if err := tx.Save(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *AddonService) ReconcileExistingExecutions() error {
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var resources []CloudResource
		if err := tx.
			Where("resource_role = ?", "cluster").
			Where("lifecycle_state = ?", "managed").
			Where("cluster_enrollment_status = ?", "enrolled").
			Order("id asc").
			Find(&resources).Error; err != nil {
			return err
		}
		for _, item := range resources {
			if err := s.ensureExecutionsForResource(tx, item); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *AddonService) ListByCluster(clusterID uint) ([]ClusterAddonExecutionView, error) {
	_ = s.RecoverTimedOutExecutions()
	var executions []ClusterAddonExecution
	if err := s.base.db.Where("cluster_id = ?", clusterID).Order("id asc").Find(&executions).Error; err != nil {
		return nil, err
	}
	var cluster k8s.Cluster
	_ = s.base.db.First(&cluster, clusterID).Error
	result := make([]ClusterAddonExecutionView, 0, len(executions))
	for _, item := range executions {
		result = append(result, clusterAddonExecutionToView(item, cluster.Name))
	}
	return result, nil
}

func (s *AddonService) Retry(clusterID uint, addonKey string) error {
	addonKey = strings.TrimSpace(addonKey)
	if addonKey == "" {
		return errors.New("addon_key 不能为空")
	}
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var item ClusterAddonExecution
		if err := tx.Where("cluster_id = ? AND addon_key = ?", clusterID, addonKey).First(&item).Error; err != nil {
			return err
		}
		item.Status = "queued"
		item.WorkerName = ""
		item.ErrorMessage = ""
		item.ResultJSON = ""
		item.StartedAt = nil
		item.EndedAt = nil
		item.HeartbeatAt = nil
		return tx.Save(&item).Error
	})
}

func (s *AddonService) ClaimNext(ctx context.Context, input ClusterAddonClaimRequest) (*ClusterAddonClaimView, error) {
	_ = s.RecoverTimedOutExecutions()
	workerName := strings.TrimSpace(input.WorkerName)
	if workerName == "" {
		return nil, errors.New("worker_name 不能为空")
	}
	var (
		execution ClusterAddonExecution
		cluster   k8s.Cluster
		account   CloudAccount
		now       = time.Now()
	)
	err := s.base.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ?", "queued").
			Order("updated_at asc, id asc").
			First(&execution).Error; err != nil {
			return err
		}
		if err := tx.First(&cluster, execution.ClusterID).Error; err != nil {
			return err
		}
		if err := tx.First(&account, execution.AccountID).Error; err != nil {
			return err
		}
		execution.Status = "installing"
		execution.WorkerName = workerName
		execution.ErrorMessage = ""
		execution.EndedAt = nil
		execution.StartedAt = &now
		execution.HeartbeatAt = &now
		return tx.Save(&execution).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	executionEnv, err := (&AccountService{base: s.base}).ResolveExecutionEnvironment(account.ID, execution.Provider, execution.Region)
	if err != nil {
		return nil, err
	}
	return &ClusterAddonClaimView{
		ExecutionID: execution.ID,
		ClusterID:   cluster.ID,
		ClusterName: cluster.Name,
		Provider:    execution.Provider,
		AccountID:   execution.AccountID,
		Region:      execution.Region,
		AddonKey:    execution.AddonKey,
		AddonType:   execution.AddonType,
		InstallMode: execution.InstallMode,
		ReleaseName: execution.ReleaseName,
		Namespace:   execution.Namespace,
		Contract:    decodeMetadata(execution.ContractJSON),
		Environment: executionEnv,
	}, nil
}

func (s *AddonService) Heartbeat(executionID uint, input ClusterAddonHeartbeatInput) error {
	workerName := strings.TrimSpace(input.WorkerName)
	if workerName == "" {
		return errors.New("worker_name 不能为空")
	}
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var item ClusterAddonExecution
		if err := tx.First(&item, executionID).Error; err != nil {
			return err
		}
		if strings.TrimSpace(item.WorkerName) != "" && item.WorkerName != workerName {
			return errors.New("addon worker 不匹配")
		}
		if item.EndedAt != nil || item.Status == "succeeded" || item.Status == "failed" {
			return nil
		}
		now := time.Now()
		item.WorkerName = workerName
		item.HeartbeatAt = &now
		if status := normalizeAddonHeartbeatStatus(input.Status); status != "" {
			item.Status = status
		}
		return tx.Save(&item).Error
	})
}

func (s *AddonService) ReportResult(executionID uint, input ClusterAddonResultInput) error {
	workerName := strings.TrimSpace(input.WorkerName)
	if workerName == "" {
		return errors.New("worker_name 不能为空")
	}
	status := normalizeAddonResultStatus(input.Status)
	if status == "" {
		return errors.New("不支持的 addon 状态")
	}
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var item ClusterAddonExecution
		if err := tx.First(&item, executionID).Error; err != nil {
			return err
		}
		if strings.TrimSpace(item.WorkerName) != "" && item.WorkerName != workerName {
			return errors.New("addon worker 不匹配")
		}
		now := time.Now()
		item.WorkerName = workerName
		item.Status = status
		item.HeartbeatAt = &now
		item.EndedAt = &now
		item.ErrorMessage = strings.TrimSpace(input.ErrorMessage)
		if input.Result != nil {
			item.ResultJSON = mustJSONString(input.Result)
		}
		if status == "succeeded" {
			item.LastSuccessAt = &now
			item.ErrorMessage = ""
		}
		return tx.Save(&item).Error
	})
}

func (s *AddonService) ensureExecutionsForResource(tx *gorm.DB, item CloudResource) error {
	if !isClusterControlPlaneResourceType(item.ResourceType) || item.SourceJobID == 0 {
		return nil
	}
	var cluster k8s.Cluster
	if err := tx.
		Where("source_resource_id = ? AND provider = ?", strings.TrimSpace(item.CloudID), normalizeProvider(item.Provider)).
		First(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var job DeploymentJob
	if err := tx.First(&job, item.SourceJobID).Error; err != nil {
		return err
	}
	contract := parsePlatformAddonContract(job.OutputJSON)
	helmAddons, _ := contract["helm_addons"].(map[string]any)
	for addonKey, raw := range helmAddons {
		addonContract, ok := raw.(map[string]any)
		if !ok || !toBool(addonContract["enabled"]) {
			continue
		}
		if normalizeAddonType(addonKey) != "helm" {
			continue
		}
		var existing ClusterAddonExecution
		err := tx.Where("cluster_id = ? AND addon_key = ?", cluster.ID, addonKey).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		record := ClusterAddonExecution{
			ClusterID:    cluster.ID,
			SourceJobID:  job.ID,
			ResourceID:   item.ID,
			Provider:     normalizeProvider(item.Provider),
			AccountID:    item.AccountID,
			Region:       firstNonEmpty(strings.TrimSpace(item.Region), executionRegion(parseInputJSONMap(job.InputJSON), "")),
			AddonKey:     addonKey,
			AddonType:    normalizeAddonType(addonKey),
			InstallMode:  stringValue(addonContract["install_mode"]),
			ReleaseName:  firstNonEmpty(stringValue(addonContract["release_name"]), strings.ReplaceAll(addonKey, "_", "-")),
			Namespace:    firstNonEmpty(stringValue(addonContract["namespace"]), "kube-system"),
			ContractJSON: mustJSONString(addonContract),
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			record.Status = "queued"
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
			continue
		}
		existing.SourceJobID = record.SourceJobID
		existing.ResourceID = record.ResourceID
		existing.Provider = record.Provider
		existing.AccountID = record.AccountID
		existing.Region = record.Region
		existing.AddonType = record.AddonType
		existing.InstallMode = record.InstallMode
		existing.ReleaseName = record.ReleaseName
		existing.Namespace = record.Namespace
		existing.ContractJSON = record.ContractJSON
		if strings.TrimSpace(existing.Status) == "" {
			existing.Status = "queued"
		}
		if err := tx.Save(&existing).Error; err != nil {
			return err
		}
	}
	return nil
}

func clusterAddonExecutionToView(item ClusterAddonExecution, clusterName string) ClusterAddonExecutionView {
	return ClusterAddonExecutionView{
		ID:            item.ID,
		ClusterID:     item.ClusterID,
		ClusterName:   clusterName,
		SourceJobID:   item.SourceJobID,
		ResourceID:    item.ResourceID,
		Provider:      item.Provider,
		AccountID:     item.AccountID,
		Region:        item.Region,
		AddonKey:      item.AddonKey,
		AddonType:     item.AddonType,
		InstallMode:   item.InstallMode,
		ReleaseName:   item.ReleaseName,
		Namespace:     item.Namespace,
		Status:        item.Status,
		WorkerName:    item.WorkerName,
		ErrorMessage:  item.ErrorMessage,
		Contract:      decodeMetadata(item.ContractJSON),
		Result:        decodeMetadata(item.ResultJSON),
		StartedAt:     item.StartedAt,
		EndedAt:       item.EndedAt,
		HeartbeatAt:   item.HeartbeatAt,
		LastSuccessAt: item.LastSuccessAt,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func parsePlatformAddonContract(outputJSON string) map[string]any {
	payload := parseInputJSONMap(outputJSON)
	outputs, _ := payload["outputs"].(map[string]any)
	row, _ := outputs["platform_addon_contract"].(map[string]any)
	value, _ := row["value"].(map[string]any)
	if len(value) > 0 {
		return value
	}
	if len(row) > 0 {
		return row
	}
	return map[string]any{}
}

func normalizeAddonType(addonKey string) string {
	switch strings.TrimSpace(addonKey) {
	case "aws_load_balancer_controller":
		return "helm"
	default:
		return ""
	}
}

func normalizeAddonHeartbeatStatus(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "claimed":
		return "claimed"
	case "installing", "running":
		return "installing"
	default:
		return ""
	}
}

func normalizeAddonResultStatus(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "succeeded", "installed":
		return "succeeded"
	case "failed", "error":
		return "failed"
	default:
		return ""
	}
}

func toBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		value := strings.TrimSpace(strings.ToLower(typed))
		return value == "true" || value == "1" || value == "yes"
	default:
		return false
	}
}
