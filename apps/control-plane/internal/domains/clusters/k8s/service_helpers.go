package k8s

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

func stringMapToAny(items map[string]string) map[string]any {
	out := make(map[string]any, len(items))
	for key, value := range items {
		out[key] = value
	}
	return out
}

func cloneShallowMap(items map[string]any) map[string]any {
	out := make(map[string]any, len(items))
	for key, value := range items {
		out[key] = value
	}
	return out
}

func normalizeIntOrString(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasSuffix(raw, "%") {
		return raw
	}
	if value, err := strconv.Atoi(raw); err == nil {
		return value
	}
	return raw
}

func coalesceString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sanitizeCluster(item Cluster, namespaceCount int64) ClusterListItem {
	return ClusterListItem{
		ID:                  item.ID,
		Name:                item.Name,
		Code:                item.Code,
		Environment:         item.Environment,
		ProjectID:           item.ProjectID,
		EnvironmentID:       item.EnvironmentID,
		StackID:             item.StackID,
		FoundationNetworkID: item.FoundationNetworkID,
		Provider:            item.Provider,
		APIEndpoint:         item.APIEndpoint,
		AuthType:            item.AuthType,
		SourceResourceID:    item.SourceResourceID,
		AccessMode:          defaultString(item.ClusterAccessMode, "direct"),
		Version:             item.Version,
		VPCID:               item.VPCID,
		SubnetRefs:          parseStringSliceJSON(item.SubnetRefsJSON),
		Description:         item.Description,
		Status:              item.Status,
		HasCredential:       item.CredentialJSON != "",
		NamespaceCount:      namespaceCount,
		LastCheckedAt:       item.LastCheckedAt,
		LastSyncedAt:        item.LastSyncedAt,
		ServerVersion:       "",
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
}

func parseStringSliceJSON(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}
	return items
}

func encodeStringSlice(items []string) string {
	raw, _ := json.Marshal(items)
	return string(raw)
}

func (s *Service) refreshWorkloadState(clusterID uint, kind, namespaceName, name string) error {
	var cluster Cluster
	if err := s.db.First(&cluster, clusterID).Error; err != nil {
		return err
	}
	client, err := newClusterClient(cluster)
	if err != nil {
		return err
	}
	allItems, err := client.ListWorkloads()
	if err != nil {
		return err
	}
	for _, item := range allItems {
		if item.Kind != kind || item.NamespaceName != namespaceName || item.Name != name {
			continue
		}
		var namespace Namespace
		var namespaceID *uint
		if err := s.db.Where("cluster_id = ? AND name = ?", cluster.ID, item.NamespaceName).First(&namespace).Error; err == nil {
			namespaceID = &namespace.ID
		}
		return s.db.Model(&Workload{}).
			Where("cluster_id = ? AND namespace_name = ? AND kind = ? AND name = ?", cluster.ID, item.NamespaceName, item.Kind, item.Name).
			Updates(map[string]any{
				"namespace_id":   namespaceID,
				"ready_replicas": item.ReadyReplicas,
				"replicas":       item.Replicas,
				"image":          item.Image,
				"status":         item.Status,
				"source_type":    "sync",
			}).Error
	}
	return s.db.Where("cluster_id = ? AND namespace_name = ? AND kind = ? AND name = ?", cluster.ID, namespaceName, kind, name).Delete(&Workload{}).Error
}

func (s *Service) handleMissingRemoteWorkload(workload *Workload, err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "404") || strings.Contains(strings.ToLower(err.Error()), "not found") {
		_ = s.db.Delete(&Workload{}, workload.ID).Error
		return gorm.ErrRecordNotFound
	}
	return err
}

func encodeCredential(input ClusterInput) string {
	if strings.TrimSpace(input.Credential) == "" {
		return ""
	}
	payload, _ := json.Marshal(map[string]string{
		"auth_type":  input.AuthType,
		"credential": input.Credential,
	})
	return string(payload)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func selectLogPod(pods []relatedPodItem, podName string) (relatedPodItem, error) {
	if strings.TrimSpace(podName) != "" {
		for _, item := range pods {
			if item.Name == podName {
				return item, nil
			}
		}
		return relatedPodItem{}, errors.New("selected pod not found in current workload")
	}
	return chooseLogPod(pods), nil
}

func (s *Service) resolveWorkloadClient(id uint) (*Workload, *Cluster, *clusterClient, error) {
	var workload Workload
	if err := s.db.First(&workload, id).Error; err != nil {
		return nil, nil, nil, err
	}
	var cluster Cluster
	if err := s.db.First(&cluster, workload.ClusterID).Error; err != nil {
		return nil, nil, nil, err
	}
	client, err := newClusterClient(cluster)
	if err != nil {
		return nil, nil, nil, err
	}
	return &workload, &cluster, client, nil
}

func chooseLogPod(pods []relatedPodItem) relatedPodItem {
	for _, item := range pods {
		if item.Status == "Running" {
			return item
		}
	}
	return pods[0]
}

func compactManifest(object map[string]any) map[string]any {
	compact := cloneMap(object)
	delete(compact, "status")
	metadata, _ := compact["metadata"].(map[string]any)
	if metadata != nil {
		delete(metadata, "managedFields")
		delete(metadata, "resourceVersion")
		delete(metadata, "uid")
		delete(metadata, "selfLink")
		delete(metadata, "generation")
		delete(metadata, "creationTimestamp")
		delete(metadata, "annotations")
		if len(metadata) == 0 {
			delete(compact, "metadata")
		}
	}
	return compact
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = cloneMap(typed)
		case []any:
			out[key] = cloneSlice(typed)
		default:
			out[key] = value
		}
	}
	return out
}

func cloneSlice(input []any) []any {
	out := make([]any, 0, len(input))
	for _, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			out = append(out, cloneMap(typed))
		case []any:
			out = append(out, cloneSlice(typed))
		default:
			out = append(out, value)
		}
	}
	return out
}
