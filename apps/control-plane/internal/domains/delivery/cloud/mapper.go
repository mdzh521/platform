package cloud

import (
	"encoding/json"
	"fmt"
	"strings"
)

func accountToView(item CloudAccount) CloudAccountView {
	return CloudAccountView{
		ID:                   item.ID,
		Name:                 item.Name,
		Provider:             item.Provider,
		Region:               item.Region,
		RoleARN:              item.RoleARN,
		DefaultTags:          parseStringList(item.DefaultTagsJSON),
		DefaultZones:         parseStringList(item.DefaultZonesJSON),
		Status:               item.Status,
		LastCheckedAt:        item.LastCheckedAt,
		DefaultProjectID:     item.DefaultProjectID,
		DefaultEnvironmentID: item.DefaultEnvironmentID,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
		HasAccessKey:         strings.TrimSpace(item.AccessKeyEncrypted) != "",
		HasSecretKey:         strings.TrimSpace(item.SecretKeyEncrypted) != "",
	}
}

func networkToView(item NetworkPlan, accountName string) NetworkPlanView {
	var topology any = map[string]any{}
	_ = json.Unmarshal([]byte(item.TopologyJSON), &topology)
	return NetworkPlanView{
		ID:            item.ID,
		Name:          item.Name,
		Provider:      item.Provider,
		AccountID:     item.AccountID,
		AccountName:   accountName,
		ProjectID:     item.ProjectID,
		EnvironmentID: item.EnvironmentID,
		StackID:       item.StackID,
		Region:        item.Region,
		VPCCIDR:       item.VPCCIDR,
		Topology:      topology,
		Status:        item.Status,
		ResourceBrief: buildNetworkBrief(item.Provider, topology),
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func jobToView(item DeploymentJob, accountName, blueprintName, networkName string, resourceCount int64) DeploymentJobView {
	var input any = map[string]any{}
	if strings.TrimSpace(item.InputJSON) != "" {
		_ = json.Unmarshal([]byte(item.InputJSON), &input)
	}
	var planSummary any = map[string]any{}
	if strings.TrimSpace(item.PlanSummaryJSON) != "" {
		_ = json.Unmarshal([]byte(item.PlanSummaryJSON), &planSummary)
	}
	var output any = map[string]any{}
	if strings.TrimSpace(item.OutputJSON) != "" {
		_ = json.Unmarshal([]byte(item.OutputJSON), &output)
	}
	return DeploymentJobView{
		ID:                 item.ID,
		Name:               item.Name,
		Provider:           item.Provider,
		AccountID:          item.AccountID,
		AccountName:        accountName,
		BlueprintID:        item.BlueprintID,
		BlueprintName:      blueprintName,
		NetworkPlanID:      item.NetworkPlanID,
		NetworkPlan:        networkName,
		ProjectID:          item.ProjectID,
		EnvironmentID:      item.EnvironmentID,
		StackID:            item.StackID,
		JobType:            item.JobType,
		Status:             item.Status,
		Action:             item.Action,
		RunnerName:         item.RunnerName,
		WorkspacePath:      item.WorkspacePath,
		RetryOfJobID:       item.RetryOfJobID,
		Input:              input,
		PlanSummary:        planSummary,
		Output:             output,
		ErrorMessage:       item.ErrorMessage,
		LogExcerpt:         item.LogExcerpt,
		ResourceSyncStatus: item.ResourceSyncStatus,
		ResourceSyncWorker: item.ResourceSyncWorker,
		ResourceSyncError:  item.ResourceSyncError,
		ResourceSyncedAt:   item.ResourceSyncedAt,
		ResourceCount:      resourceCount,
		Resources:          nil,
		StartedAt:          item.StartedAt,
		EndedAt:            item.EndedAt,
		HeartbeatAt:        item.HeartbeatAt,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
}

func resourceToView(item DeploymentResource) DeploymentResourceView {
	var metadata any = map[string]any{}
	if strings.TrimSpace(item.MetadataJSON) != "" {
		_ = json.Unmarshal([]byte(item.MetadataJSON), &metadata)
	}
	return DeploymentResourceView{
		ID:           item.ID,
		JobID:        item.JobID,
		ResourceType: item.ResourceType,
		ResourceName: item.ResourceName,
		CloudID:      item.CloudID,
		Metadata:     metadata,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func cloudResourceToView(item CloudResource, accountName, blueprintName, networkName string) CloudResourceView {
	var metadata any = map[string]any{}
	if strings.TrimSpace(item.MetadataJSON) != "" {
		_ = json.Unmarshal([]byte(item.MetadataJSON), &metadata)
	}
	return CloudResourceView{
		ID:                      item.ID,
		Provider:                item.Provider,
		AccountID:               item.AccountID,
		AccountName:             accountName,
		BlueprintID:             item.BlueprintID,
		BlueprintName:           blueprintName,
		NetworkPlanID:           item.NetworkPlanID,
		NetworkPlan:             networkName,
		ProjectID:               item.ProjectID,
		EnvironmentID:           item.EnvironmentID,
		StackID:                 item.StackID,
		SourceJobID:             item.SourceJobID,
		Category:                item.Category,
		Region:                  item.Region,
		ResourceType:            item.ResourceType,
		ResourceName:            item.ResourceName,
		CloudID:                 item.CloudID,
		ResourceRole:            item.ResourceRole,
		OwnershipMode:           item.OwnershipMode,
		MachineEnrollmentStatus: item.MachineEnrollmentStatus,
		MachineEnrollmentWorker: item.MachineEnrollmentWorker,
		MachineEnrollmentError:  item.MachineEnrollmentError,
		MachineEnrolledAt:       item.MachineEnrolledAt,
		ClusterEnrollmentStatus: item.ClusterEnrollmentStatus,
		ClusterEnrollmentWorker: item.ClusterEnrollmentWorker,
		ClusterEnrollmentError:  item.ClusterEnrollmentError,
		ClusterEnrolledAt:       item.ClusterEnrolledAt,
		LifecycleState:          item.LifecycleState,
		Metadata:                metadata,
		LastSyncedAt:            item.LastSyncedAt,
		CreatedAt:               item.CreatedAt,
		UpdatedAt:               item.UpdatedAt,
	}
}

func buildNetworkBrief(provider string, topology any) string {
	raw, ok := topology.(map[string]any)
	if !ok {
		return "Foundation Network"
	}
	roleSummary := "-"
	if groups, ok := raw["subnet_groups"].([]any); ok && len(groups) > 0 {
		roles := make([]string, 0, len(groups))
		for _, item := range groups {
			group, ok := item.(map[string]any)
			if !ok {
				continue
			}
			role := strings.TrimSpace(fmt.Sprint(group["role"]))
			tier := strings.TrimSpace(fmt.Sprint(group["tier"]))
			trafficProfile := strings.TrimSpace(fmt.Sprint(group["traffic_profile"]))
			if role == "" {
				continue
			}
			if strings.EqualFold(provider, "alicloud") {
				if trafficProfile == "" {
					trafficProfile = "workload"
				}
				roles = append(roles, fmt.Sprintf("%s(%s)", role, trafficProfile))
				continue
			}
			roles = append(roles, fmt.Sprintf("%s(%s)", role, tier))
		}
		if len(roles) > 0 {
			roleSummary = strings.Join(roles, " / ")
		}
	}
	if strings.EqualFold(provider, "alicloud") {
		return fmt.Sprintf("可用区 %v / NAT %v / 交换机 %s", raw["availability_zones"], raw["nat_gateway_count"], roleSummary)
	}
	return fmt.Sprintf("AZ %v / NAT %v / 子网 %s", raw["availability_zones"], raw["nat_gateway_count"], roleSummary)
}
