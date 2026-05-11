package k8s

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

func newClusterClient(cluster Cluster) (*clusterClient, error) {
	switch cluster.AuthType {
	case "token":
		credential, err := parseTokenCredential(cluster)
		if err != nil {
			return nil, err
		}
		httpClient, err := buildHTTPClient(credential.CACert, credential.InsecureSkipVerify, nil, nil)
		if err != nil {
			return nil, err
		}
		return &clusterClient{
			baseURL:    strings.TrimRight(cluster.APIEndpoint, "/"),
			httpClient: httpClient,
			token:      credential.Token,
		}, nil
	case "kubeconfig":
		return buildFromKubeconfig(cluster)
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", cluster.AuthType)
	}
}

func (c *clusterClient) Version() (string, error) {
	var payload versionPayload
	if err := c.getJSON("/version", &payload); err != nil {
		return "", err
	}
	return payload.GitVersion, nil
}

func (c *clusterClient) ListNamespaces() ([]namespaceRemoteItem, error) {
	var payload namespacesPayload
	if err := c.getJSON("/api/v1/namespaces", &payload); err != nil {
		return nil, err
	}
	items := make([]namespaceRemoteItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, namespaceRemoteItem{
			Name:   item.Metadata.Name,
			Labels: item.Metadata.Labels,
		})
	}
	return items, nil
}

func (c *clusterClient) CreateObjects(objects []map[string]any) error {
	for _, object := range objects {
		if err := c.CreateObject(object); err != nil {
			return err
		}
	}
	return nil
}

func (c *clusterClient) CreateObject(object map[string]any) error {
	path, err := createResourcePath(object)
	if err != nil {
		return err
	}
	body, err := json.Marshal(object)
	if err != nil {
		return err
	}
	return c.doJSON(http.MethodPost, path, "application/json", body, nil)
}

func (c *clusterClient) ListNodes() ([]clusterNodeItem, error) {
	var payload nodesPayload
	if err := c.getJSON("/api/v1/nodes", &payload); err != nil {
		return nil, err
	}
	items := make([]clusterNodeItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		roles := make([]string, 0)
		for key := range item.Metadata.Labels {
			if strings.HasPrefix(key, "node-role.kubernetes.io/") {
				role := strings.TrimPrefix(key, "node-role.kubernetes.io/")
				if role == "" {
					role = "worker"
				}
				roles = append(roles, role)
			}
		}
		slices.Sort(roles)
		ready := false
		for _, condition := range item.Status.Conditions {
			if condition.Type == "Ready" {
				ready = condition.Status == "True"
				break
			}
		}
		internalIP := ""
		for _, address := range item.Status.Addresses {
			if address.Type == "InternalIP" {
				internalIP = address.Address
				break
			}
		}
		items = append(items, clusterNodeItem{
			Name:           item.Metadata.Name,
			Ready:          ready,
			Roles:          roles,
			InternalIP:     internalIP,
			PodCIDR:        item.Spec.PodCIDR,
			KubeletVersion: item.Status.NodeInfo.KubeletVersion,
			OSImage:        item.Status.NodeInfo.OSImage,
			CreatedAt:      item.Metadata.CreationTimestamp,
		})
	}
	slices.SortFunc(items, func(a, b clusterNodeItem) int {
		return strings.Compare(a.Name, b.Name)
	})
	return items, nil
}

func (c *clusterClient) ListRecentClusterEvents(limit int) ([]clusterEventItem, error) {
	var payload eventsPayload
	if err := c.getJSON("/api/v1/events", &payload); err != nil {
		return nil, err
	}
	items := make([]clusterEventItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		timestamp := firstNonEmpty(item.EventTime, item.LastTimestamp, item.FirstTimestamp)
		namespace := firstNonEmpty(item.InvolvedObject.Namespace, item.Metadata.Namespace, "-")
		items = append(items, clusterEventItem{
			Type:         item.Type,
			Namespace:    namespace,
			InvolvedKind: item.InvolvedObject.Kind,
			InvolvedName: item.InvolvedObject.Name,
			Reason:       item.Reason,
			Message:      item.Message,
			Component:    item.Source.Component,
			Count:        item.Count,
			Timestamp:    timestamp,
		})
	}
	slices.SortFunc(items, func(a, b clusterEventItem) int {
		return strings.Compare(b.Timestamp, a.Timestamp)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (c *clusterClient) ListNamespaceGovernance() ([]namespaceGovernanceItem, error) {
	var payload namespacesOverviewPayload
	if err := c.getJSON("/api/v1/namespaces", &payload); err != nil {
		return nil, err
	}
	items := make([]namespaceGovernanceItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		quotaCount, err := c.getNamespaceResourceCount(item.Metadata.Name, "resourcequotas")
		if err != nil {
			return nil, err
		}
		limitRangeCount, err := c.getNamespaceResourceCount(item.Metadata.Name, "limitranges")
		if err != nil {
			return nil, err
		}
		items = append(items, namespaceGovernanceItem{
			Name:               item.Metadata.Name,
			Status:             firstNonEmpty(item.Status.Phase, "-"),
			ResourceQuotaCount: quotaCount,
			LimitRangeCount:    limitRangeCount,
		})
	}
	slices.SortFunc(items, func(a, b namespaceGovernanceItem) int {
		return strings.Compare(a.Name, b.Name)
	})
	return items, nil
}
