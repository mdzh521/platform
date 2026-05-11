package k8s

import (
	"encoding/json"
	"errors"
	"strings"

	"gopkg.in/yaml.v3"
)

func (s *Service) ListNamespaceResources(clusterID uint, namespace, kind string) (map[string]any, error) {
	cluster, client, err := s.resolveClusterClient(clusterID)
	if err != nil {
		return nil, err
	}
	items, err := client.ListNamespaceResources(namespace, kind)
	if err != nil {
		return nil, err
	}
	results := make([]ResourceListItem, 0, len(items))
	for _, item := range items {
		results = append(results, ResourceListItem(item))
	}
	return map[string]any{
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    namespace,
		"kind":         kind,
		"items":        results,
	}, nil
}

func (s *Service) GetNamespaceResourceDetail(clusterID uint, namespace, kind, name string) (map[string]any, error) {
	cluster, client, err := s.resolveClusterClient(clusterID)
	if err != nil {
		return nil, err
	}
	var detail map[string]any
	switch kind {
	case "ServiceAccount":
		item, err := client.GetServiceAccountDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		detail = map[string]any{"service_account": ServiceAccountDetail{
			Name:         item.Name,
			Secrets:      item.Secrets,
			ImagePullRef: item.ImagePullSecrets,
			ReferencedBy: item.ReferencedBy,
			Annotations:  item.Annotations,
			CreatedAt:    item.CreatedAt,
		}}
	case "PersistentVolumeClaim":
		item, err := client.GetPersistentVolumeClaimDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		detail = map[string]any{"persistent_volume_claim": PersistentVolumeClaimDetail{
			Name:         item.Name,
			Status:       item.Status,
			Volume:       item.VolumeName,
			StorageClass: item.StorageClassName,
			AccessModes:  item.AccessModes,
			Requested:    item.RequestedStorage,
			Capacity:     item.Capacity,
			MountedBy:    item.MountedBy,
			Annotations:  item.Annotations,
			CreatedAt:    item.CreatedAt,
		}}
	case "Service":
		item, err := client.GetServiceDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		detail = map[string]any{"service": ServiceDetail{
			Name:            item.Name,
			Type:            item.Type,
			ClusterIP:       item.ClusterIP,
			SessionAffinity: item.SessionAffinity,
			Selector:        item.Selector,
			ExternalIPs:     item.ExternalIPs,
			Ports:           item.Ports,
			Endpoints:       item.Endpoints,
			SelectedBy:      item.SelectedBy,
			Annotations:     item.Annotations,
			CreatedAt:       item.CreatedAt,
		}}
	case "Ingress":
		item, err := client.GetIngressDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		detail = map[string]any{"ingress": IngressDetail{
			Name:            item.Name,
			IngressClass:    item.IngressClass,
			DefaultBackend:  item.DefaultBackend,
			Addresses:       item.Addresses,
			Rules:           item.Rules,
			TLS:             item.TLS,
			BackendServices: item.BackendServices,
			Annotations:     item.Annotations,
			CreatedAt:       item.CreatedAt,
		}}
	case "ConfigMap":
		item, err := client.GetConfigMapDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		detail = map[string]any{"config_map": ConfigMapDetail{
			Name:         item.Name,
			DataCount:    item.DataCount,
			Entries:      item.Entries,
			ReferencedBy: item.ReferencedBy,
			Annotations:  item.Annotations,
			CreatedAt:    item.CreatedAt,
		}}
	case "Secret":
		item, err := client.GetSecretDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		detail = map[string]any{"secret": SecretDetail{
			Name:         item.Name,
			Type:         item.Type,
			DataCount:    item.DataCount,
			Keys:         item.Keys,
			KeySizes:     item.KeySizes,
			ReferencedBy: item.ReferencedBy,
			Annotations:  item.Annotations,
			CreatedAt:    item.CreatedAt,
		}}
	case "ResourceQuota":
		item, err := client.GetResourceQuotaDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		detail = map[string]any{"resource_quota": ResourceQuotaDetail{
			Name:              item.Name,
			Hard:              item.Hard,
			Used:              item.Used,
			Scopes:            item.Scopes,
			ImpactedWorkloads: item.ImpactedWorkloads,
			Annotations:       item.Annotations,
			CreatedAt:         item.CreatedAt,
		}}
	case "LimitRange":
		item, err := client.GetLimitRangeDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		detail = map[string]any{"limit_range": LimitRangeDetail{
			Name:              item.Name,
			Limits:            item.Limits,
			ImpactedWorkloads: item.ImpactedWorkloads,
			Annotations:       item.Annotations,
			CreatedAt:         item.CreatedAt,
		}}
	default:
		return nil, errors.New("unsupported resource kind")
	}
	return map[string]any{
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    namespace,
		"kind":         kind,
		"name":         name,
		"detail":       detail,
	}, nil
}

func (s *Service) GetNamespaceResourceManifest(clusterID uint, namespace, kind, name, mode string) (map[string]any, error) {
	cluster, client, err := s.resolveClusterClient(clusterID)
	if err != nil {
		return nil, err
	}
	object, err := client.GetNamespaceResourceObject(kind, namespace, name)
	if err != nil {
		return nil, err
	}
	if mode == "compact" {
		object = compactManifest(object)
	}
	data, err := yaml.Marshal(object)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"cluster_id":    cluster.ID,
		"cluster_name":  cluster.Name,
		"namespace":     namespace,
		"kind":          kind,
		"name":          name,
		"mode":          mode,
		"manifest_yaml": string(data),
	}, nil
}

func (s *Service) DeleteNamespaceResource(clusterID uint, namespace, kind, name string) (map[string]any, error) {
	cluster, client, err := s.resolveClusterClient(clusterID)
	if err != nil {
		return nil, err
	}
	if err := client.DeleteNamespaceResource(kind, namespace, name); err != nil {
		return nil, err
	}
	if kind == "Deployment" || kind == "StatefulSet" || kind == "DaemonSet" || kind == "Job" || kind == "CronJob" || kind == "Pod" {
		if _, err := s.SyncWorkloads(cluster.ID); err != nil {
			return nil, err
		}
	}
	return map[string]any{
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    namespace,
		"kind":         kind,
		"name":         name,
		"message":      "resource deleted",
	}, nil
}

func (s *Service) UpdateNamespaceResource(clusterID uint, namespace, kind, name string, input UpdateNamespaceResourceInput) (map[string]any, error) {
	cluster, client, err := s.resolveClusterClient(clusterID)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "ConfigMap":
		if input.ConfigMap == nil {
			return nil, errors.New("config_map payload is required")
		}
		entries, err := parseOptionalStringMap(input.ConfigMap.EntriesText)
		if err != nil {
			return nil, err
		}
		if err := client.UpdateConfigMap(namespace, name, entries); err != nil {
			return nil, err
		}
	case "Secret":
		if input.Secret == nil {
			return nil, errors.New("secret payload is required")
		}
		entries, err := parseOptionalStringMap(input.Secret.EntriesText)
		if err != nil {
			return nil, err
		}
		annotations, err := parseOptionalStringMap(input.Secret.AnnotationsText)
		if err != nil {
			return nil, err
		}
		if err := client.UpdateSecret(namespace, name, secretUpdateSpec{
			Entries:     entries,
			RemoveKeys:  parseMultilineValues(input.Secret.RemoveKeysText),
			Annotations: annotations,
		}); err != nil {
			return nil, err
		}
	case "Service":
		if input.Service == nil {
			return nil, errors.New("service payload is required")
		}
		selector, err := parseOptionalStringMap(input.Service.SelectorText)
		if err != nil {
			return nil, err
		}
		ports, err := parseServicePorts(input.Service.PortsText)
		if err != nil {
			return nil, err
		}
		if len(ports) == 0 {
			return nil, errors.New("at least one service port is required")
		}
		if err := client.UpdateService(namespace, name, serviceUpdateSpec{
			Type:            firstNonEmpty(input.Service.Type, "ClusterIP"),
			SessionAffinity: firstNonEmpty(input.Service.SessionAffinity, "None"),
			Selector:        selector,
			ExternalIPs:     parseMultilineValues(input.Service.ExternalIPsText),
			Ports:           ports,
		}); err != nil {
			return nil, err
		}
	case "Ingress":
		if input.Ingress == nil {
			return nil, errors.New("ingress payload is required")
		}
		annotations, err := parseOptionalStringMap(input.Ingress.AnnotationsText)
		if err != nil {
			return nil, err
		}
		rules, err := parseIngressRules(input.Ingress.RulesText)
		if err != nil {
			return nil, err
		}
		tls, err := parseIngressTLS(input.Ingress.TLSText)
		if err != nil {
			return nil, err
		}
		if err := client.UpdateIngress(namespace, name, ingressUpdateSpec{
			IngressClass: firstNonEmpty(input.Ingress.IngressClass, ""),
			Annotations:  annotations,
			Rules:        rules,
			TLS:          tls,
		}); err != nil {
			return nil, err
		}
	case "ResourceQuota":
		if input.ResourceQuota == nil {
			return nil, errors.New("resource_quota payload is required")
		}
		hard, err := parseOptionalStringMap(input.ResourceQuota.HardText)
		if err != nil {
			return nil, err
		}
		if len(hard) == 0 {
			return nil, errors.New("resource quota hard metrics are required")
		}
		if err := client.UpdateResourceQuota(namespace, name, resourceQuotaUpdateSpec{
			Hard:   hard,
			Scopes: parseMultilineValues(input.ResourceQuota.ScopesText),
		}); err != nil {
			return nil, err
		}
	case "LimitRange":
		if input.LimitRange == nil {
			return nil, errors.New("limit_range payload is required")
		}
		limits, err := parseLimitRangeRules(input.LimitRange.LimitsText)
		if err != nil {
			return nil, err
		}
		if err := client.UpdateLimitRange(namespace, name, limitRangeUpdateSpec{
			Limits: limits,
		}); err != nil {
			return nil, err
		}
	case "ServiceAccount":
		if input.ServiceAccount == nil {
			return nil, errors.New("service_account payload is required")
		}
		annotations, err := parseOptionalStringMap(input.ServiceAccount.AnnotationsText)
		if err != nil {
			return nil, err
		}
		if err := client.UpdateServiceAccount(namespace, name, serviceAccountUpdateSpec{
			ImagePullSecrets: parseMultilineValues(input.ServiceAccount.ImagePullSecretsText),
			Annotations:      annotations,
		}); err != nil {
			return nil, err
		}
	case "PersistentVolumeClaim":
		if input.PVC == nil {
			return nil, errors.New("persistent_volume_claim payload is required")
		}
		annotations, err := parseOptionalStringMap(input.PVC.AnnotationsText)
		if err != nil {
			return nil, err
		}
		if err := client.UpdatePersistentVolumeClaim(namespace, name, persistentVolumeClaimUpdateSpec{
			RequestedStorage: input.PVC.RequestedStorage,
			Annotations:      annotations,
		}); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("current version only supports editing ConfigMap / Secret / Service / Ingress / ResourceQuota / LimitRange / ServiceAccount / PersistentVolumeClaim")
	}
	return map[string]any{
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    namespace,
		"kind":         kind,
		"name":         name,
		"message":      "resource updated",
	}, nil
}

func parseLimitRangeRules(raw string) ([]map[string]any, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, errors.New("limitrange rules are required")
	}
	var limits []map[string]any
	if err := json.Unmarshal([]byte(text), &limits); err == nil {
		if len(limits) == 0 {
			return nil, errors.New("at least one limitrange rule is required")
		}
		return limits, nil
	}
	if err := yaml.Unmarshal([]byte(text), &limits); err != nil {
		return nil, errors.New("limitrange rules must be valid JSON or YAML")
	}
	if len(limits) == 0 {
		return nil, errors.New("at least one limitrange rule is required")
	}
	return limits, nil
}

func extractCreatedResources(objects []map[string]any) []CreatedResource {
	results := make([]CreatedResource, 0, len(objects))
	for _, object := range objects {
		kind, _ := object["kind"].(string)
		metadata, _ := object["metadata"].(map[string]any)
		name, _ := metadata["name"].(string)
		namespace, _ := metadata["namespace"].(string)
		if strings.TrimSpace(kind) == "" || strings.TrimSpace(name) == "" {
			continue
		}
		results = append(results, CreatedResource{
			Kind:      strings.TrimSpace(kind),
			Name:      strings.TrimSpace(name),
			Namespace: strings.TrimSpace(namespace),
		})
	}
	return results
}
