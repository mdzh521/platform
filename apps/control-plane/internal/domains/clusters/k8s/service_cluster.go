package k8s

import (
	"backend-center/internal/domains/shared/scopeutil"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

func (s *Service) ListClusters() ([]ClusterListItem, error) {
	return withCachedJSON(context.Background(), s.cache, s.clusterListCacheKey(), clusterListCacheTTL, func() ([]ClusterListItem, error) {
		var clusters []Cluster
		if err := s.db.Order("id asc").Find(&clusters).Error; err != nil {
			return nil, err
		}

		items := make([]ClusterListItem, 0, len(clusters))
		for _, item := range clusters {
			var namespaceCount int64
			if err := s.db.Model(&Namespace{}).Where("cluster_id = ?", item.ID).Count(&namespaceCount).Error; err != nil {
				return nil, err
			}
			items = append(items, sanitizeCluster(item, namespaceCount))
		}
		return items, nil
	})
}

func (s *Service) CreateCluster(input ClusterInput) (*ClusterListItem, error) {
	if err := scopeutil.Validate(s.db, scopeutil.ScopeInput{
		ProjectID:           input.ProjectID,
		EnvironmentID:       input.EnvironmentID,
		StackID:             input.StackID,
		FoundationNetworkID: input.FoundationNetworkID,
		Provider:            input.Provider,
	}); err != nil {
		return nil, err
	}
	model := Cluster{
		Name:                input.Name,
		Code:                input.Code,
		Environment:         input.Environment,
		ProjectID:           input.ProjectID,
		EnvironmentID:       input.EnvironmentID,
		StackID:             input.StackID,
		FoundationNetworkID: input.FoundationNetworkID,
		Provider:            input.Provider,
		APIEndpoint:         input.APIEndpoint,
		AuthType:            input.AuthType,
		CredentialJSON:      encodeCredential(input),
		SourceResourceID:    input.SourceResourceID,
		ClusterAccessMode:   defaultString(input.AccessMode, "direct"),
		Version:             strings.TrimSpace(input.Version),
		VPCID:               strings.TrimSpace(input.VPCID),
		SubnetRefsJSON:      encodeStringSlice(input.SubnetRefs),
		Description:         input.Description,
		Status:              "draft",
	}
	if err := s.db.Create(&model).Error; err != nil {
		return nil, err
	}
	s.invalidateClusterCaches(context.Background(), model.ID)
	item := sanitizeCluster(model, 0)
	return &item, nil
}

func (s *Service) UpdateCluster(id uint, input ClusterInput) (*ClusterListItem, error) {
	var current Cluster
	if err := s.db.First(&current, id).Error; err != nil {
		return nil, err
	}
	if err := scopeutil.Validate(s.db, scopeutil.ScopeInput{
		ProjectID:           input.ProjectID,
		EnvironmentID:       input.EnvironmentID,
		StackID:             input.StackID,
		FoundationNetworkID: input.FoundationNetworkID,
		Provider:            input.Provider,
	}); err != nil {
		return nil, err
	}
	credentialJSON := current.CredentialJSON
	if strings.TrimSpace(input.Credential) != "" {
		credentialJSON = encodeCredential(input)
	}
	if err := s.db.Model(&current).Updates(map[string]any{
		"name":                  input.Name,
		"code":                  input.Code,
		"environment":           input.Environment,
		"project_id":            input.ProjectID,
		"environment_id":        input.EnvironmentID,
		"stack_id":              input.StackID,
		"foundation_network_id": input.FoundationNetworkID,
		"provider":              input.Provider,
		"api_endpoint":          input.APIEndpoint,
		"auth_type":             input.AuthType,
		"credential_json":       credentialJSON,
		"source_resource_id":    input.SourceResourceID,
		"cluster_access_mode":   defaultString(input.AccessMode, "direct"),
		"version":               strings.TrimSpace(input.Version),
		"vpc_id":                strings.TrimSpace(input.VPCID),
		"subnet_refs_json":      encodeStringSlice(input.SubnetRefs),
		"description":           input.Description,
	}).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&current, id).Error; err != nil {
		return nil, err
	}
	s.invalidateClusterCaches(context.Background(), current.ID)
	var namespaceCount int64
	if err := s.db.Model(&Namespace{}).Where("cluster_id = ?", current.ID).Count(&namespaceCount).Error; err != nil {
		return nil, err
	}
	item := sanitizeCluster(current, namespaceCount)
	return &item, nil
}

func (s *Service) DeleteCluster(id uint) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("cluster_id = ?", id).Delete(&Namespace{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Cluster{}, id).Error
	})
	if err == nil {
		s.invalidateClusterCaches(context.Background(), id)
	}
	return err
}

func (s *Service) TestCluster(id uint) (map[string]any, error) {
	var cluster Cluster
	if err := s.db.First(&cluster, id).Error; err != nil {
		return nil, err
	}

	client, err := newClusterClient(cluster)
	if err != nil {
		_ = s.db.Model(&Cluster{}).Where("id = ?", id).Update("status", "error").Error
		return nil, err
	}
	version, err := client.Version()
	if err != nil {
		_ = s.db.Model(&Cluster{}).Where("id = ?", id).Update("status", "error").Error
		return nil, err
	}
	now := time.Now()
	if err := s.db.Model(&Cluster{}).Where("id = ?", id).Updates(map[string]any{
		"status":          "ready",
		"last_checked_at": &now,
	}).Error; err != nil {
		return nil, err
	}
	s.invalidateClusterCaches(context.Background(), id)
	return map[string]any{
		"cluster_id":     id,
		"result":         "ok",
		"server_version": version,
		"message":        "kubernetes api connection succeeded",
	}, nil
}

func (s *Service) GetClusterOverview(id uint) (map[string]any, error) {
	return withCachedJSON(context.Background(), s.cache, s.clusterOverviewCacheKey(id), clusterOverviewCacheTTL, func() (map[string]any, error) {
		var cluster Cluster
		if err := s.db.First(&cluster, id).Error; err != nil {
			return nil, err
		}

		client, err := newClusterClient(cluster)
		if err != nil {
			return nil, err
		}
		nodes, err := client.ListNodes()
		if err != nil {
			return nil, err
		}
		events, err := client.ListRecentClusterEvents(20)
		if err != nil {
			return nil, err
		}
		namespaces, err := client.ListNamespaceGovernance()
		if err != nil {
			return nil, err
		}

		nodeItems := make([]ClusterOverviewNode, 0, len(nodes))
		readyNodes := 0
		for _, item := range nodes {
			if item.Ready {
				readyNodes++
			}
			nodeItems = append(nodeItems, ClusterOverviewNode{
				Name:           item.Name,
				Ready:          item.Ready,
				Roles:          item.Roles,
				InternalIP:     item.InternalIP,
				PodCIDR:        item.PodCIDR,
				KubeletVersion: item.KubeletVersion,
				OSImage:        item.OSImage,
				CreatedAt:      item.CreatedAt,
			})
		}
		eventItems := make([]ClusterOverviewEvent, 0, len(events))
		for _, item := range events {
			eventItems = append(eventItems, ClusterOverviewEvent{
				Type:         item.Type,
				Namespace:    item.Namespace,
				InvolvedKind: item.InvolvedKind,
				InvolvedName: item.InvolvedName,
				Reason:       item.Reason,
				Message:      item.Message,
				Component:    item.Component,
				Count:        item.Count,
				Timestamp:    item.Timestamp,
			})
		}
		namespaceItems := make([]NamespaceGovernanceSummary, 0, len(namespaces))
		withQuota := 0
		withLimits := 0
		for _, item := range namespaces {
			if item.ResourceQuotaCount > 0 {
				withQuota++
			}
			if item.LimitRangeCount > 0 {
				withLimits++
			}
			namespaceItems = append(namespaceItems, NamespaceGovernanceSummary{
				Name:               item.Name,
				Status:             item.Status,
				ResourceQuotaCount: item.ResourceQuotaCount,
				LimitRangeCount:    item.LimitRangeCount,
			})
		}

		return map[string]any{
			"cluster": map[string]any{
				"id":                    cluster.ID,
				"name":                  cluster.Name,
				"code":                  cluster.Code,
				"environment":           cluster.Environment,
				"project_id":            cluster.ProjectID,
				"environment_id":        cluster.EnvironmentID,
				"stack_id":              cluster.StackID,
				"foundation_network_id": cluster.FoundationNetworkID,
				"provider":              cluster.Provider,
				"api_endpoint":          cluster.APIEndpoint,
				"source_resource_id":    cluster.SourceResourceID,
				"access_mode":           cluster.ClusterAccessMode,
				"version":               cluster.Version,
				"vpc_id":                cluster.VPCID,
				"subnet_refs":           parseStringSliceJSON(cluster.SubnetRefsJSON),
				"status":                cluster.Status,
				"last_checked_at":       cluster.LastCheckedAt,
				"last_synced_at":        cluster.LastSyncedAt,
			},
			"summary": map[string]any{
				"node_count":                  len(nodeItems),
				"ready_node_count":            readyNodes,
				"recent_event_count":          len(eventItems),
				"namespace_count":             len(namespaceItems),
				"namespaces_with_quota":       withQuota,
				"namespaces_with_limit_range": withLimits,
			},
			"nodes":                nodeItems,
			"recent_events":        eventItems,
			"namespace_governance": namespaceItems,
			"cached_at":            time.Now().UTC().Format(time.RFC3339),
		}, nil
	})
}

func (s *Service) SyncNamespaces(id uint) (map[string]any, error) {
	var cluster Cluster
	if err := s.db.First(&cluster, id).Error; err != nil {
		return nil, err
	}

	client, err := newClusterClient(cluster)
	if err != nil {
		return nil, err
	}
	items, err := client.ListNamespaces()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		seen := make(map[string]struct{}, len(items))
		for _, item := range items {
			seen[item.Name] = struct{}{}
			labelsJSON, _ := json.Marshal(item.Labels)
			record := Namespace{ClusterID: cluster.ID, Name: item.Name}
			if err := tx.Where(Namespace{ClusterID: cluster.ID, Name: item.Name}).Assign(Namespace{
				ClusterID:   cluster.ID,
				Name:        item.Name,
				DisplayName: item.Name,
				Description: "Synced from Kubernetes API",
				LabelsJSON:  string(labelsJSON),
				SourceType:  "sync",
				Status:      "active",
			}).FirstOrCreate(&record).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&Cluster{}).Where("id = ?", cluster.ID).
			Update("last_synced_at", now).Error; err != nil {
			return err
		}
		if err := tx.Model(&Cluster{}).Where("id = ?", cluster.ID).
			Update("status", "ready").Error; err != nil {
			return err
		}
		var synced []Namespace
		if err := tx.Where("cluster_id = ? AND source_type = ?", cluster.ID, "sync").Find(&synced).Error; err != nil {
			return err
		}
		for _, item := range synced {
			if _, ok := seen[item.Name]; ok {
				continue
			}
			if err := tx.Delete(&Namespace{}, item.ID).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	s.invalidateClusterCaches(context.Background(), cluster.ID)
	s.publishEvent(context.Background(), "k8s.sync.namespaces", map[string]any{
		"cluster_id":        id,
		"cluster_name":      cluster.Name,
		"synced_namespaces": len(items),
		"provider":          cluster.Provider,
		"environment":       cluster.Environment,
	})

	return map[string]any{
		"cluster_id":        id,
		"synced_namespaces": len(items),
		"message":           "namespace sync completed",
	}, nil
}

func (s *Service) SyncWorkloads(id uint) (map[string]any, error) {
	var cluster Cluster
	if err := s.db.First(&cluster, id).Error; err != nil {
		return nil, err
	}

	client, err := newClusterClient(cluster)
	if err != nil {
		return nil, err
	}
	items, err := client.ListWorkloads()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		seen := make(map[string]struct{}, len(items))
		for _, item := range items {
			key := item.NamespaceName + ":" + item.Kind + ":" + item.Name
			seen[key] = struct{}{}

			var namespace Namespace
			var namespaceID *uint
			if err := tx.Where("cluster_id = ? AND name = ?", cluster.ID, item.NamespaceName).First(&namespace).Error; err == nil {
				namespaceID = &namespace.ID
			}

			attributes := map[string]any{
				"cluster_id":     cluster.ID,
				"namespace_id":   namespaceID,
				"namespace_name": item.NamespaceName,
				"kind":           item.Kind,
				"name":           item.Name,
				"ready_replicas": item.ReadyReplicas,
				"replicas":       item.Replicas,
				"image":          item.Image,
				"status":         item.Status,
				"source_type":    "sync",
			}

			var record Workload
			err := tx.Where("cluster_id = ? AND namespace_name = ? AND kind = ? AND name = ?", cluster.ID, item.NamespaceName, item.Kind, item.Name).First(&record).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				record = Workload{
					ClusterID:     cluster.ID,
					NamespaceID:   namespaceID,
					NamespaceName: item.NamespaceName,
					Kind:          item.Kind,
					Name:          item.Name,
					ReadyReplicas: item.ReadyReplicas,
					Replicas:      item.Replicas,
					Image:         item.Image,
					Status:        item.Status,
					SourceType:    "sync",
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			if err := tx.Model(&record).Updates(attributes).Error; err != nil {
				return err
			}
		}

		var existing []Workload
		if err := tx.Where("cluster_id = ?", cluster.ID).Find(&existing).Error; err != nil {
			return err
		}
		for _, item := range existing {
			key := item.NamespaceName + ":" + item.Kind + ":" + item.Name
			if _, ok := seen[key]; ok {
				continue
			}
			if err := tx.Delete(&Workload{}, item.ID).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&Cluster{}).Where("id = ?", cluster.ID).
			Update("last_synced_at", now).Error; err != nil {
			return err
		}
		if err := tx.Model(&Cluster{}).Where("id = ?", cluster.ID).
			Update("status", "ready").Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	s.invalidateClusterCaches(context.Background(), cluster.ID)
	s.publishEvent(context.Background(), "k8s.sync.workloads", map[string]any{
		"cluster_id":       id,
		"cluster_name":     cluster.Name,
		"synced_workloads": len(items),
		"provider":         cluster.Provider,
		"environment":      cluster.Environment,
	})

	return map[string]any{
		"cluster_id":       id,
		"synced_workloads": len(items),
		"message":          "workload sync completed",
	}, nil
}

func (s *Service) ListNamespaces(clusterID *uint) ([]NamespaceListItem, error) {
	query := s.db.Table("namespaces").
		Select("namespaces.id, namespaces.cluster_id, clusters.name as cluster_name, namespaces.name, namespaces.display_name, namespaces.description, namespaces.source_type, namespaces.status, namespaces.created_at, namespaces.updated_at").
		Joins("join clusters on clusters.id = namespaces.cluster_id").
		Order("namespaces.id asc")
	if clusterID != nil && *clusterID > 0 {
		query = query.Where("namespaces.cluster_id = ?", *clusterID)
	}

	var items []NamespaceListItem
	if err := query.Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) CreateNamespace(input NamespaceInput) (*Namespace, error) {
	model := &Namespace{
		ClusterID:   input.ClusterID,
		Name:        input.Name,
		DisplayName: input.DisplayName,
		Description: input.Description,
		SourceType:  defaultString(input.SourceType, "manual"),
		Status:      defaultString(input.Status, "active"),
	}
	if err := s.db.Create(model).Error; err != nil {
		return nil, err
	}
	s.invalidateClusterCaches(context.Background(), model.ClusterID)
	return model, nil
}

func (s *Service) UpdateNamespace(id uint, input NamespaceInput) (*Namespace, error) {
	if err := s.db.Model(&Namespace{}).Where("id = ?", id).Updates(map[string]any{
		"cluster_id":   input.ClusterID,
		"name":         input.Name,
		"display_name": input.DisplayName,
		"description":  input.Description,
		"source_type":  defaultString(input.SourceType, "manual"),
		"status":       defaultString(input.Status, "active"),
	}).Error; err != nil {
		return nil, err
	}
	var item Namespace
	if err := s.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	s.invalidateClusterCaches(context.Background(), item.ClusterID)
	return &item, nil
}

func (s *Service) DeleteNamespace(id uint) error {
	var item Namespace
	if err := s.db.First(&item, id).Error; err != nil {
		return err
	}
	if err := s.db.Delete(&Namespace{}, id).Error; err != nil {
		return err
	}
	s.invalidateClusterCaches(context.Background(), item.ClusterID)
	return nil
}

func (s *Service) resolveClusterClient(id uint) (*Cluster, *clusterClient, error) {
	var cluster Cluster
	if err := s.db.First(&cluster, id).Error; err != nil {
		return nil, nil, err
	}
	client, err := newClusterClient(cluster)
	if err != nil {
		return nil, nil, err
	}
	return &cluster, client, nil
}
