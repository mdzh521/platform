package graph

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"backend-center/internal/domains/delivery/cloud"
	"backend-center/internal/domains/clusters/k8s"
	"backend-center/internal/domains/machines/machine"
	"backend-center/internal/domains/projects/project"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

type NodeView struct {
	Key      string         `json:"key"`
	Type     string         `json:"type"`
	RefID    string         `json:"ref_id"`
	Name     string         `json:"name"`
	Status   string         `json:"status"`
	Metadata map[string]any `json:"metadata"`
}

type EdgeView struct {
	Key      string         `json:"key"`
	From     string         `json:"from"`
	To       string         `json:"to"`
	Relation string         `json:"relation"`
	Metadata map[string]any `json:"metadata"`
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetProjectGraph(projectID uint) (map[string]any, error) {
	if err := s.rebuildProjectGraph(projectID); err != nil {
		return nil, err
	}
	var projectRecord project.Project
	if err := s.db.First(&projectRecord, projectID).Error; err != nil {
		return nil, err
	}
	scopeRef := strconv.FormatUint(uint64(projectID), 10)
	var nodes []Node
	if err := s.db.Where("scope_kind = ? AND scope_ref = ?", "project", scopeRef).Order("node_type asc, id asc").Find(&nodes).Error; err != nil {
		return nil, err
	}
	var edges []Edge
	if err := s.db.Where("scope_kind = ? AND scope_ref = ?", "project", scopeRef).Order("relation asc, id asc").Find(&edges).Error; err != nil {
		return nil, err
	}
	nodeViews := make([]NodeView, 0, len(nodes))
	for _, item := range nodes {
		nodeViews = append(nodeViews, NodeView{
			Key:      item.NodeKey,
			Type:     item.NodeType,
			RefID:    item.RefID,
			Name:     item.Name,
			Status:   item.Status,
			Metadata: decodeGraphMetadata(item.MetadataJSON),
		})
	}
	edgeViews := make([]EdgeView, 0, len(edges))
	for _, item := range edges {
		edgeViews = append(edgeViews, EdgeView{
			Key:      item.EdgeKey,
			From:     item.FromNodeKey,
			To:       item.ToNodeKey,
			Relation: item.Relation,
			Metadata: decodeGraphMetadata(item.MetadataJSON),
		})
	}
	return map[string]any{
		"project": map[string]any{
			"id":     projectRecord.ID,
			"code":   projectRecord.Code,
			"name":   projectRecord.Name,
			"status": projectRecord.Status,
		},
		"nodes": nodeViews,
		"edges": edgeViews,
	}, nil
}

func (s *Service) rebuildProjectGraph(projectID uint) error {
	scopeRef := strconv.FormatUint(uint64(projectID), 10)
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("scope_kind = ? AND scope_ref = ?", "project", scopeRef).Delete(&Edge{}).Error; err != nil {
			return err
		}
		if err := tx.Where("scope_kind = ? AND scope_ref = ?", "project", scopeRef).Delete(&Node{}).Error; err != nil {
			return err
		}

		var projectRecord project.Project
		if err := tx.First(&projectRecord, projectID).Error; err != nil {
			return err
		}
		projectKey := nodeKey("project", projectRecord.ID)
		if err := tx.Create(&Node{
			ScopeKind: "project", ScopeRef: scopeRef, NodeKey: projectKey, NodeType: "project",
			RefID: strconv.FormatUint(uint64(projectRecord.ID), 10), Name: projectRecord.Name, Status: projectRecord.Status,
			MetadataJSON: encodeGraphMetadata(map[string]any{"code": projectRecord.Code, "business_line": projectRecord.BusinessLine}),
		}).Error; err != nil {
			return err
		}

		var environments []project.Environment
		if err := tx.Where("project_id = ?", projectID).Find(&environments).Error; err != nil {
			return err
		}
		for _, env := range environments {
			envKey := nodeKey("environment", env.ID)
			if err := tx.Create(&Node{
				ScopeKind: "project", ScopeRef: scopeRef, NodeKey: envKey, NodeType: "environment",
				RefID: strconv.FormatUint(uint64(env.ID), 10), Name: env.Name, Status: env.Status,
				MetadataJSON: encodeGraphMetadata(map[string]any{"code": env.Code, "kind": env.Kind}),
			}).Error; err != nil {
				return err
			}
			if err := tx.Create(&Edge{
				ScopeKind: "project", ScopeRef: scopeRef, EdgeKey: edgeKey(projectKey, envKey, "contains"),
				FromNodeKey: projectKey, ToNodeKey: envKey, Relation: "contains",
			}).Error; err != nil {
				return err
			}
		}

		var stacks []project.Stack
		if err := tx.Where("project_id = ?", projectID).Find(&stacks).Error; err != nil {
			return err
		}
		for _, stack := range stacks {
			stackKey := nodeKey("stack", stack.ID)
			if err := tx.Create(&Node{
				ScopeKind: "project", ScopeRef: scopeRef, NodeKey: stackKey, NodeType: "stack",
				RefID: strconv.FormatUint(uint64(stack.ID), 10), Name: stack.Name, Status: stack.Status,
				MetadataJSON: encodeGraphMetadata(map[string]any{"stack_code": stack.StackCode, "stack_type": stack.StackType, "provider": stack.Provider, "region": stack.Region}),
			}).Error; err != nil {
				return err
			}
			envKey := nodeKey("environment", stack.EnvironmentID)
			if err := tx.Create(&Edge{
				ScopeKind: "project", ScopeRef: scopeRef, EdgeKey: edgeKey(envKey, stackKey, "contains"),
				FromNodeKey: envKey, ToNodeKey: stackKey, Relation: "contains",
			}).Error; err != nil {
				return err
			}
		}

		var foundations []cloud.NetworkPlan
		if err := tx.Where("project_id = ?", projectID).Find(&foundations).Error; err != nil {
			return err
		}
		for _, foundation := range foundations {
			foundationStatus, err := s.resolveFoundationGraphStatus(tx, foundation)
			if err != nil {
				return err
			}
			key := nodeKey("foundation-network", foundation.ID)
			if err := tx.Create(&Node{
				ScopeKind: "project", ScopeRef: scopeRef, NodeKey: key, NodeType: "foundation-network",
				RefID: strconv.FormatUint(uint64(foundation.ID), 10), Name: foundation.Name, Status: foundationStatus,
				MetadataJSON: encodeGraphMetadata(map[string]any{"provider": foundation.Provider, "region": foundation.Region, "stack_id": foundation.StackID}),
			}).Error; err != nil {
				return err
			}
			if foundation.StackID != nil {
				if err := tx.Create(&Edge{
					ScopeKind: "project", ScopeRef: scopeRef, EdgeKey: edgeKey(nodeKey("stack", *foundation.StackID), key, "owns"),
					FromNodeKey: nodeKey("stack", *foundation.StackID), ToNodeKey: key, Relation: "owns",
				}).Error; err != nil {
					return err
				}
			}
		}

		var assets []machine.Asset
		if err := tx.Where("project_id = ?", projectID).Find(&assets).Error; err != nil {
			return err
		}
		for _, asset := range assets {
			key := nodeKey("machine-asset", asset.ID)
			if err := tx.Create(&Node{
				ScopeKind: "project", ScopeRef: scopeRef, NodeKey: key, NodeType: "machine-asset",
				RefID: strconv.FormatUint(uint64(asset.ID), 10), Name: asset.Name, Status: asset.Status,
				MetadataJSON: encodeGraphMetadata(map[string]any{"address": asset.Address, "private_ip": asset.PrivateIP, "enrollment_status": asset.EnrollmentStatus, "is_gateway_node": asset.IsGatewayNode}),
			}).Error; err != nil {
				return err
			}
			if asset.StackID != nil {
				if err := tx.Create(&Edge{
					ScopeKind: "project", ScopeRef: scopeRef, EdgeKey: edgeKey(nodeKey("stack", *asset.StackID), key, "owns"),
					FromNodeKey: nodeKey("stack", *asset.StackID), ToNodeKey: key, Relation: "owns",
				}).Error; err != nil {
					return err
				}
			}
			if asset.FoundationNetworkID != nil {
				if err := tx.Create(&Edge{
					ScopeKind: "project", ScopeRef: scopeRef, EdgeKey: edgeKey(nodeKey("foundation-network", *asset.FoundationNetworkID), key, "connects"),
					FromNodeKey: nodeKey("foundation-network", *asset.FoundationNetworkID), ToNodeKey: key, Relation: "connects",
				}).Error; err != nil {
					return err
				}
			}
			if asset.GatewayAssetID != nil {
				if err := tx.Create(&Edge{
					ScopeKind: "project", ScopeRef: scopeRef, EdgeKey: edgeKey(nodeKey("machine-asset", *asset.GatewayAssetID), key, "gateway"),
					FromNodeKey: nodeKey("machine-asset", *asset.GatewayAssetID), ToNodeKey: key, Relation: "gateway",
				}).Error; err != nil {
					return err
				}
			}
		}

		var clusters []k8s.Cluster
		if err := tx.Where("project_id = ?", projectID).Find(&clusters).Error; err != nil {
			return err
		}
		for _, cluster := range clusters {
			key := nodeKey("k8s-cluster", cluster.ID)
			if err := tx.Create(&Node{
				ScopeKind: "project", ScopeRef: scopeRef, NodeKey: key, NodeType: "k8s-cluster",
				RefID: strconv.FormatUint(uint64(cluster.ID), 10), Name: cluster.Name, Status: cluster.Status,
				MetadataJSON: encodeGraphMetadata(map[string]any{"provider": cluster.Provider, "version": cluster.Version, "endpoint": cluster.APIEndpoint, "access_mode": cluster.ClusterAccessMode}),
			}).Error; err != nil {
				return err
			}
			if cluster.StackID != nil {
				if err := tx.Create(&Edge{
					ScopeKind: "project", ScopeRef: scopeRef, EdgeKey: edgeKey(nodeKey("stack", *cluster.StackID), key, "owns"),
					FromNodeKey: nodeKey("stack", *cluster.StackID), ToNodeKey: key, Relation: "owns",
				}).Error; err != nil {
					return err
				}
			}
			if cluster.FoundationNetworkID != nil {
				if err := tx.Create(&Edge{
					ScopeKind: "project", ScopeRef: scopeRef, EdgeKey: edgeKey(nodeKey("foundation-network", *cluster.FoundationNetworkID), key, "connects"),
					FromNodeKey: nodeKey("foundation-network", *cluster.FoundationNetworkID), ToNodeKey: key, Relation: "connects",
				}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *Service) resolveFoundationGraphStatus(tx *gorm.DB, foundation cloud.NetworkPlan) (string, error) {
	var job cloud.DeploymentJob
	err := tx.
		Where("network_plan_id = ?", foundation.ID).
		Order("created_at desc").
		First(&job).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return foundation.Status, nil
		}
		return "", err
	}
	switch strings.TrimSpace(strings.ToLower(job.Status)) {
	case "succeeded":
		if strings.TrimSpace(strings.ToLower(job.Action)) == "destroy" {
			return "destroyed", nil
		}
		return "succeeded", nil
	case "destroyed":
		return "destroyed", nil
	case "planned", "failed", "queued", "claimed", "planning", "applying", "destroying", "cancelled":
		return job.Status, nil
	default:
		return foundation.Status, nil
	}
}

func nodeKey(kind string, id uint) string {
	return fmt.Sprintf("%s:%d", strings.TrimSpace(kind), id)
}

func edgeKey(from, to, relation string) string {
	return fmt.Sprintf("%s>%s>%s", from, relation, to)
}

func encodeGraphMetadata(value map[string]any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func decodeGraphMetadata(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return map[string]any{}
	}
	return value
}
