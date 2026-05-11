package cloud

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"backend-center/internal/domains/clusters/k8s"

	"gorm.io/gorm"
)

func syncClustersFromCloud(tx *gorm.DB, job DeploymentJob, entries []ResourceInventoryEntryInput) error {
	if tx == nil || strings.TrimSpace(strings.ToLower(job.Status)) != "succeeded" {
		return nil
	}
	for _, entry := range entries {
		if !shouldSyncCluster(entry) {
			continue
		}
		if err := upsertClusterFromCloud(tx, job, entry); err != nil {
			return err
		}
	}
	return nil
}

func shouldSyncCluster(entry ResourceInventoryEntryInput) bool {
	resourceType := strings.TrimSpace(strings.ToLower(entry.ResourceType))
	if strings.TrimSpace(strings.ToLower(entry.LifecycleState)) != "managed" {
		return false
	}
	return isClusterControlPlaneResourceType(resourceType)
}

func upsertClusterFromCloud(tx *gorm.DB, job DeploymentJob, entry ResourceInventoryEntryInput) error {
	metadata := cloneInventoryMetadata(entry.Metadata)
	resourceID := firstNonEmpty(strings.TrimSpace(entry.CloudID), strings.TrimSpace(fmt.Sprint(metadata["cloud_id"])))
	if resourceID == "" {
		return nil
	}
	name := firstNonEmpty(
		strings.TrimSpace(fmt.Sprint(metadata["cluster_name"])),
		strings.TrimSpace(fmt.Sprint(metadata["name"])),
		strings.TrimSpace(entry.ResourceName),
		resourceID,
	)
	endpoint := firstNonEmpty(
		strings.TrimSpace(fmt.Sprint(metadata["endpoint"])),
		strings.TrimSpace(fmt.Sprint(metadata["api_server_endpoint"])),
	)
	if endpoint == "" {
		endpoint = "pending-enrollment"
	}
	code := slugifyClusterCode(name)
	if code == "" {
		code = slugifyClusterCode(resourceID)
	}
	subnetRefsJSON, _ := json.Marshal(extractStringSliceAny(metadata["subnet_refs"], metadata["subnets"], metadata["vswitch_ids"]))
	var existing k8s.Cluster
	err := tx.Where("source_resource_id = ? AND provider = ?", resourceID, job.Provider).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	record := k8s.Cluster{
		Name:                name,
		Code:                ensureUniqueClusterCode(code, resourceID),
		Environment:         resolveClusterEnvironment(job, metadata),
		ProjectID:           job.ProjectID,
		EnvironmentID:       job.EnvironmentID,
		StackID:             job.StackID,
		FoundationNetworkID: job.NetworkPlanID,
		Provider:            normalizeProvider(job.Provider),
		APIEndpoint:         endpoint,
		AuthType:            "provider_enrolled",
		SourceResourceID:    resourceID,
		ClusterAccessMode:   resolveClusterAccessMode(metadata),
		Version:             firstNonEmpty(strings.TrimSpace(fmt.Sprint(metadata["version"])), strings.TrimSpace(fmt.Sprint(metadata["cluster_version"]))),
		VPCID:               firstNonEmpty(strings.TrimSpace(fmt.Sprint(metadata["vpc_id"])), strings.TrimSpace(fmt.Sprint(metadata["vpcId"]))),
		SubnetRefsJSON:      string(subnetRefsJSON),
		Description:         buildClusterDescription(job, entry, metadata),
		Status:              normalizeClusterSyncStatus(metadata),
		LastSyncedAt:        ptrTime(time.Now()),
	}

	if err == gorm.ErrRecordNotFound {
		return tx.Create(&record).Error
	}
	existing.Name = record.Name
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

func resolveClusterEnvironment(job DeploymentJob, metadata map[string]any) string {
	return firstNonEmpty(
		strings.TrimSpace(fmt.Sprint(metadata["environment_code"])),
		strings.TrimSpace(fmt.Sprint(metadata["environment"])),
		strings.TrimSpace(fmt.Sprint(parseNestedJobInput(job.InputJSON)["environment"])),
		"unknown",
	)
}

func resolveClusterAccessMode(metadata map[string]any) string {
	value := strings.TrimSpace(strings.ToLower(fmt.Sprint(metadata["access_mode"])))
	switch value {
	case "direct", "via_gateway", "private-network":
		return value
	default:
		return "direct"
	}
}

func normalizeClusterSyncStatus(metadata map[string]any) string {
	value := strings.TrimSpace(strings.ToLower(fmt.Sprint(metadata["status"])))
	switch value {
	case "active", "ready":
		return "ready"
	case "creating", "pending":
		return "draft"
	case "failed", "error":
		return "error"
	default:
		return "ready"
	}
}

func buildClusterDescription(job DeploymentJob, entry ResourceInventoryEntryInput, metadata map[string]any) string {
	payload := map[string]any{
		"source":          "cloud-auto-enrollment",
		"provider":        job.Provider,
		"resource_type":   entry.ResourceType,
		"resource_name":   entry.ResourceName,
		"cloud_id":        entry.CloudID,
		"source_job_id":   job.ID,
		"network_plan_id": job.NetworkPlanID,
		"region":          entry.Region,
		"vpc_id":          metadata["vpc_id"],
	}
	raw, _ := json.Marshal(payload)
	return string(raw)
}

func extractStringSliceAny(values ...any) []string {
	result := make([]string, 0)
	for _, value := range values {
		result = append(result, extractStringSlice(value)...)
	}
	return result
}

func slugifyClusterCode(value string) string {
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
	return strings.Trim(builder.String(), "-")
}

func ensureUniqueClusterCode(code, resourceID string) string {
	if code != "" {
		return code
	}
	return slugifyClusterCode(resourceID)
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
