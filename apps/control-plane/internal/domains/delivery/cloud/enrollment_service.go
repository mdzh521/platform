package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend-center/internal/domains/clusters/k8s"
	"backend-center/internal/domains/machines/machine"
	"backend-center/internal/domains/projects/project"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *ResourceService) ClaimNextMachineEnrollment(ctx context.Context, input MachineEnrollmentClaimRequest) (*MachineEnrollmentClaimView, error) {
	workerName := strings.TrimSpace(input.WorkerName)
	if workerName == "" {
		return nil, errors.New("worker_name 不能为空")
	}
	var item CloudResource
	err := s.base.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidates []CloudResource
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("resource_role = ?", "compute").
			Where("lifecycle_state = ?", "managed").
			Where("machine_enrollment_status IN ?", []string{"pending", "failed"}).
			Order("updated_at asc, id asc").
			Limit(50).
			Find(&candidates).Error; err != nil {
			return err
		}
		for _, candidate := range candidates {
			if isMachineAssetResourceType(candidate.ResourceType) {
				item = candidate
				break
			}
		}
		if item.ID == 0 {
			return gorm.ErrRecordNotFound
		}
		item.MachineEnrollmentStatus = "syncing"
		item.MachineEnrollmentWorker = workerName
		item.MachineEnrollmentError = ""
		return tx.Save(&item).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &MachineEnrollmentClaimView{
		ResourceID:          item.ID,
		WorkerName:          workerName,
		Provider:            item.Provider,
		AccountID:           item.AccountID,
		Region:              item.Region,
		ProjectID:           item.ProjectID,
		EnvironmentID:       item.EnvironmentID,
		StackID:             item.StackID,
		FoundationNetworkID: item.NetworkPlanID,
		ResourceType:        item.ResourceType,
		ResourceName:        item.ResourceName,
		CloudID:             item.CloudID,
		Metadata:            decodeMetadata(item.MetadataJSON),
	}, nil
}

func (s *ResourceService) ReportMachineEnrollmentResult(resourceID uint, input MachineEnrollmentResultInput) error {
	workerName := strings.TrimSpace(input.WorkerName)
	if workerName == "" {
		return errors.New("worker_name 不能为空")
	}
	status := strings.TrimSpace(strings.ToLower(input.Status))
	if status == "" {
		status = "enrolled"
	}
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var item CloudResource
		if err := tx.First(&item, resourceID).Error; err != nil {
			return err
		}
		if item.MachineEnrollmentWorker != "" && item.MachineEnrollmentWorker != workerName {
			return errors.New("machine enrollment worker 不匹配")
		}
		if status == "error" || status == "failed" {
			item.MachineEnrollmentStatus = "error"
			item.MachineEnrollmentError = normalizeField(input.ErrorMessage, "machine enrollment failed")
			return tx.Save(&item).Error
		}
		var job DeploymentJob
		if err := tx.First(&job, item.SourceJobID).Error; err != nil {
			return err
		}
		entry := ResourceInventoryEntryInput{
			Category:       item.Category,
			Region:         item.Region,
			ResourceType:   item.ResourceType,
			ResourceName:   item.ResourceName,
			CloudID:        item.CloudID,
			LifecycleState: item.LifecycleState,
			Metadata:       decodeMetadata(item.MetadataJSON),
		}
		if err := upsertMachineAssetFromCloud(tx, job, entry); err != nil {
			item.MachineEnrollmentStatus = "error"
			item.MachineEnrollmentError = err.Error()
			_ = tx.Save(&item).Error
			return err
		}
		now := time.Now()
		item.MachineEnrollmentStatus = "enrolled"
		item.MachineEnrollmentError = ""
		item.MachineEnrolledAt = &now
		if err := tx.Save(&item).Error; err != nil {
			return err
		}
		return tx.Model(&machine.Asset{}).
			Where("source_type = ? AND source_provider = ? AND source_resource_id = ?", "cloud", item.Provider, firstNonEmpty(item.CloudID, fmt.Sprint(entry.Metadata["cloud_id"]))).
			Updates(map[string]any{"enrollment_status": "enrolled", "enrollment_error": ""}).Error
	})
}

func (s *ResourceService) ClaimNextClusterEnrollment(ctx context.Context, input ClusterEnrollmentClaimRequest) (*ClusterEnrollmentClaimView, error) {
	workerName := strings.TrimSpace(input.WorkerName)
	if workerName == "" {
		return nil, errors.New("worker_name 不能为空")
	}
	var item CloudResource
	var account CloudAccount
	err := s.base.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidates []CloudResource
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("resource_role = ?", "cluster").
			Where("lifecycle_state = ?", "managed").
			Where("cluster_enrollment_status IN ?", []string{"pending", "failed"}).
			Order("updated_at asc, id asc").
			Limit(50).
			Find(&candidates).Error; err != nil {
			return err
		}
		for _, candidate := range candidates {
			if isClusterControlPlaneResourceType(candidate.ResourceType) {
				item = candidate
				break
			}
		}
		if item.ID == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.First(&account, item.AccountID).Error; err != nil {
			return err
		}
		item.ClusterEnrollmentStatus = "syncing"
		item.ClusterEnrollmentWorker = workerName
		item.ClusterEnrollmentError = ""
		return tx.Save(&item).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	executionEnv, err := (&AccountService{base: s.base}).ResolveExecutionEnvironment(account.ID, item.Provider, item.Region)
	if err != nil {
		return nil, err
	}
	return &ClusterEnrollmentClaimView{
		ResourceID:          item.ID,
		WorkerName:          workerName,
		Provider:            item.Provider,
		AccountID:           item.AccountID,
		Region:              item.Region,
		ProjectID:           item.ProjectID,
		EnvironmentID:       item.EnvironmentID,
		StackID:             item.StackID,
		FoundationNetworkID: item.NetworkPlanID,
		ResourceType:        item.ResourceType,
		ResourceName:        item.ResourceName,
		CloudID:             item.CloudID,
		Metadata:            decodeMetadata(item.MetadataJSON),
		Environment:         executionEnv,
	}, nil
}

func (s *ResourceService) ReportClusterEnrollmentResult(resourceID uint, input ClusterEnrollmentResultInput) error {
	workerName := strings.TrimSpace(input.WorkerName)
	if workerName == "" {
		return errors.New("worker_name 不能为空")
	}
	status := strings.TrimSpace(strings.ToLower(input.Status))
	if status == "" {
		status = "enrolled"
	}
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var item CloudResource
		if err := tx.First(&item, resourceID).Error; err != nil {
			return err
		}
		if item.ClusterEnrollmentWorker != "" && item.ClusterEnrollmentWorker != workerName {
			return errors.New("cluster enrollment worker 不匹配")
		}
		now := time.Now()
		if status == "error" || status == "failed" {
			item.ClusterEnrollmentStatus = "error"
			item.ClusterEnrollmentError = normalizeField(input.ErrorMessage, "cluster enrollment failed")
			item.ClusterEnrolledAt = &now
			return tx.Save(&item).Error
		}
		if err := upsertClusterEnrollmentRecord(tx, item, input); err != nil {
			item.ClusterEnrollmentStatus = "error"
			item.ClusterEnrollmentError = err.Error()
			item.ClusterEnrolledAt = &now
			_ = tx.Save(&item).Error
			return err
		}
		if err := (&AddonService{base: s.base}).ensureExecutionsForResource(tx, item); err != nil {
			item.ClusterEnrollmentStatus = "error"
			item.ClusterEnrollmentError = err.Error()
			item.ClusterEnrolledAt = &now
			_ = tx.Save(&item).Error
			return err
		}
		item.ClusterEnrollmentStatus = status
		item.ClusterEnrollmentError = ""
		item.ClusterEnrolledAt = &now
		return tx.Save(&item).Error
	})
}

func (s *ResourceService) RetryResourceEnrollment(resourceID uint, input RetryResourceEnrollmentInput) error {
	target := strings.TrimSpace(strings.ToLower(input.Target))
	if target == "" {
		target = "all"
	}
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var item CloudResource
		if err := tx.First(&item, resourceID).Error; err != nil {
			return err
		}
		if item.LifecycleState != "managed" {
			return errors.New("仅 managed 状态资源允许重触发 enrollment")
		}

		updates := map[string]any{}
		handled := false

		if (target == "all" || target == "machine") && item.ResourceRole == "compute" {
			updates["machine_enrollment_status"] = "pending"
			updates["machine_enrollment_worker"] = ""
			updates["machine_enrollment_error"] = ""
			updates["machine_enrolled_at"] = nil
			handled = true
			if err := tx.Model(&machine.Asset{}).
				Where("source_type = ? AND source_provider = ? AND source_resource_id = ?", "cloud", item.Provider, firstNonEmpty(item.CloudID, item.ResourceName)).
				Updates(map[string]any{"enrollment_status": "pending", "enrollment_error": ""}).Error; err != nil {
				return err
			}
		}

		if (target == "all" || target == "cluster") && item.ResourceRole == "cluster" && isClusterControlPlaneResourceType(item.ResourceType) {
			updates["cluster_enrollment_status"] = "pending"
			updates["cluster_enrollment_worker"] = ""
			updates["cluster_enrollment_error"] = ""
			updates["cluster_enrolled_at"] = nil
			handled = true
		}

		if !handled {
			return errors.New("当前资源不支持所选 enrollment 重触发")
		}
		return tx.Model(&item).Updates(updates).Error
	})
}

func decodeMetadata(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return map[string]any{}
	}
	return value
}

func upsertClusterEnrollmentRecord(tx *gorm.DB, item CloudResource, input ClusterEnrollmentResultInput) error {
	cloudMetadata := decodeMetadata(item.MetadataJSON)
	resourceID := resolveClusterEnrollmentResourceID(item, input, cloudMetadata)
	clusterName := resolveClusterEnrollmentName(item, input, cloudMetadata, resourceID)
	code := slugifyClusterCode(clusterName)
	if code == "" {
		code = slugifyClusterCode(resourceID)
	}
	subnetRefsJSON, _ := json.Marshal(input.SubnetRefs)
	var existing k8s.Cluster
	err := tx.Where("source_resource_id = ? AND provider = ?", resourceID, item.Provider).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		err = tx.Where("name = ? AND provider = ?", clusterName, normalizeProvider(item.Provider)).First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
	}
	clusterStatus := normalizeClusterStatusFromEnrollment(input.Status, input.ProviderStatus, input.Endpoint)
	environmentCode, err := resolveClusterEnrollmentEnvironment(tx, item, input, cloudMetadata)
	if err != nil {
		return err
	}
	record := k8s.Cluster{
		Name:                clusterName,
		Code:                ensureUniqueClusterCode(code, resourceID),
		Environment:         environmentCode,
		ProjectID:           item.ProjectID,
		EnvironmentID:       item.EnvironmentID,
		StackID:             item.StackID,
		FoundationNetworkID: item.NetworkPlanID,
		Provider:            normalizeProvider(item.Provider),
		APIEndpoint:         firstNonEmpty(strings.TrimSpace(input.Endpoint), "pending-enrollment"),
		AuthType:            "provider_enrolled",
		SourceResourceID:    resourceID,
		ClusterAccessMode:   normalizeClusterAccessMode(input.AccessMode),
		Version:             strings.TrimSpace(input.Version),
		VPCID:               strings.TrimSpace(input.VPCID),
		SubnetRefsJSON:      string(subnetRefsJSON),
		Description:         encodeClusterEnrollmentDescription(item, input),
		Status:              clusterStatus,
		LastSyncedAt:        ptrTime(time.Now()),
	}
	if err == gorm.ErrRecordNotFound {
		return tx.Create(&record).Error
	}
	existing.Name = record.Name
	existing.Code = record.Code
	existing.Environment = record.Environment
	existing.ProjectID = record.ProjectID
	existing.EnvironmentID = record.EnvironmentID
	existing.StackID = record.StackID
	existing.FoundationNetworkID = record.FoundationNetworkID
	existing.Provider = record.Provider
	existing.APIEndpoint = record.APIEndpoint
	existing.AuthType = record.AuthType
	existing.SourceResourceID = record.SourceResourceID
	existing.ClusterAccessMode = record.ClusterAccessMode
	existing.Version = record.Version
	existing.VPCID = record.VPCID
	existing.SubnetRefsJSON = record.SubnetRefsJSON
	existing.Description = record.Description
	existing.Status = record.Status
	existing.LastSyncedAt = record.LastSyncedAt
	return tx.Save(&existing).Error
}

func resolveClusterEnrollmentResourceID(item CloudResource, input ClusterEnrollmentResultInput, cloudMetadata map[string]any) string {
	return firstNonEmpty(
		strings.TrimSpace(item.CloudID),
		stringValue(input.Metadata["cloud_id"]),
		stringValue(cloudMetadata["cloud_id"]),
		stringValue(nestedMetadataValue(cloudMetadata, "cloud_id")),
	)
}

func resolveClusterEnrollmentName(item CloudResource, input ClusterEnrollmentResultInput, cloudMetadata map[string]any, resourceID string) string {
	return firstNonEmpty(
		strings.TrimSpace(input.ClusterName),
		stringValue(cloudMetadata["cluster_name"]),
		stringValue(nestedMetadataValue(cloudMetadata, "name")),
		stringValue(cloudMetadata["resource_name"]),
		strings.TrimSpace(item.ResourceName),
		resourceID,
	)
}

func resolveClusterEnrollmentEnvironment(tx *gorm.DB, item CloudResource, input ClusterEnrollmentResultInput, cloudMetadata map[string]any) (string, error) {
	environmentCode, err := resolveClusterEnvironmentCode(tx, item.EnvironmentID)
	if err != nil {
		return "", err
	}
	if environmentCode != "unknown" {
		return environmentCode, nil
	}
	return firstNonEmpty(
		stringValue(input.Metadata["environment_code"]),
		stringValue(input.Metadata["environment"]),
		stringValue(cloudMetadata["environment_code"]),
		stringValue(cloudMetadata["environment"]),
		stringValue(nestedTagValue(cloudMetadata, "Environment")),
		stringValue(nestedTagValue(cloudMetadata, "env")),
		resolveSourceJobEnvironmentCode(tx, item.SourceJobID),
		"unknown",
	), nil
}

func resolveSourceJobEnvironmentCode(tx *gorm.DB, sourceJobID uint) string {
	if sourceJobID == 0 {
		return ""
	}
	var job DeploymentJob
	if err := tx.Select("input_json").First(&job, sourceJobID).Error; err != nil {
		return ""
	}
	return stringValue(parseNestedJobInput(job.InputJSON)["environment"])
}

func nestedMetadataValue(metadata map[string]any, key string) any {
	nested, ok := metadata["metadata"].(map[string]any)
	if !ok {
		return nil
	}
	return nested[key]
}

func nestedTagValue(metadata map[string]any, key string) any {
	nested, ok := metadata["metadata"].(map[string]any)
	if !ok {
		return nil
	}
	tags, ok := nested["tags"].(map[string]any)
	if !ok {
		return nil
	}
	return tags[key]
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	trimmed := strings.TrimSpace(fmt.Sprint(value))
	if trimmed == "" || trimmed == "<nil>" {
		return ""
	}
	return trimmed
}

func resolveClusterEnvironmentCode(tx *gorm.DB, environmentID *uint) (string, error) {
	if environmentID == nil {
		return "unknown", nil
	}
	var environment project.Environment
	if err := tx.First(&environment, *environmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "unknown", nil
		}
		return "", err
	}
	if strings.TrimSpace(environment.Code) == "" {
		return "unknown", nil
	}
	return strings.TrimSpace(environment.Code), nil
}

func normalizeClusterStatusFromEnrollment(status, providerStatus, endpoint string) string {
	if strings.TrimSpace(strings.ToLower(status)) == "access_pending" {
		return "access_pending"
	}
	switch strings.TrimSpace(strings.ToLower(providerStatus)) {
	case "active", "running":
		if strings.TrimSpace(endpoint) == "" {
			return "access_pending"
		}
		return "enrolled"
	case "creating", "pending":
		return "created"
	case "failed", "error":
		return "error"
	default:
		if strings.TrimSpace(endpoint) == "" {
			return "access_pending"
		}
		return "enrolled"
	}
}

func normalizeClusterAccessMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "direct", "via_gateway", "private-network":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "direct"
	}
}

func encodeClusterEnrollmentDescription(item CloudResource, input ClusterEnrollmentResultInput) string {
	summary := firstNonEmpty(
		stringValue(input.Metadata["source"]),
		"cluster-enrollment-worker",
	)
	description := fmt.Sprintf(
		"source=%s provider=%s cloud_id=%s provider_status=%s",
		summary,
		strings.TrimSpace(item.Provider),
		strings.TrimSpace(item.CloudID),
		strings.TrimSpace(input.ProviderStatus),
	)
	description = strings.TrimSpace(description)
	if len(description) > 255 {
		return description[:255]
	}
	return description
}

func (s *ResourceService) RepairClusterEnvironmentCodes() error {
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var clusters []k8s.Cluster
		if err := tx.Where("environment_id IS NOT NULL").
			Where("environment = '' OR environment = 'unknown' OR environment LIKE ?", "env-%").
			Find(&clusters).Error; err != nil {
			return err
		}
		for _, cluster := range clusters {
			environmentCode, err := resolveClusterEnvironmentCode(tx, cluster.EnvironmentID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(environmentCode) == "" {
				continue
			}
			if err := tx.Model(&cluster).Update("environment", environmentCode).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ResourceService) RepairClusterEnrollmentScope() error {
	return s.base.db.Transaction(func(tx *gorm.DB) error {
		var items []CloudResource
		if err := tx.Where("resource_role = ?", "cluster").
			Where("cluster_enrollment_status <> ?", "not_applicable").
			Find(&items).Error; err != nil {
			return err
		}
		for _, item := range items {
			if isClusterControlPlaneResourceType(item.ResourceType) {
				continue
			}
			if err := tx.Model(&item).Updates(map[string]any{
				"cluster_enrollment_status": "not_applicable",
				"cluster_enrollment_worker": "",
				"cluster_enrollment_error":  "",
				"cluster_enrolled_at":       nil,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
