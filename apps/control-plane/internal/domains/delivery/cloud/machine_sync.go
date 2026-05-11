package cloud

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"backend-center/internal/domains/machines/machine"

	"gorm.io/gorm"
)

func syncMachineAssetsFromCloud(tx *gorm.DB, job DeploymentJob, entries []ResourceInventoryEntryInput) error {
	if tx == nil || strings.TrimSpace(strings.ToLower(job.Status)) != "succeeded" {
		return nil
	}
	for _, entry := range entries {
		if !shouldSyncMachineAsset(entry) {
			continue
		}
		if err := upsertMachineAssetFromCloud(tx, job, entry); err != nil {
			return err
		}
	}
	return nil
}

func shouldSyncMachineAsset(entry ResourceInventoryEntryInput) bool {
	return isMachineAssetResourceType(entry.ResourceType) && strings.TrimSpace(strings.ToLower(entry.LifecycleState)) == "managed"
}

func upsertMachineAssetFromCloud(tx *gorm.DB, job DeploymentJob, entry ResourceInventoryEntryInput) error {
	metadata := cloneInventoryMetadata(entry.Metadata)
	resourceID := firstNonEmpty(
		strings.TrimSpace(entry.CloudID),
		metadataString(metadata["cloud_id"]),
		metadataString(nestedMetadataValue(metadata, "cloud_id")),
	)
	if resourceID == "" {
		return nil
	}

	address := firstNonEmpty(
		metadataString(metadata["public_ip"]),
		metadataString(nestedMetadataValue(metadata, "public_ip")),
		metadataString(metadata["private_ip"]),
		metadataString(nestedMetadataValue(metadata, "private_ip")),
	)
	privateIP := firstNonEmpty(
		metadataString(metadata["private_ip"]),
		metadataString(nestedMetadataValue(metadata, "private_ip")),
	)
	if address == "" {
		return nil
	}

	name := resolveMachineAssetName(entry, metadata)
	groupName := resolveMachineAssetGroup(job, metadata)
	accountName := resolveMachineLoginUser(job, metadata)
	port := 22
	protocol := "ssh"
	loginPolicy := "prompt_first"
	status := "online"
	now := time.Now()

	description := buildMachineAssetDescription(job, entry, metadata)
	tags := buildMachineAssetTags(job, entry, metadata)

	var existing machine.Asset
	err := tx.Where("source_type = ? AND source_provider = ? AND source_resource_id = ?", "cloud", job.Provider, resourceID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	sourceJobID := job.ID
	updates := map[string]any{
		"name":                  name,
		"address":               address,
		"private_ip":            privateIP,
		"platform":              "linux",
		"protocol":              protocol,
		"project_id":            job.ProjectID,
		"environment_id":        job.EnvironmentID,
		"stack_id":              job.StackID,
		"foundation_network_id": job.NetworkPlanID,
		"access_mode":           "direct",
		"enrollment_status":     "pending",
		"enrollment_error":      "",
		"is_gateway_node":       isGatewayNodeAsset(entry, metadata),
		"group_name":            groupName,
		"login_policy":          loginPolicy,
		"port":                  port,
		"account":               accountName,
		"status":                status,
		"tags":                  tags,
		"description":           description,
		"last_seen_at":          &now,
		"source_type":           "cloud",
		"source_provider":       strings.TrimSpace(job.Provider),
		"source_resource_id":    resourceID,
		"source_job_id":         &sourceJobID,
		"updated_at":            now,
	}

	if err == gorm.ErrRecordNotFound {
		record := machine.Asset{
			Name:                name,
			Address:             address,
			PrivateIP:           privateIP,
			Platform:            "linux",
			Protocol:            protocol,
			ProjectID:           job.ProjectID,
			EnvironmentID:       job.EnvironmentID,
			StackID:             job.StackID,
			FoundationNetworkID: job.NetworkPlanID,
			AccessMode:          "direct",
			EnrollmentStatus:    "pending",
			EnrollmentError:     "",
			IsGatewayNode:       isGatewayNodeAsset(entry, metadata),
			GroupName:           groupName,
			LoginPolicy:         loginPolicy,
			Port:                port,
			Account:             accountName,
			Status:              status,
			Tags:                tags,
			Description:         description,
			LastSeenAt:          &now,
			SourceType:          "cloud",
			SourceProvider:      strings.TrimSpace(job.Provider),
			SourceResourceID:    resourceID,
			SourceJobID:         &sourceJobID,
		}
		return tx.Create(&record).Error
	}

	return tx.Model(&existing).Updates(updates).Error
}

func cloneInventoryMetadata(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func resolveMachineAssetName(entry ResourceInventoryEntryInput, metadata map[string]any) string {
	if tags, ok := metadata["tags"].(map[string]any); ok {
		if name := strings.TrimSpace(fmt.Sprint(tags["Name"])); name != "" {
			return name
		}
	}
	if name := stringValue(nestedTagValue(metadata, "Name")); name != "" {
		return name
	}
	return firstNonEmpty(
		metadataString(metadata["name"]),
		metadataString(nestedMetadataValue(metadata, "name")),
		strings.TrimSpace(entry.ResourceName),
		strings.TrimSpace(entry.CloudID),
	)
}

func resolveMachineLoginUser(job DeploymentJob, metadata map[string]any) string {
	if value := metadataString(metadata["login_username"]); value != "" {
		return value
	}
	if value := metadataString(nestedMetadataValue(metadata, "login_username")); value != "" {
		return value
	}
	switch strings.TrimSpace(strings.ToLower(job.Provider)) {
	case "aws":
		return "ec2-user"
	case "alicloud":
		return "root"
	default:
		return ""
	}
}

func buildMachineAssetDescription(job DeploymentJob, entry ResourceInventoryEntryInput, metadata map[string]any) string {
	parts := []string{
		"cloud-auto-sync",
		strings.TrimSpace(job.Provider),
		strings.TrimSpace(entry.ResourceType),
		firstNonEmpty(strings.TrimSpace(entry.CloudID), metadataString(metadata["cloud_id"])),
		fmt.Sprintf("job=%d", job.ID),
	}
	if job.NetworkPlanID != nil {
		parts = append(parts, fmt.Sprintf("network=%d", *job.NetworkPlanID))
	}
	if region := strings.TrimSpace(entry.Region); region != "" {
		parts = append(parts, region)
	}
	if blueprintCode := metadataString(metadata["blueprint_code"]); blueprintCode != "" {
		parts = append(parts, blueprintCode)
	}
	if machineGroup := resolveMachineAssetGroup(job, metadata); machineGroup != "" {
		parts = append(parts, fmt.Sprintf("group=%s", machineGroup))
	}
	description := strings.Join(parts, " | ")
	if len(description) > 255 {
		return description[:255]
	}
	return description
}

func resolveMachineAssetGroup(job DeploymentJob, metadata map[string]any) string {
	if value := metadataString(metadata["machine_group_name"]); value != "" {
		return value
	}
	input := parseMapFromJSON(job.InputJSON)
	if nested := parseNestedJobInput(job.InputJSON); len(nested) > 0 {
		input = nested
	}
	if value := metadataString(input["machine_group_name"]); value != "" {
		return value
	}
	if value := metadataString(input["project_name"]); value != "" {
		return value
	}
	return "Cloud Imported"
}

func isMachineAssetResourceType(resourceType string) bool {
	value := strings.TrimSpace(strings.ToLower(resourceType))
	return value == "aws_instance" || value == "alicloud_instance"
}

func metadataString(value any) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" {
		return ""
	}
	return text
}

func parseNestedJobInput(raw string) map[string]any {
	root := parseMapFromJSON(raw)
	nested, ok := root["input"].(map[string]any)
	if ok {
		return nested
	}
	return map[string]any{}
}

func parseMapFromJSON(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return map[string]any{}
	}
	return result
}

func buildMachineAssetTags(job DeploymentJob, entry ResourceInventoryEntryInput, metadata map[string]any) string {
	values := []string{
		"cloud",
		"auto-managed",
		strings.TrimSpace(strings.ToLower(job.Provider)),
		strings.TrimSpace(strings.ToLower(entry.ResourceType)),
	}
	if tags, ok := metadata["tags"].(map[string]any); ok {
		if role := strings.TrimSpace(strings.ToLower(fmt.Sprint(tags["Role"]))); role != "" {
			values = append(values, role)
		}
	}
	if role := strings.TrimSpace(strings.ToLower(stringValue(nestedTagValue(metadata, "Role")))); role != "" {
		values = append(values, role)
	}
	if isGatewayNodeAsset(entry, metadata) {
		values = append(values, "gateway", "jump-host")
	}
	return strings.Join(uniqueNormalizedTags(values), ",")
}

func isGatewayNodeAsset(entry ResourceInventoryEntryInput, metadata map[string]any) bool {
	resourceName := strings.TrimSpace(strings.ToLower(entry.ResourceName))
	blueprintCode := strings.TrimSpace(strings.ToLower(fmt.Sprint(metadata["blueprint_code"])))
	if strings.Contains(resourceName, "bastion") || strings.Contains(blueprintCode, "bastion") {
		return true
	}
	if tags, ok := metadata["tags"].(map[string]any); ok {
		role := strings.TrimSpace(strings.ToLower(fmt.Sprint(tags["Role"])))
		if role == "bastion" || role == "gateway" {
			return true
		}
	}
	role := strings.TrimSpace(strings.ToLower(stringValue(nestedTagValue(metadata, "Role"))))
	return role == "bastion" || role == "gateway"
}

func uniqueNormalizedTags(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		tag := slugifyMachineSyncTag(value)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}
	return result
}

func slugifyMachineSyncTag(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
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
