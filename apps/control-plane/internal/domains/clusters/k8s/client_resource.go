package k8s

import (
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"
)

func (c *clusterClient) GetServiceDetail(namespace, name string) (*serviceDetailItem, error) {
	var payload serviceDetailPayload
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/services/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	endpoints, err := c.getServiceEndpoints(namespace, name)
	if err != nil {
		return nil, err
	}
	selectedBy, err := c.findWorkloadsMatchingSelector(namespace, payload.Spec.Selector)
	if err != nil {
		return nil, err
	}
	ports := make([]map[string]any, 0, len(payload.Spec.Ports))
	for _, port := range payload.Spec.Ports {
		ports = append(ports, map[string]any{
			"name":        port.Name,
			"protocol":    firstNonEmpty(port.Protocol, "TCP"),
			"port":        port.Port,
			"target_port": stringifyAny(port.TargetPort),
			"node_port":   port.NodePort,
		})
	}
	return &serviceDetailItem{
		Name:            payload.Metadata.Name,
		Type:            firstNonEmpty(payload.Spec.Type, "ClusterIP"),
		ClusterIP:       firstNonEmpty(payload.Spec.ClusterIP, "-"),
		SessionAffinity: firstNonEmpty(payload.Spec.SessionAffinity, "None"),
		Selector:        selectorPairs(payload.Spec.Selector),
		ExternalIPs:     uniqueStrings(payload.Spec.ExternalIPs),
		Ports:           ports,
		Endpoints:       endpoints,
		SelectedBy:      selectedBy,
		Annotations:     payload.Metadata.Annotations,
		CreatedAt:       payload.Metadata.CreationTimestamp,
	}, nil
}

func (c *clusterClient) GetIngressDetail(namespace, name string) (*ingressDetailItem, error) {
	var payload ingressDetailPayload
	if err := c.getJSON(fmt.Sprintf("/apis/networking.k8s.io/v1/namespaces/%s/ingresses/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	tls := make([]map[string]any, 0, len(payload.Spec.TLS))
	for _, item := range payload.Spec.TLS {
		tls = append(tls, map[string]any{
			"hosts":       uniqueStrings(item.Hosts),
			"secret_name": item.SecretName,
		})
	}
	rules := make([]map[string]any, 0, len(payload.Spec.Rules))
	serviceRefs := map[string][]string{}
	addServiceRef := func(serviceName, route string) {
		serviceName = strings.TrimSpace(serviceName)
		route = strings.TrimSpace(route)
		if serviceName == "" {
			return
		}
		serviceRefs[serviceName] = append(serviceRefs[serviceName], route)
	}
	defaultBackendName := payload.Spec.DefaultBackend.Service.Name
	defaultBackendText := stringifyBackend(defaultBackendName, payload.Spec.DefaultBackend.Service.Port.Number, payload.Spec.DefaultBackend.Service.Port.Name)
	if strings.TrimSpace(defaultBackendName) != "" {
		addServiceRef(defaultBackendName, "default backend")
	}
	for _, rule := range payload.Spec.Rules {
		paths := make([]map[string]any, 0, len(rule.HTTP.Paths))
		for _, path := range rule.HTTP.Paths {
			routeText := firstNonEmpty(rule.Host, "*")
			routeText = fmt.Sprintf("%s%s", routeText, firstNonEmpty(path.Path, "/"))
			addServiceRef(path.Backend.Service.Name, routeText)
			paths = append(paths, map[string]any{
				"path":         path.Path,
				"path_type":    path.PathType,
				"service":      path.Backend.Service.Name,
				"service_port": firstNonEmpty(stringifyPort(path.Backend.Service.Port.Number, path.Backend.Service.Port.Name), "-"),
			})
		}
		rules = append(rules, map[string]any{
			"host":  rule.Host,
			"paths": paths,
		})
	}
	backendServices, err := c.describeIngressBackendServices(namespace, serviceRefs)
	if err != nil {
		return nil, err
	}
	return &ingressDetailItem{
		Name:            payload.Metadata.Name,
		IngressClass:    payload.Spec.IngressClassName,
		Addresses:       loadBalancerAddresses(payload.Status.LoadBalancer.Ingress),
		DefaultBackend:  defaultBackendText,
		TLS:             tls,
		Rules:           rules,
		BackendServices: backendServices,
		Annotations:     payload.Metadata.Annotations,
		CreatedAt:       payload.Metadata.CreationTimestamp,
	}, nil
}

func (c *clusterClient) ListNamespaceResources(namespace, kind string) ([]namespaceResourceItem, error) {
	switch kind {
	case "ServiceAccount":
		return c.listNamespaceServiceAccounts(namespace)
	case "PersistentVolumeClaim":
		return c.listNamespacePersistentVolumeClaims(namespace)
	case "Service":
		return c.listNamespaceServices(namespace)
	case "Ingress":
		return c.listNamespaceIngresses(namespace)
	case "ConfigMap":
		return c.listNamespaceConfigMaps(namespace)
	case "Secret":
		return c.listNamespaceSecrets(namespace)
	case "ResourceQuota":
		return c.listNamespaceResourceQuotas(namespace)
	case "LimitRange":
		return c.listNamespaceLimitRanges(namespace)
	default:
		return nil, fmt.Errorf("unsupported resource kind: %s", kind)
	}
}

func (c *clusterClient) GetNamespaceResourceDetail(kind, namespace, name string) (map[string]any, error) {
	switch kind {
	case "ServiceAccount":
		item, err := c.GetServiceAccountDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		return map[string]any{"service_account": item}, nil
	case "PersistentVolumeClaim":
		item, err := c.GetPersistentVolumeClaimDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		return map[string]any{"persistent_volume_claim": item}, nil
	case "Service":
		item, err := c.GetServiceDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		return map[string]any{"service": item}, nil
	case "Ingress":
		item, err := c.GetIngressDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		return map[string]any{"ingress": item}, nil
	case "ConfigMap":
		item, err := c.GetConfigMapDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		return map[string]any{"config_map": item}, nil
	case "Secret":
		item, err := c.GetSecretDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		return map[string]any{"secret": item}, nil
	case "ResourceQuota":
		item, err := c.GetResourceQuotaDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		return map[string]any{"resource_quota": item}, nil
	case "LimitRange":
		item, err := c.GetLimitRangeDetail(namespace, name)
		if err != nil {
			return nil, err
		}
		return map[string]any{"limit_range": item}, nil
	default:
		return nil, fmt.Errorf("unsupported resource kind: %s", kind)
	}
}

func (c *clusterClient) GetNamespaceResourceObject(kind, namespace, name string) (map[string]any, error) {
	path, err := namespaceResourcePath(kind, namespace, name)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := c.getJSON(path, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *clusterClient) GetConfigMapDetail(namespace, name string) (*configMapDetailItem, error) {
	var payload configMapDetailPayload
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/configmaps/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	keys := sortedKeys(payload.Data)
	entries := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		value := payload.Data[key]
		entries = append(entries, map[string]any{
			"key":          key,
			"value":        value,
			"line_count":   strings.Count(value, "\n") + 1,
			"char_count":   len(value),
			"is_multiline": strings.Contains(value, "\n"),
		})
	}
	referencedBy, err := c.findReferencingWorkloads(namespace, "ConfigMap", name)
	if err != nil {
		return nil, err
	}
	return &configMapDetailItem{
		Name:         payload.Metadata.Name,
		Immutable:    payload.Immutable,
		Keys:         keys,
		DataCount:    len(payload.Data),
		CreatedAt:    payload.Metadata.CreationTimestamp,
		Entries:      entries,
		ReferencedBy: referencedBy,
		Annotations:  payload.Metadata.Annotations,
	}, nil
}

func (c *clusterClient) GetSecretDetail(namespace, name string) (*secretDetailItem, error) {
	var payload secretDetailPayload
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/secrets/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	keys := sortedKeys(payload.Data)
	keySizes := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		keySizes = append(keySizes, map[string]any{
			"key":          key,
			"encoded_size": len(payload.Data[key]),
		})
	}
	referencedBy, err := c.findReferencingWorkloads(namespace, "Secret", name)
	if err != nil {
		return nil, err
	}
	return &secretDetailItem{
		Name:         payload.Metadata.Name,
		Type:         payload.Type,
		Keys:         keys,
		DataCount:    len(payload.Data),
		CreatedAt:    payload.Metadata.CreationTimestamp,
		KeySizes:     keySizes,
		ReferencedBy: referencedBy,
		Annotations:  payload.Metadata.Annotations,
	}, nil
}

func (c *clusterClient) GetServiceAccountDetail(namespace, name string) (*serviceAccountDetailItem, error) {
	var payload struct {
		Metadata struct {
			Name              string            `json:"name"`
			CreationTimestamp string            `json:"creationTimestamp"`
			Annotations       map[string]string `json:"annotations"`
		} `json:"metadata"`
		Secrets []struct {
			Name string `json:"name"`
		} `json:"secrets"`
		ImagePullSecrets []struct {
			Name string `json:"name"`
		} `json:"imagePullSecrets"`
	}
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/serviceaccounts/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	referencedBy, err := c.findWorkloadsByServiceAccount(namespace, name)
	if err != nil {
		return nil, err
	}
	secrets := make([]string, 0, len(payload.Secrets))
	for _, item := range payload.Secrets {
		if strings.TrimSpace(item.Name) != "" {
			secrets = append(secrets, item.Name)
		}
	}
	imagePullSecrets := make([]string, 0, len(payload.ImagePullSecrets))
	for _, item := range payload.ImagePullSecrets {
		if strings.TrimSpace(item.Name) != "" {
			imagePullSecrets = append(imagePullSecrets, item.Name)
		}
	}
	return &serviceAccountDetailItem{
		Name:             payload.Metadata.Name,
		Secrets:          uniqueStrings(secrets),
		ImagePullSecrets: uniqueStrings(imagePullSecrets),
		CreatedAt:        payload.Metadata.CreationTimestamp,
		ReferencedBy:     referencedBy,
		Annotations:      payload.Metadata.Annotations,
	}, nil
}

func (c *clusterClient) GetPersistentVolumeClaimDetail(namespace, name string) (*persistentVolumeClaimDetailItem, error) {
	var payload struct {
		Metadata struct {
			Name              string            `json:"name"`
			CreationTimestamp string            `json:"creationTimestamp"`
			Annotations       map[string]string `json:"annotations"`
		} `json:"metadata"`
		Spec struct {
			AccessModes      []string `json:"accessModes"`
			StorageClassName string   `json:"storageClassName"`
			VolumeName       string   `json:"volumeName"`
			Resources        struct {
				Requests map[string]string `json:"requests"`
			} `json:"resources"`
		} `json:"spec"`
		Status struct {
			Phase    string            `json:"phase"`
			Capacity map[string]string `json:"capacity"`
		} `json:"status"`
	}
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/persistentvolumeclaims/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	mountedBy, err := c.findWorkloadsByPersistentVolumeClaim(namespace, name)
	if err != nil {
		return nil, err
	}
	return &persistentVolumeClaimDetailItem{
		Name:             payload.Metadata.Name,
		Status:           firstNonEmpty(payload.Status.Phase, "Pending"),
		VolumeName:       payload.Spec.VolumeName,
		StorageClassName: payload.Spec.StorageClassName,
		AccessModes:      uniqueStrings(payload.Spec.AccessModes),
		Capacity:         payload.Status.Capacity["storage"],
		RequestedStorage: payload.Spec.Resources.Requests["storage"],
		Annotations:      payload.Metadata.Annotations,
		CreatedAt:        payload.Metadata.CreationTimestamp,
		MountedBy:        mountedBy,
	}, nil
}

func (c *clusterClient) GetResourceQuotaDetail(namespace, name string) (*resourceQuotaDetailItem, error) {
	var payload struct {
		Metadata struct {
			Name              string            `json:"name"`
			CreationTimestamp string            `json:"creationTimestamp"`
			Annotations       map[string]string `json:"annotations"`
		} `json:"metadata"`
		Spec struct {
			Hard   map[string]string `json:"hard"`
			Scopes []string          `json:"scopes"`
		} `json:"spec"`
		Status struct {
			Hard map[string]string `json:"hard"`
			Used map[string]string `json:"used"`
		} `json:"status"`
	}
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/resourcequotas/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	hard := payload.Status.Hard
	if len(hard) == 0 {
		hard = payload.Spec.Hard
	}
	impactedWorkloads, err := c.listNamespaceControllerWorkloads(namespace)
	if err != nil {
		return nil, err
	}
	return &resourceQuotaDetailItem{
		Name:              payload.Metadata.Name,
		CreatedAt:         payload.Metadata.CreationTimestamp,
		Hard:              hard,
		Used:              payload.Status.Used,
		Scopes:            payload.Spec.Scopes,
		ImpactedWorkloads: impactedWorkloads,
		Annotations:       payload.Metadata.Annotations,
	}, nil
}

func (c *clusterClient) GetLimitRangeDetail(namespace, name string) (*limitRangeDetailItem, error) {
	var payload struct {
		Metadata struct {
			Name              string            `json:"name"`
			CreationTimestamp string            `json:"creationTimestamp"`
			Annotations       map[string]string `json:"annotations"`
		} `json:"metadata"`
		Spec struct {
			Limits []struct {
				Type                 string            `json:"type"`
				Default              map[string]string `json:"default"`
				DefaultRequest       map[string]string `json:"defaultRequest"`
				Max                  map[string]string `json:"max"`
				Min                  map[string]string `json:"min"`
				MaxLimitRequestRatio map[string]string `json:"maxLimitRequestRatio"`
			} `json:"limits"`
		} `json:"spec"`
	}
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/limitranges/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	limits := make([]map[string]any, 0, len(payload.Spec.Limits))
	for _, item := range payload.Spec.Limits {
		limits = append(limits, map[string]any{
			"type":                    item.Type,
			"default":                 item.Default,
			"default_request":         item.DefaultRequest,
			"max":                     item.Max,
			"min":                     item.Min,
			"max_limit_request_ratio": item.MaxLimitRequestRatio,
		})
	}
	impactedWorkloads, err := c.listNamespaceControllerWorkloads(namespace)
	if err != nil {
		return nil, err
	}
	return &limitRangeDetailItem{
		Name:              payload.Metadata.Name,
		CreatedAt:         payload.Metadata.CreationTimestamp,
		Limits:            limits,
		ImpactedWorkloads: impactedWorkloads,
		Annotations:       payload.Metadata.Annotations,
	}, nil
}

func (c *clusterClient) findReferencingWorkloads(namespace, targetKind, targetName string) ([]map[string]any, error) {
	items, err := c.ListWorkloads()
	if err != nil {
		return nil, err
	}
	results := make([]map[string]any, 0)
	for _, item := range items {
		if item.NamespaceName != namespace {
			continue
		}
		if item.Kind != "Deployment" && item.Kind != "StatefulSet" {
			continue
		}
		object, err := c.GetWorkloadObject(item.Kind, item.NamespaceName, item.Name)
		if err != nil {
			return nil, err
		}
		spec, _ := object["spec"].(map[string]any)
		if spec == nil {
			continue
		}
		podSpec := extractMap(spec, "template", "spec")
		refs := collectWorkloadRefs(podSpec)
		var matched bool
		switch targetKind {
		case "ConfigMap":
			matched = containsString(refs.ConfigMaps, targetName)
		case "Secret":
			matched = containsString(refs.Secrets, targetName)
		}
		if !matched {
			continue
		}
		results = append(results, map[string]any{
			"kind":      item.Kind,
			"name":      item.Name,
			"namespace": item.NamespaceName,
			"image":     item.Image,
			"status":    item.Status,
		})
	}
	return results, nil
}

func (c *clusterClient) findWorkloadsByServiceAccount(namespace, serviceAccountName string) ([]map[string]any, error) {
	items, err := c.ListWorkloads()
	if err != nil {
		return nil, err
	}
	results := make([]map[string]any, 0)
	for _, item := range items {
		if item.NamespaceName != namespace {
			continue
		}
		if item.Kind != "Deployment" && item.Kind != "StatefulSet" {
			continue
		}
		object, err := c.GetWorkloadObject(item.Kind, item.NamespaceName, item.Name)
		if err != nil {
			return nil, err
		}
		podSpec := extractMap(extractMap(object, "spec"), "template", "spec")
		if firstNonEmpty(stringValue(podSpec["serviceAccountName"]), "default") != serviceAccountName {
			continue
		}
		results = append(results, map[string]any{
			"kind":      item.Kind,
			"name":      item.Name,
			"namespace": item.NamespaceName,
			"image":     item.Image,
			"status":    item.Status,
		})
	}
	return results, nil
}

func (c *clusterClient) findWorkloadsByPersistentVolumeClaim(namespace, claimName string) ([]map[string]any, error) {
	items, err := c.ListWorkloads()
	if err != nil {
		return nil, err
	}
	results := make([]map[string]any, 0)
	for _, item := range items {
		if item.NamespaceName != namespace {
			continue
		}
		if item.Kind != "Deployment" && item.Kind != "StatefulSet" {
			continue
		}
		object, err := c.GetWorkloadObject(item.Kind, item.NamespaceName, item.Name)
		if err != nil {
			return nil, err
		}
		spec := extractMap(object, "spec")
		podSpec := extractMap(spec, "template", "spec")
		volumes, _ := podSpec["volumes"].([]any)
		matched := false
		for _, raw := range volumes {
			volume, _ := raw.(map[string]any)
			pvc, _ := volume["persistentVolumeClaim"].(map[string]any)
			if firstNonEmpty(stringValue(pvc["claimName"])) == claimName {
				matched = true
				break
			}
		}
		if !matched && item.Kind == "StatefulSet" {
			for _, tmpl := range extractClaimTemplates(spec) {
				if strings.HasPrefix(claimName, fmt.Sprintf("%s-%s-", tmpl.Name, item.Name)) {
					matched = true
					break
				}
			}
		}
		if !matched {
			continue
		}
		results = append(results, map[string]any{
			"kind":      item.Kind,
			"name":      item.Name,
			"namespace": item.NamespaceName,
			"image":     item.Image,
			"status":    item.Status,
		})
	}
	return results, nil
}

func (c *clusterClient) findWorkloadsMatchingSelector(namespace string, selector map[string]string) ([]map[string]any, error) {
	if len(selector) == 0 {
		return []map[string]any{}, nil
	}
	items, err := c.ListWorkloads()
	if err != nil {
		return nil, err
	}
	results := make([]map[string]any, 0)
	for _, item := range items {
		if item.NamespaceName != namespace {
			continue
		}
		if item.Kind != "Deployment" && item.Kind != "StatefulSet" && item.Kind != "DaemonSet" {
			continue
		}
		object, err := c.GetWorkloadObject(item.Kind, item.NamespaceName, item.Name)
		if err != nil {
			return nil, err
		}
		templateLabels := extractMap(extractMap(extractMap(object, "spec"), "template"), "metadata")
		labels, _ := templateLabels["labels"].(map[string]any)
		if !selectorMatchesAnyMap(selector, labels) {
			continue
		}
		results = append(results, map[string]any{
			"kind":      item.Kind,
			"name":      item.Name,
			"namespace": item.NamespaceName,
			"image":     item.Image,
			"status":    item.Status,
		})
	}
	return results, nil
}

func (c *clusterClient) describeIngressBackendServices(namespace string, refs map[string][]string) ([]map[string]any, error) {
	if len(refs) == 0 {
		return []map[string]any{}, nil
	}
	names := make([]string, 0, len(refs))
	for name := range refs {
		names = append(names, name)
	}
	slices.Sort(names)
	results := make([]map[string]any, 0, len(names))
	for _, name := range names {
		detail, err := c.GetServiceDetail(namespace, name)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				results = append(results, map[string]any{
					"name":       name,
					"routes":     uniqueStrings(refs[name]),
					"status":     "missing",
					"type":       "-",
					"cluster_ip": "-",
				})
				continue
			}
			return nil, err
		}
		results = append(results, map[string]any{
			"name":        detail.Name,
			"routes":      uniqueStrings(refs[name]),
			"type":        detail.Type,
			"cluster_ip":  detail.ClusterIP,
			"ports":       detail.Ports,
			"selected_by": detail.SelectedBy,
			"status":      ternaryString(len(detail.SelectedBy) > 0, "routed", "no-target"),
		})
	}
	return results, nil
}

func selectorMatchesAnyMap(selector map[string]string, labels map[string]any) bool {
	for key, value := range selector {
		current, ok := labels[key]
		if !ok || strings.TrimSpace(stringValue(current)) != strings.TrimSpace(value) {
			return false
		}
	}
	return true
}

func (c *clusterClient) listNamespaceControllerWorkloads(namespace string) ([]map[string]any, error) {
	items, err := c.ListWorkloads()
	if err != nil {
		return nil, err
	}
	results := make([]map[string]any, 0)
	for _, item := range items {
		if item.NamespaceName != namespace {
			continue
		}
		if item.Kind != "Deployment" && item.Kind != "StatefulSet" {
			continue
		}
		results = append(results, map[string]any{
			"kind":           item.Kind,
			"name":           item.Name,
			"namespace":      item.NamespaceName,
			"image":          item.Image,
			"status":         item.Status,
			"replicas":       item.Replicas,
			"ready_replicas": item.ReadyReplicas,
		})
	}
	return results, nil
}

func (c *clusterClient) DeleteNamespaceResource(kind, namespace, name string) error {
	path, err := namespaceResourcePath(kind, namespace, name)
	if err != nil {
		return err
	}
	return c.delete(path)
}

func (c *clusterClient) UpdateConfigMap(namespace, name string, entries map[string]string) error {
	object, err := c.GetNamespaceResourceObject("ConfigMap", namespace, name)
	if err != nil {
		return err
	}
	metadata := sanitizeKubernetesMetadata(extractMap(object, "metadata"))
	if metadata == nil {
		return errors.New("configmap metadata is missing")
	}
	object["metadata"] = metadata
	object["data"] = stringMapToAny(entries)
	return c.putJSON(fmt.Sprintf("/api/v1/namespaces/%s/configmaps/%s", namespace, name), object, nil)
}

func (c *clusterClient) UpdateSecret(namespace, name string, settings secretUpdateSpec) error {
	object, err := c.GetNamespaceResourceObject("Secret", namespace, name)
	if err != nil {
		return err
	}
	metadata := sanitizeKubernetesMetadata(extractMap(object, "metadata"))
	if metadata == nil {
		return errors.New("secret metadata is missing")
	}
	metadata["annotations"] = stringMapToAny(settings.Annotations)
	data, _ := object["data"].(map[string]any)
	if data == nil {
		data = map[string]any{}
	}
	for key, value := range settings.Entries {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		data[trimmedKey] = base64.StdEncoding.EncodeToString([]byte(value))
	}
	for _, key := range settings.RemoveKeys {
		delete(data, strings.TrimSpace(key))
	}
	object["metadata"] = metadata
	object["data"] = data
	delete(object, "stringData")
	return c.putJSON(fmt.Sprintf("/api/v1/namespaces/%s/secrets/%s", namespace, name), object, nil)
}

func (c *clusterClient) UpdateService(namespace, name string, settings serviceUpdateSpec) error {
	object, err := c.GetNamespaceResourceObject("Service", namespace, name)
	if err != nil {
		return err
	}
	metadata := sanitizeKubernetesMetadata(extractMap(object, "metadata"))
	spec := sanitizeServiceSpec(extractMap(object, "spec"))
	if metadata == nil || spec == nil {
		return errors.New("service object is incomplete")
	}
	spec["type"] = firstNonEmpty(settings.Type, "ClusterIP")
	spec["sessionAffinity"] = firstNonEmpty(settings.SessionAffinity, "None")
	spec["selector"] = stringMapToAny(settings.Selector)
	spec["ports"] = settings.Ports
	if len(settings.ExternalIPs) > 0 {
		spec["externalIPs"] = settings.ExternalIPs
	} else {
		delete(spec, "externalIPs")
	}
	object["metadata"] = metadata
	object["spec"] = spec
	return c.putJSON(fmt.Sprintf("/api/v1/namespaces/%s/services/%s", namespace, name), object, nil)
}

func (c *clusterClient) UpdateIngress(namespace, name string, settings ingressUpdateSpec) error {
	object, err := c.GetNamespaceResourceObject("Ingress", namespace, name)
	if err != nil {
		return err
	}
	metadata := sanitizeKubernetesMetadata(extractMap(object, "metadata"))
	spec := sanitizeIngressSpec(extractMap(object, "spec"))
	if metadata == nil || spec == nil {
		return errors.New("ingress object is incomplete")
	}
	metadata["annotations"] = stringMapToAny(settings.Annotations)
	if settings.IngressClass != "" {
		spec["ingressClassName"] = settings.IngressClass
	} else {
		delete(spec, "ingressClassName")
	}
	spec["rules"] = settings.Rules
	if len(settings.TLS) > 0 {
		spec["tls"] = settings.TLS
	} else {
		delete(spec, "tls")
	}
	object["metadata"] = metadata
	object["spec"] = spec
	return c.putJSON(fmt.Sprintf("/apis/networking.k8s.io/v1/namespaces/%s/ingresses/%s", namespace, name), object, nil)
}

func (c *clusterClient) UpdateResourceQuota(namespace, name string, settings resourceQuotaUpdateSpec) error {
	object, err := c.GetNamespaceResourceObject("ResourceQuota", namespace, name)
	if err != nil {
		return err
	}
	metadata := sanitizeKubernetesMetadata(extractMap(object, "metadata"))
	spec := extractMap(object, "spec")
	if metadata == nil || spec == nil {
		return errors.New("resourcequota object is incomplete")
	}
	spec["hard"] = stringMapToAny(settings.Hard)
	if len(settings.Scopes) > 0 {
		spec["scopes"] = settings.Scopes
	} else {
		delete(spec, "scopes")
	}
	object["metadata"] = metadata
	object["spec"] = spec
	return c.putJSON(fmt.Sprintf("/api/v1/namespaces/%s/resourcequotas/%s", namespace, name), object, nil)
}

func (c *clusterClient) UpdateLimitRange(namespace, name string, settings limitRangeUpdateSpec) error {
	object, err := c.GetNamespaceResourceObject("LimitRange", namespace, name)
	if err != nil {
		return err
	}
	metadata := sanitizeKubernetesMetadata(extractMap(object, "metadata"))
	spec := extractMap(object, "spec")
	if metadata == nil || spec == nil {
		return errors.New("limitrange object is incomplete")
	}
	if len(settings.Limits) == 0 {
		return errors.New("at least one limitrange rule is required")
	}
	spec["limits"] = settings.Limits
	object["metadata"] = metadata
	object["spec"] = spec
	return c.putJSON(fmt.Sprintf("/api/v1/namespaces/%s/limitranges/%s", namespace, name), object, nil)
}

func (c *clusterClient) UpdateServiceAccount(namespace, name string, settings serviceAccountUpdateSpec) error {
	object, err := c.GetNamespaceResourceObject("ServiceAccount", namespace, name)
	if err != nil {
		return err
	}
	metadata := sanitizeKubernetesMetadata(extractMap(object, "metadata"))
	if metadata == nil {
		return errors.New("serviceaccount object is incomplete")
	}
	metadata["annotations"] = stringMapToAny(settings.Annotations)
	imagePullSecrets := make([]map[string]any, 0, len(settings.ImagePullSecrets))
	for _, item := range settings.ImagePullSecrets {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		imagePullSecrets = append(imagePullSecrets, map[string]any{"name": value})
	}
	object["metadata"] = metadata
	if len(imagePullSecrets) > 0 {
		object["imagePullSecrets"] = imagePullSecrets
	} else {
		delete(object, "imagePullSecrets")
	}
	return c.putJSON(fmt.Sprintf("/api/v1/namespaces/%s/serviceaccounts/%s", namespace, name), object, nil)
}

func (c *clusterClient) UpdatePersistentVolumeClaim(namespace, name string, settings persistentVolumeClaimUpdateSpec) error {
	object, err := c.GetNamespaceResourceObject("PersistentVolumeClaim", namespace, name)
	if err != nil {
		return err
	}
	metadata := sanitizeKubernetesMetadata(extractMap(object, "metadata"))
	spec := extractMap(object, "spec")
	if metadata == nil || spec == nil {
		return errors.New("persistentvolumeclaim object is incomplete")
	}
	requested := strings.TrimSpace(settings.RequestedStorage)
	if requested == "" {
		return errors.New("pvc requested storage is required")
	}
	metadata["annotations"] = stringMapToAny(settings.Annotations)
	resources := extractMap(spec, "resources")
	if resources == nil {
		resources = map[string]any{}
	}
	requests := extractMap(resources, "requests")
	if requests == nil {
		requests = map[string]any{}
	}
	requests["storage"] = requested
	resources["requests"] = requests
	spec["resources"] = resources
	object["metadata"] = metadata
	object["spec"] = spec
	return c.putJSON(fmt.Sprintf("/api/v1/namespaces/%s/persistentvolumeclaims/%s", namespace, name), object, nil)
}

func (c *clusterClient) getServiceObject(namespace, name string) (map[string]any, error) {
	var payload map[string]any
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/services/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *clusterClient) replaceStatefulSetServiceMode(namespace, name, mode string) error {
	serviceObject, err := c.getServiceObject(namespace, name)
	if err != nil {
		return err
	}
	metadata := extractMap(serviceObject, "metadata")
	spec := extractMap(serviceObject, "spec")
	if metadata == nil || spec == nil {
		return errors.New("service object is incomplete")
	}
	delete(metadata, "uid")
	delete(metadata, "resourceVersion")
	delete(metadata, "managedFields")
	delete(metadata, "creationTimestamp")
	delete(metadata, "selfLink")

	delete(spec, "clusterIPs")
	delete(spec, "healthCheckNodePort")
	delete(spec, "ipFamilies")
	delete(spec, "ipFamilyPolicy")
	delete(spec, "internalTrafficPolicy")
	delete(spec, "trafficDistribution")
	delete(spec, "allocateLoadBalancerNodePorts")

	switch firstNonEmpty(mode, "ClusterIP") {
	case "Headless":
		spec["type"] = "ClusterIP"
		spec["clusterIP"] = "None"
		spec["publishNotReadyAddresses"] = true
	case "LoadBalancer":
		delete(spec, "clusterIP")
		spec["type"] = "LoadBalancer"
		spec["publishNotReadyAddresses"] = false
	default:
		delete(spec, "clusterIP")
		spec["type"] = "ClusterIP"
		spec["publishNotReadyAddresses"] = false
	}

	if err := c.delete(fmt.Sprintf("/api/v1/namespaces/%s/services/%s", namespace, name)); err != nil {
		return err
	}
	return c.CreateObject(map[string]any{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata":   metadata,
		"spec":       spec,
	})
}

func sanitizeKubernetesMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}
	delete(metadata, "uid")
	delete(metadata, "resourceVersion")
	delete(metadata, "managedFields")
	delete(metadata, "creationTimestamp")
	delete(metadata, "generation")
	delete(metadata, "selfLink")
	delete(metadata, "ownerReferences")
	return metadata
}

func sanitizeServiceSpec(spec map[string]any) map[string]any {
	if spec == nil {
		return nil
	}
	delete(spec, "clusterIP")
	delete(spec, "clusterIPs")
	delete(spec, "healthCheckNodePort")
	delete(spec, "ipFamilies")
	delete(spec, "ipFamilyPolicy")
	delete(spec, "internalTrafficPolicy")
	delete(spec, "trafficDistribution")
	delete(spec, "allocateLoadBalancerNodePorts")
	return spec
}

func sanitizeIngressSpec(spec map[string]any) map[string]any {
	if spec == nil {
		return nil
	}
	delete(spec, "defaultBackend")
	return spec
}

func (c *clusterClient) listNamespaceServices(namespace string) ([]namespaceResourceItem, error) {
	var payload servicesPayload
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/services", namespace), &payload); err != nil {
		return nil, err
	}
	items := make([]namespaceResourceItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		portText := make([]string, 0, len(item.Spec.Ports))
		for _, port := range item.Spec.Ports {
			target := stringifyAny(port.TargetPort)
			if target == "-" || target == "" {
				target = fmt.Sprintf("%d", port.Port)
			}
			portText = append(portText, fmt.Sprintf("%d -> %s", port.Port, target))
		}
		items = append(items, namespaceResourceItem{
			Kind:      "Service",
			Name:      item.Metadata.Name,
			Namespace: namespace,
			Summary:   firstNonEmpty(strings.Join(portText, "，"), "无端口配置"),
			Status:    firstNonEmpty(item.Spec.Type, "ClusterIP"),
			CreatedAt: item.Metadata.CreationTimestamp,
		})
	}
	return items, nil
}

func (c *clusterClient) listNamespaceServiceAccounts(namespace string) ([]namespaceResourceItem, error) {
	var payload struct {
		Items []struct {
			Metadata struct {
				Name              string `json:"name"`
				CreationTimestamp string `json:"creationTimestamp"`
			} `json:"metadata"`
			Secrets []struct {
				Name string `json:"name"`
			} `json:"secrets"`
			ImagePullSecrets []struct {
				Name string `json:"name"`
			} `json:"imagePullSecrets"`
		} `json:"items"`
	}
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/serviceaccounts", namespace), &payload); err != nil {
		return nil, err
	}
	items := make([]namespaceResourceItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, namespaceResourceItem{
			Kind:      "ServiceAccount",
			Name:      item.Metadata.Name,
			Namespace: namespace,
			Summary:   fmt.Sprintf("%d secrets / %d imagePullSecrets", len(item.Secrets), len(item.ImagePullSecrets)),
			Status:    ternaryString(item.Metadata.Name == "default", "default", "custom"),
			CreatedAt: item.Metadata.CreationTimestamp,
		})
	}
	return items, nil
}

func (c *clusterClient) listNamespacePersistentVolumeClaims(namespace string) ([]namespaceResourceItem, error) {
	var payload struct {
		Items []struct {
			Metadata struct {
				Name              string `json:"name"`
				CreationTimestamp string `json:"creationTimestamp"`
			} `json:"metadata"`
			Spec struct {
				AccessModes []string `json:"accessModes"`
				Resources   struct {
					Requests map[string]string `json:"requests"`
				} `json:"resources"`
			} `json:"spec"`
			Status struct {
				Phase string `json:"phase"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/persistentvolumeclaims", namespace), &payload); err != nil {
		return nil, err
	}
	items := make([]namespaceResourceItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, namespaceResourceItem{
			Kind:      "PersistentVolumeClaim",
			Name:      item.Metadata.Name,
			Namespace: namespace,
			Summary:   firstNonEmpty(item.Spec.Resources.Requests["storage"], "未声明容量"),
			Status:    firstNonEmpty(item.Status.Phase, "Pending"),
			CreatedAt: item.Metadata.CreationTimestamp,
		})
	}
	return items, nil
}

func (c *clusterClient) listNamespaceIngresses(namespace string) ([]namespaceResourceItem, error) {
	var payload ingressesPayload
	if err := c.getJSON(fmt.Sprintf("/apis/networking.k8s.io/v1/namespaces/%s/ingresses", namespace), &payload); err != nil {
		if strings.Contains(err.Error(), "404") {
			return []namespaceResourceItem{}, nil
		}
		return nil, err
	}
	items := make([]namespaceResourceItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		hosts := make([]string, 0)
		for _, rule := range item.Spec.Rules {
			if strings.TrimSpace(rule.Host) != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		items = append(items, namespaceResourceItem{
			Kind:      "Ingress",
			Name:      item.Metadata.Name,
			Namespace: namespace,
			Summary:   firstNonEmpty(strings.Join(uniqueStrings(hosts), "，"), "无 Host"),
			Status:    fmt.Sprintf("%d 条规则", len(item.Spec.Rules)),
			CreatedAt: "",
		})
	}
	return items, nil
}

func (c *clusterClient) listNamespaceConfigMaps(namespace string) ([]namespaceResourceItem, error) {
	var payload configMapsPayload
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/configmaps", namespace), &payload); err != nil {
		return nil, err
	}
	items := make([]namespaceResourceItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, namespaceResourceItem{
			Kind:      "ConfigMap",
			Name:      item.Metadata.Name,
			Namespace: namespace,
			Summary:   fmt.Sprintf("%d 个键", len(item.Data)),
			Status:    ternaryString(item.Immutable, "Immutable", "Mutable"),
			CreatedAt: item.Metadata.CreationTimestamp,
		})
	}
	return items, nil
}

func (c *clusterClient) listNamespaceSecrets(namespace string) ([]namespaceResourceItem, error) {
	var payload secretsPayload
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/secrets", namespace), &payload); err != nil {
		return nil, err
	}
	items := make([]namespaceResourceItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, namespaceResourceItem{
			Kind:      "Secret",
			Name:      item.Metadata.Name,
			Namespace: namespace,
			Summary:   fmt.Sprintf("%d 个键", len(item.Data)),
			Status:    firstNonEmpty(item.Type, "Opaque"),
			CreatedAt: item.Metadata.CreationTimestamp,
		})
	}
	return items, nil
}

func (c *clusterClient) listNamespaceResourceQuotas(namespace string) ([]namespaceResourceItem, error) {
	var payload struct {
		Items []struct {
			Metadata struct {
				Name              string `json:"name"`
				CreationTimestamp string `json:"creationTimestamp"`
			} `json:"metadata"`
			Status struct {
				Hard map[string]string `json:"hard"`
				Used map[string]string `json:"used"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/resourcequotas", namespace), &payload); err != nil {
		return nil, err
	}
	items := make([]namespaceResourceItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, namespaceResourceItem{
			Kind:      "ResourceQuota",
			Name:      item.Metadata.Name,
			Namespace: namespace,
			Summary:   fmt.Sprintf("%d hard / %d used 指标", len(item.Status.Hard), len(item.Status.Used)),
			Status:    ternaryString(len(item.Status.Hard) > 0, "configured", "empty"),
			CreatedAt: item.Metadata.CreationTimestamp,
		})
	}
	return items, nil
}

func (c *clusterClient) listNamespaceLimitRanges(namespace string) ([]namespaceResourceItem, error) {
	var payload struct {
		Items []struct {
			Metadata struct {
				Name              string `json:"name"`
				CreationTimestamp string `json:"creationTimestamp"`
			} `json:"metadata"`
			Spec struct {
				Limits []struct{} `json:"limits"`
			} `json:"spec"`
		} `json:"items"`
	}
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/limitranges", namespace), &payload); err != nil {
		return nil, err
	}
	items := make([]namespaceResourceItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, namespaceResourceItem{
			Kind:      "LimitRange",
			Name:      item.Metadata.Name,
			Namespace: namespace,
			Summary:   fmt.Sprintf("%d 条 limit 策略", len(item.Spec.Limits)),
			Status:    ternaryString(len(item.Spec.Limits) > 0, "configured", "empty"),
			CreatedAt: item.Metadata.CreationTimestamp,
		})
	}
	return items, nil
}

func (c *clusterClient) getServiceEndpoints(namespace, name string) ([]string, error) {
	var payload endpointsPayload
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/endpoints/%s", namespace, name), &payload); err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	items := make([]string, 0)
	for _, subset := range payload.Subsets {
		ports := make([]string, 0, len(subset.Ports))
		for _, port := range subset.Ports {
			ports = append(ports, fmt.Sprintf("%d", port.Port))
		}
		portText := strings.Join(ports, ",")
		for _, address := range subset.Addresses {
			if strings.TrimSpace(address.IP) == "" {
				continue
			}
			if portText != "" {
				items = append(items, fmt.Sprintf("%s:%s", address.IP, portText))
			} else {
				items = append(items, address.IP)
			}
		}
	}
	return uniqueStrings(items), nil
}
