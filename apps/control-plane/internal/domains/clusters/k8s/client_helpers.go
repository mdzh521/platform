package k8s

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
)

func firstDeploymentImage(containers []struct {
	Image string `json:"image"`
}) string {
	if len(containers) == 0 {
		return "-"
	}
	return containers[0].Image
}

func firstStatefulSetImage(containers []struct {
	Image string `json:"image"`
}) string {
	if len(containers) == 0 {
		return "-"
	}
	return containers[0].Image
}

func firstDaemonSetImage(containers []struct {
	Image string `json:"image"`
}) string {
	if len(containers) == 0 {
		return "-"
	}
	return containers[0].Image
}

func firstJobImage(containers []struct {
	Image string `json:"image"`
}) string {
	if len(containers) == 0 {
		return "-"
	}
	return containers[0].Image
}

func firstCronJobImage(containers []struct {
	Image string `json:"image"`
}) string {
	if len(containers) == 0 {
		return "-"
	}
	return containers[0].Image
}

func firstPodImage(containers []struct {
	Image string `json:"image"`
}) string {
	if len(containers) == 0 {
		return "-"
	}
	return containers[0].Image
}

func workloadStatus(readyReplicas, replicas int) string {
	if replicas == 0 {
		return "scaled-to-zero"
	}
	if readyReplicas >= replicas {
		return "ready"
	}
	if readyReplicas > 0 {
		return "degraded"
	}
	return "pending"
}

func jobStatus(active, succeeded, failed, desired int) string {
	if failed > 0 {
		return "failed"
	}
	if succeeded >= maxInt(desired, 1) {
		return "completed"
	}
	if active > 0 {
		return "running"
	}
	return "pending"
}

func cronJobStatus(suspended bool, active int) string {
	if suspended {
		return "suspended"
	}
	if active > 0 {
		return "running"
	}
	return "scheduled"
}

func workloadResourcePath(kind, namespace, name string) (string, error) {
	switch kind {
	case "Deployment":
		return fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments/%s", namespace, name), nil
	case "DaemonSet":
		return fmt.Sprintf("/apis/apps/v1/namespaces/%s/daemonsets/%s", namespace, name), nil
	case "StatefulSet":
		return fmt.Sprintf("/apis/apps/v1/namespaces/%s/statefulsets/%s", namespace, name), nil
	case "Job":
		return fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs/%s", namespace, name), nil
	case "CronJob":
		return fmt.Sprintf("/apis/batch/v1/namespaces/%s/cronjobs/%s", namespace, name), nil
	case "Pod":
		return fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", namespace, name), nil
	default:
		return "", fmt.Errorf("unsupported workload kind: %s", kind)
	}
}

func createResourcePath(object map[string]any) (string, error) {
	kind, _ := object["kind"].(string)
	metadata, _ := object["metadata"].(map[string]any)
	namespace := ""
	if metadata != nil {
		if value, ok := metadata["namespace"].(string); ok {
			namespace = strings.TrimSpace(value)
		}
	}
	switch kind {
	case "Namespace":
		return "/api/v1/namespaces", nil
	case "ServiceAccount":
		if namespace == "" {
			return "", errors.New("namespace is required for ServiceAccount")
		}
		return fmt.Sprintf("/api/v1/namespaces/%s/serviceaccounts", namespace), nil
	case "ConfigMap":
		if namespace == "" {
			return "", errors.New("namespace is required for ConfigMap")
		}
		return fmt.Sprintf("/api/v1/namespaces/%s/configmaps", namespace), nil
	case "PersistentVolumeClaim":
		if namespace == "" {
			return "", errors.New("namespace is required for PersistentVolumeClaim")
		}
		return fmt.Sprintf("/api/v1/namespaces/%s/persistentvolumeclaims", namespace), nil
	case "Service":
		if namespace == "" {
			return "", errors.New("namespace is required for Service")
		}
		return fmt.Sprintf("/api/v1/namespaces/%s/services", namespace), nil
	case "Deployment":
		if namespace == "" {
			return "", errors.New("namespace is required for Deployment")
		}
		return fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments", namespace), nil
	case "StatefulSet":
		if namespace == "" {
			return "", errors.New("namespace is required for StatefulSet")
		}
		return fmt.Sprintf("/apis/apps/v1/namespaces/%s/statefulsets", namespace), nil
	case "DaemonSet":
		if namespace == "" {
			return "", errors.New("namespace is required for DaemonSet")
		}
		return fmt.Sprintf("/apis/apps/v1/namespaces/%s/daemonsets", namespace), nil
	case "Job":
		if namespace == "" {
			return "", errors.New("namespace is required for Job")
		}
		return fmt.Sprintf("/apis/batch/v1/namespaces/%s/jobs", namespace), nil
	case "CronJob":
		if namespace == "" {
			return "", errors.New("namespace is required for CronJob")
		}
		return fmt.Sprintf("/apis/batch/v1/namespaces/%s/cronjobs", namespace), nil
	case "Ingress":
		if namespace == "" {
			return "", errors.New("namespace is required for Ingress")
		}
		return fmt.Sprintf("/apis/networking.k8s.io/v1/namespaces/%s/ingresses", namespace), nil
	default:
		return "", fmt.Errorf("unsupported resource kind for create: %s", kind)
	}
}

func namespaceResourcePath(kind, namespace, name string) (string, error) {
	switch kind {
	case "ServiceAccount":
		return fmt.Sprintf("/api/v1/namespaces/%s/serviceaccounts/%s", namespace, name), nil
	case "PersistentVolumeClaim":
		return fmt.Sprintf("/api/v1/namespaces/%s/persistentvolumeclaims/%s", namespace, name), nil
	case "Service":
		return fmt.Sprintf("/api/v1/namespaces/%s/services/%s", namespace, name), nil
	case "Ingress":
		return fmt.Sprintf("/apis/networking.k8s.io/v1/namespaces/%s/ingresses/%s", namespace, name), nil
	case "ConfigMap":
		return fmt.Sprintf("/api/v1/namespaces/%s/configmaps/%s", namespace, name), nil
	case "Secret":
		return fmt.Sprintf("/api/v1/namespaces/%s/secrets/%s", namespace, name), nil
	case "ResourceQuota":
		return fmt.Sprintf("/api/v1/namespaces/%s/resourcequotas/%s", namespace, name), nil
	case "LimitRange":
		return fmt.Sprintf("/api/v1/namespaces/%s/limitranges/%s", namespace, name), nil
	default:
		return "", fmt.Errorf("unsupported resource kind: %s", kind)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func ternaryString(condition bool, truthy, falsy string) string {
	if condition {
		return truthy
	}
	return falsy
}

func sortedKeys(items map[string]string) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func labelSelector(matchLabels map[string]string) string {
	if len(matchLabels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(matchLabels))
	for key, value := range matchLabels {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, ",")
}

func stringifyAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%.0f", typed)
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	case nil:
		return "-"
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func extractMap(source map[string]any, path ...string) map[string]any {
	current := source
	for _, key := range path {
		next, _ := current[key].(map[string]any)
		if next == nil {
			return nil
		}
		current = next
	}
	return current
}

func extractArray(source map[string]any, path ...string) []any {
	if len(path) == 0 {
		return nil
	}
	parent := source
	if len(path) > 1 {
		parent = extractMap(source, path[:len(path)-1]...)
	}
	if parent == nil {
		return nil
	}
	items, _ := parent[path[len(path)-1]].([]any)
	return items
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func anyStringSlice(value any) []string {
	raw, _ := value.([]any)
	items := make([]string, 0, len(raw))
	for _, item := range raw {
		text := stringValue(item)
		if text != "" {
			items = append(items, text)
		}
	}
	return uniqueStrings(items)
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func extractStringMap(source map[string]any, path ...string) map[string]string {
	current := extractMap(source, path...)
	if current == nil {
		return nil
	}
	out := make(map[string]string, len(current))
	for key, value := range current {
		if text, ok := value.(string); ok {
			out[key] = text
		}
	}
	return out
}

func collectWorkloadRefs(podSpec map[string]any) workloadResourceRefs {
	refs := workloadResourceRefs{
		ConfigMaps:     []string{},
		Secrets:        []string{},
		PersistentPVCs: []string{},
		ServiceAccount: "default",
		ClaimTemplates: []statefulSetClaimTemplateItem{},
	}
	if podSpec == nil {
		return refs
	}
	refs.ServiceAccount = firstNonEmpty(stringValue(podSpec["serviceAccountName"]), "default")
	if volumes, ok := podSpec["volumes"].([]any); ok {
		for _, item := range volumes {
			volume, _ := item.(map[string]any)
			if volume == nil {
				continue
			}
			if configMap, _ := volume["configMap"].(map[string]any); configMap != nil {
				if name, _ := configMap["name"].(string); strings.TrimSpace(name) != "" {
					refs.ConfigMaps = append(refs.ConfigMaps, name)
				}
			}
			if secret, _ := volume["secret"].(map[string]any); secret != nil {
				if name, _ := secret["secretName"].(string); strings.TrimSpace(name) != "" {
					refs.Secrets = append(refs.Secrets, name)
				}
			}
			if persistentVolumeClaim, _ := volume["persistentVolumeClaim"].(map[string]any); persistentVolumeClaim != nil {
				if name, _ := persistentVolumeClaim["claimName"].(string); strings.TrimSpace(name) != "" {
					refs.PersistentPVCs = append(refs.PersistentPVCs, name)
				}
			}
		}
	}
	if containers, ok := podSpec["containers"].([]any); ok {
		for _, item := range containers {
			container, _ := item.(map[string]any)
			if container == nil {
				continue
			}
			refs = collectContainerRefs(container, refs)
		}
	}
	refs.ConfigMaps = uniqueStrings(refs.ConfigMaps)
	refs.Secrets = uniqueStrings(refs.Secrets)
	refs.PersistentPVCs = uniqueStrings(refs.PersistentPVCs)
	return refs
}

func extractClaimTemplates(spec map[string]any) []statefulSetClaimTemplateItem {
	rawTemplates, _ := spec["volumeClaimTemplates"].([]any)
	items := make([]statefulSetClaimTemplateItem, 0, len(rawTemplates))
	for _, raw := range rawTemplates {
		template, _ := raw.(map[string]any)
		if template == nil {
			continue
		}
		metadata, _ := template["metadata"].(map[string]any)
		templateSpec, _ := template["spec"].(map[string]any)
		if metadata == nil || templateSpec == nil {
			continue
		}
		name := stringValue(metadata["name"])
		if name == "" {
			continue
		}
		resources, _ := templateSpec["resources"].(map[string]any)
		requests, _ := resources["requests"].(map[string]any)
		items = append(items, statefulSetClaimTemplateItem{
			Name:             name,
			StorageClassName: stringValue(templateSpec["storageClassName"]),
			RequestedStorage: stringValue(requests["storage"]),
			AccessModes:      anyStringSlice(templateSpec["accessModes"]),
			Labels:           selectorPairs(extractStringMap(metadata, "labels")),
			Annotations:      selectorPairs(extractStringMap(metadata, "annotations")),
		})
	}
	slices.SortFunc(items, func(a, b statefulSetClaimTemplateItem) int {
		return strings.Compare(a.Name, b.Name)
	})
	return items
}

func collectContainerRefs(container map[string]any, refs workloadResourceRefs) workloadResourceRefs {
	if envFrom, ok := container["envFrom"].([]any); ok {
		for _, item := range envFrom {
			entry, _ := item.(map[string]any)
			if entry == nil {
				continue
			}
			if configMapRef, _ := entry["configMapRef"].(map[string]any); configMapRef != nil {
				if name, _ := configMapRef["name"].(string); strings.TrimSpace(name) != "" {
					refs.ConfigMaps = append(refs.ConfigMaps, name)
				}
			}
			if secretRef, _ := entry["secretRef"].(map[string]any); secretRef != nil {
				if name, _ := secretRef["name"].(string); strings.TrimSpace(name) != "" {
					refs.Secrets = append(refs.Secrets, name)
				}
			}
		}
	}
	if envs, ok := container["env"].([]any); ok {
		for _, item := range envs {
			env, _ := item.(map[string]any)
			if env == nil {
				continue
			}
			valueFrom, _ := env["valueFrom"].(map[string]any)
			if valueFrom == nil {
				continue
			}
			if configMapKeyRef, _ := valueFrom["configMapKeyRef"].(map[string]any); configMapKeyRef != nil {
				if name, _ := configMapKeyRef["name"].(string); strings.TrimSpace(name) != "" {
					refs.ConfigMaps = append(refs.ConfigMaps, name)
				}
			}
			if secretKeyRef, _ := valueFrom["secretKeyRef"].(map[string]any); secretKeyRef != nil {
				if name, _ := secretKeyRef["name"].(string); strings.TrimSpace(name) != "" {
					refs.Secrets = append(refs.Secrets, name)
				}
			}
		}
	}
	return refs
}

func primaryContainerName(kind string, object map[string]any) (string, error) {
	containers, err := workloadContainerImages(kind, object)
	if err != nil {
		return "", err
	}
	if len(containers) == 0 {
		return "", errors.New("no containers found in current workload")
	}
	return containers[0].Name, nil
}

func workloadPrimaryImagePatch(kind, containerName, image string) map[string]any {
	return workloadContainerImagesPatch(kind, []containerImageItem{{Name: containerName, Image: image}})
}

func workloadContainerImagesPatch(kind string, containers []containerImageItem) map[string]any {
	containerPatch := make([]map[string]any, 0, len(containers))
	for _, item := range containers {
		containerPatch = append(containerPatch, map[string]any{
			"name":  item.Name,
			"image": item.Image,
		})
	}
	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet", "Job":
		return map[string]any{
			"spec": map[string]any{
				"template": map[string]any{
					"spec": map[string]any{
						"containers": containerPatch,
					},
				},
			},
		}
	case "CronJob":
		return map[string]any{
			"spec": map[string]any{
				"jobTemplate": map[string]any{
					"spec": map[string]any{
						"template": map[string]any{
							"spec": map[string]any{
								"containers": containerPatch,
							},
						},
					},
				},
			},
		}
	default:
		return map[string]any{}
	}
}

func workloadContainerImages(kind string, object map[string]any) ([]containerImageItem, error) {
	spec, _ := object["spec"].(map[string]any)
	if spec == nil {
		return nil, errors.New("workload spec not found")
	}
	var containers []any
	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet", "Job":
		containers = extractArray(spec, "template", "spec", "containers")
	case "CronJob":
		containers = extractArray(spec, "jobTemplate", "spec", "template", "spec", "containers")
	case "Pod":
		containers = extractArray(spec, "containers")
	default:
		return nil, fmt.Errorf("image update is unsupported for workload kind: %s", kind)
	}
	items := make([]containerImageItem, 0, len(containers))
	for _, raw := range containers {
		container, _ := raw.(map[string]any)
		name := stringValue(container["name"])
		image := stringValue(container["image"])
		if name == "" {
			continue
		}
		items = append(items, containerImageItem{Name: name, Image: image})
	}
	return items, nil
}

func selectorMatches(serviceSelector, workloadSelector map[string]string) bool {
	if len(serviceSelector) == 0 || len(workloadSelector) == 0 {
		return false
	}
	for key, value := range serviceSelector {
		if workloadSelector[key] != value {
			return false
		}
	}
	return true
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func selectorPairs(selector map[string]string) []string {
	if len(selector) == 0 {
		return nil
	}
	items := make([]string, 0, len(selector))
	for key, value := range selector {
		items = append(items, fmt.Sprintf("%s=%s", key, value))
	}
	return uniqueStrings(items)
}

func loadBalancerAddresses(items []struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
}) []string {
	addresses := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.IP) != "" {
			addresses = append(addresses, item.IP)
		}
		if strings.TrimSpace(item.Hostname) != "" {
			addresses = append(addresses, item.Hostname)
		}
	}
	return uniqueStrings(addresses)
}

func stringifyPort(number int, name string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	if number > 0 {
		return fmt.Sprintf("%d", number)
	}
	return ""
}

func stringifyBackend(serviceName string, number int, name string) string {
	if strings.TrimSpace(serviceName) == "" {
		return "-"
	}
	port := stringifyPort(number, name)
	if port == "" {
		return serviceName
	}
	return fmt.Sprintf("%s:%s", serviceName, port)
}

func resolveExecToken(command string, args []string, envItems []execEnvItem) (string, error) {
	base := command
	if idx := strings.LastIndex(command, "/"); idx >= 0 {
		base = command[idx+1:]
	}
	if base != "aws" {
		return "", fmt.Errorf("kubeconfig uses exec auth (%s); current version supports aws eks get-token style exec only", base)
	}

	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	for _, item := range envItems {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", item.Name, item.Value))
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("aws exec auth failed: %s", message)
	}

	var payload execCredentialOutput
	if err := json.Unmarshal(output, &payload); err != nil {
		return "", fmt.Errorf("invalid exec credential output: %w", err)
	}
	if payload.Status.Token == "" {
		return "", errors.New("exec auth completed without returning token")
	}
	return payload.Status.Token, nil
}
