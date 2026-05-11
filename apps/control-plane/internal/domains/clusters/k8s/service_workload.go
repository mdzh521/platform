package k8s

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

func (s *Service) CreateWorkloadBundle(input CreateWorkloadBundleInput) (map[string]any, error) {
	if strings.TrimSpace(input.Namespace) == "" {
		return nil, errors.New("namespace is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("name is required")
	}
	cluster, client, err := s.resolveClusterClient(input.ClusterID)
	if err != nil {
		return nil, err
	}
	objects, createdKinds, err := buildWorkloadBundle(input)
	if err != nil {
		return nil, err
	}
	if err := client.CreateObjects(objects); err != nil {
		return nil, err
	}
	if _, err := s.SyncWorkloads(cluster.ID); err != nil {
		return nil, err
	}
	s.publishEvent(context.Background(), "k8s.workload.bundle.created", map[string]any{
		"cluster_id":    cluster.ID,
		"cluster_name":  cluster.Name,
		"namespace":     input.Namespace,
		"name":          input.Name,
		"workload_type": input.WorkloadType,
		"created_kinds": createdKinds,
	})
	return map[string]any{
		"cluster_id":        cluster.ID,
		"cluster_name":      cluster.Name,
		"namespace":         input.Namespace,
		"name":              input.Name,
		"created_kinds":     createdKinds,
		"created_resources": extractCreatedResources(objects),
		"message":           "workload bundle created",
	}, nil
}

func (s *Service) ApplyManifest(input ApplyManifestInput) (map[string]any, error) {
	cluster, client, err := s.resolveClusterClient(input.ClusterID)
	if err != nil {
		return nil, err
	}
	objects, appliedKinds, err := parseManifestDocuments(input.ManifestYAML, input.Namespace)
	if err != nil {
		return nil, err
	}
	if len(objects) == 0 {
		return nil, errors.New("no manifest documents found")
	}
	if err := client.CreateObjects(objects); err != nil {
		return nil, err
	}
	if _, err := s.SyncNamespaces(cluster.ID); err != nil {
		return nil, err
	}
	if _, err := s.SyncWorkloads(cluster.ID); err != nil {
		return nil, err
	}
	s.publishEvent(context.Background(), "k8s.manifest.applied", map[string]any{
		"cluster_id":     cluster.ID,
		"cluster_name":   cluster.Name,
		"namespace":      input.Namespace,
		"applied_kinds":  appliedKinds,
		"document_count": len(objects),
	})
	return map[string]any{
		"cluster_id":        cluster.ID,
		"cluster_name":      cluster.Name,
		"applied_kinds":     appliedKinds,
		"created_resources": extractCreatedResources(objects),
		"document_count":    len(objects),
		"message":           "manifest applied",
	}, nil
}

func (s *Service) ListWorkloads(clusterID *uint, namespace string, includePods bool) ([]WorkloadListItem, error) {
	query := s.db.Table("workloads").
		Select("id, cluster_id, namespace_id, namespace_name, kind, name, ready_replicas, replicas, image, status, source_type, updated_at").
		Order("namespace_name asc, kind asc, name asc")
	if clusterID != nil && *clusterID > 0 {
		query = query.Where("cluster_id = ?", *clusterID)
	}
	if strings.TrimSpace(namespace) != "" {
		query = query.Where("namespace_name = ?", namespace)
	}
	if !includePods {
		query = query.Where("kind <> ?", "Pod")
	}

	var items []WorkloadListItem
	if err := query.Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) GetWorkload(id uint) (*WorkloadDetail, error) {
	var item WorkloadDetail
	err := s.db.Table("workloads").
		Select("workloads.id, workloads.cluster_id, workloads.namespace_id, workloads.namespace_name, workloads.kind, workloads.name, workloads.ready_replicas, workloads.replicas, workloads.image, workloads.status, workloads.source_type, workloads.updated_at, clusters.name as cluster_name, clusters.code as cluster_code, clusters.provider, clusters.environment").
		Joins("join clusters on clusters.id = workloads.cluster_id").
		Where("workloads.id = ?", id).
		Scan(&item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

func (s *Service) GetWorkloadManifest(id uint, mode string) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	object, err := client.GetWorkloadObject(workload.Kind, workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	if mode == "compact" {
		object = compactManifest(object)
	}
	data, err := yaml.Marshal(object)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"workload_id":   workload.ID,
		"cluster_id":    cluster.ID,
		"cluster_name":  cluster.Name,
		"namespace":     workload.NamespaceName,
		"kind":          workload.Kind,
		"name":          workload.Name,
		"mode":          mode,
		"manifest_yaml": string(data),
	}, nil
}

func (s *Service) GetWorkloadEvents(id uint) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	events, err := client.GetWorkloadEvents(workload.Kind, workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	return map[string]any{
		"workload_id":  workload.ID,
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    workload.NamespaceName,
		"kind":         workload.Kind,
		"name":         workload.Name,
		"events":       events,
	}, nil
}

func (s *Service) GetWorkloadPods(id uint) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	pods, err := client.GetRelatedPods(workload.Kind, workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	items := make([]RelatedPod, 0, len(pods))
	for _, item := range pods {
		items = append(items, RelatedPod{
			Name:   item.Name,
			Status: item.Status,
		})
	}
	return map[string]any{
		"workload_id":  workload.ID,
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    workload.NamespaceName,
		"kind":         workload.Kind,
		"name":         workload.Name,
		"pods":         items,
	}, nil
}

func (s *Service) GetWorkloadLogs(id uint, tailLines int, podName string) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	pods, err := client.GetRelatedPods(workload.Kind, workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	if len(pods) == 0 {
		return nil, errors.New("no related pods found for current workload")
	}
	selectedPod, err := selectLogPod(pods, podName)
	if err != nil {
		return nil, err
	}
	logs, err := client.GetPodLogs(selectedPod.Namespace, selectedPod.Name, tailLines)
	if err != nil {
		return nil, err
	}
	items := make([]RelatedPod, 0, len(pods))
	for _, item := range pods {
		items = append(items, RelatedPod{
			Name:   item.Name,
			Status: item.Status,
		})
	}
	return map[string]any{
		"workload_id":  workload.ID,
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    workload.NamespaceName,
		"kind":         workload.Kind,
		"name":         workload.Name,
		"tail_lines":   tailLines,
		"source_pod":   selectedPod.Name,
		"pods":         items,
		"logs":         logs,
	}, nil
}

func (s *Service) ExecWorkloadCommand(id uint, input ExecWorkloadCommandInput) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	pods, err := client.GetRelatedPods(workload.Kind, workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	if len(pods) == 0 {
		return nil, errors.New("no related pods found for current workload")
	}
	selectedPod, err := selectLogPod(pods, input.PodName)
	if err != nil {
		return nil, err
	}
	commandArgs := parseExecCommand(input.Command)
	if len(commandArgs) == 0 {
		return nil, errors.New("command is required")
	}
	timeoutSeconds := input.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 15
	}
	if timeoutSeconds > 120 {
		timeoutSeconds = 120
	}
	output, execErr := executePodCommand(cluster, selectedPod.Namespace, selectedPod.Name, input.ContainerName, commandArgs, time.Duration(timeoutSeconds)*time.Second)
	result := map[string]any{
		"workload_id":     workload.ID,
		"cluster_id":      cluster.ID,
		"cluster_name":    cluster.Name,
		"namespace":       workload.NamespaceName,
		"kind":            workload.Kind,
		"name":            workload.Name,
		"source_pod":      selectedPod.Name,
		"container_name":  strings.TrimSpace(input.ContainerName),
		"command":         commandArgs,
		"timeout_seconds": timeoutSeconds,
		"output":          output,
		"success":         execErr == nil,
	}
	if execErr != nil {
		result["error_message"] = execErr.Error()
		result["message"] = "command execution finished with error"
		return result, nil
	}
	result["message"] = "command executed"
	return result, nil
}

func (s *Service) GetWorkloadRollout(id uint) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	rollout, err := client.GetWorkloadRollout(workload.Kind, workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	replicaSets := make([]RolloutReplicaSet, 0, len(rollout.ReplicaSets))
	for _, item := range rollout.ReplicaSets {
		replicaSets = append(replicaSets, RolloutReplicaSet{
			Name:          item.Name,
			Desired:       item.Replicas,
			Ready:         item.ReadyReplicas,
			Available:     item.AvailableReplicas,
			Revision:      item.Revision,
			CreationStamp: item.CreationTimestamp,
		})
	}
	return map[string]any{
		"workload_id":          workload.ID,
		"cluster_id":           cluster.ID,
		"cluster_name":         cluster.Name,
		"namespace":            workload.NamespaceName,
		"kind":                 workload.Kind,
		"name":                 workload.Name,
		"strategy":             rollout.Strategy,
		"max_surge":            rollout.MaxSurge,
		"max_unavailable":      rollout.MaxUnavailable,
		"observed_generation":  rollout.ObservedGeneration,
		"generation":           rollout.Generation,
		"ready_replicas":       rollout.ReadyReplicas,
		"updated_replicas":     rollout.UpdatedReplicas,
		"available_replicas":   rollout.AvailableReplicas,
		"unavailable_replicas": rollout.UnavailableReplicas,
		"current_replicas":     rollout.CurrentReplicas,
		"desired_replicas":     rollout.DesiredReplicas,
		"current_revision":     rollout.CurrentRevision,
		"update_revision":      rollout.UpdateRevision,
		"conditions":           rollout.Conditions,
		"replica_sets":         replicaSets,
	}, nil
}

func (s *Service) GetWorkloadResources(id uint) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	services, ingresses, refs, err := client.GetWorkloadResources(workload.Kind, workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	persistentVolumeClaims, err := client.GetWorkloadPersistentVolumeClaims(workload.Kind, workload.NamespaceName, workload.Name, refs)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	serviceItems := make([]RelatedService, 0, len(services))
	for _, item := range services {
		serviceItems = append(serviceItems, RelatedService{
			Name:      item.Name,
			Type:      item.Type,
			ClusterIP: item.ClusterIP,
			Ports:     item.Ports,
			Selector:  item.Selector,
			Endpoints: item.Endpoints,
		})
	}
	ingressItems := make([]RelatedIngress, 0, len(ingresses))
	for _, item := range ingresses {
		ingressItems = append(ingressItems, RelatedIngress{
			Name:      item.Name,
			Hosts:     item.Hosts,
			Paths:     item.Paths,
			Backends:  item.Backends,
			Addresses: item.Addresses,
		})
	}
	persistentVolumeClaimItems := make([]RelatedPersistentVolumeClaim, 0, len(persistentVolumeClaims))
	for _, item := range persistentVolumeClaims {
		persistentVolumeClaimItems = append(persistentVolumeClaimItems, RelatedPersistentVolumeClaim{
			Name:         item.Name,
			Status:       item.Status,
			Volume:       item.VolumeName,
			StorageClass: item.StorageClassName,
			AccessModes:  item.AccessModes,
			Requested:    item.RequestedStorage,
			Capacity:     item.Capacity,
			TemplateName: item.ClaimTemplate,
		})
	}
	claimTemplates := make([]StatefulSetClaimTemplate, 0, len(refs.ClaimTemplates))
	for _, item := range refs.ClaimTemplates {
		claimTemplates = append(claimTemplates, StatefulSetClaimTemplate{
			Name:         item.Name,
			StorageClass: item.StorageClassName,
			AccessModes:  item.AccessModes,
			Requested:    item.RequestedStorage,
		})
	}
	return map[string]any{
		"workload_id":                 workload.ID,
		"cluster_id":                  cluster.ID,
		"cluster_name":                cluster.Name,
		"namespace":                   workload.NamespaceName,
		"kind":                        workload.Kind,
		"name":                        workload.Name,
		"services":                    serviceItems,
		"ingresses":                   ingressItems,
		"config_maps":                 refs.ConfigMaps,
		"secrets":                     refs.Secrets,
		"service_account":             refs.ServiceAccount,
		"persistent_volume_claims":    persistentVolumeClaimItems,
		"statefulset_claim_templates": claimTemplates,
	}, nil
}

func (s *Service) GetWorkloadServiceDetail(id uint, serviceName string) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	detail, err := client.GetServiceDetail(workload.NamespaceName, serviceName)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	return map[string]any{
		"workload_id":  workload.ID,
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    workload.NamespaceName,
		"kind":         workload.Kind,
		"name":         workload.Name,
		"service": ServiceDetail{
			Name:            detail.Name,
			Type:            detail.Type,
			ClusterIP:       detail.ClusterIP,
			SessionAffinity: detail.SessionAffinity,
			Selector:        detail.Selector,
			ExternalIPs:     detail.ExternalIPs,
			Ports:           detail.Ports,
			Endpoints:       detail.Endpoints,
		},
	}, nil
}

func (s *Service) GetWorkloadIngressDetail(id uint, ingressName string) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	detail, err := client.GetIngressDetail(workload.NamespaceName, ingressName)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	return map[string]any{
		"workload_id":  workload.ID,
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    workload.NamespaceName,
		"kind":         workload.Kind,
		"name":         workload.Name,
		"ingress": IngressDetail{
			Name:           detail.Name,
			IngressClass:   detail.IngressClass,
			DefaultBackend: detail.DefaultBackend,
			Addresses:      detail.Addresses,
			Rules:          detail.Rules,
			TLS:            detail.TLS,
			Annotations:    detail.Annotations,
		},
	}, nil
}

func (s *Service) ScaleWorkload(id uint, replicas int) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	if workload.Kind != "Deployment" && workload.Kind != "StatefulSet" {
		return nil, errors.New("scale is only supported for Deployment and StatefulSet")
	}
	result, err := client.ScaleWorkload(workload.Kind, workload.NamespaceName, workload.Name, replicas)
	if err != nil {
		return nil, err
	}
	if err := s.refreshWorkloadState(cluster.ID, workload.Kind, workload.NamespaceName, workload.Name); err != nil {
		return nil, err
	}
	return map[string]any{
		"workload_id":  workload.ID,
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    workload.NamespaceName,
		"kind":         workload.Kind,
		"name":         workload.Name,
		"replicas":     result.Replicas,
		"message":      "workload scaled",
	}, nil
}

func (s *Service) UpdateWorkloadPrimaryImage(id uint, image string) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	if err := client.UpdateWorkloadPrimaryImage(workload.Kind, workload.NamespaceName, workload.Name, image); err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	if err := s.refreshWorkloadState(cluster.ID, workload.Kind, workload.NamespaceName, workload.Name); err != nil {
		return nil, err
	}
	return map[string]any{
		"workload_id":  workload.ID,
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    workload.NamespaceName,
		"kind":         workload.Kind,
		"name":         workload.Name,
		"image":        image,
		"message":      "workload image updated",
	}, nil
}

func (s *Service) GetWorkloadContainerImages(id uint) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	containers, err := client.GetWorkloadContainerImages(workload.Kind, workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	return map[string]any{
		"workload_id":  workload.ID,
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    workload.NamespaceName,
		"kind":         workload.Kind,
		"name":         workload.Name,
		"containers":   containers,
	}, nil
}

func (s *Service) UpdateWorkloadImages(id uint, input UpdateWorkloadImagesInput) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	if len(input.Containers) == 0 {
		return nil, errors.New("containers are required")
	}
	updates := make([]containerImageItem, 0, len(input.Containers))
	for _, item := range input.Containers {
		if strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Image) == "" {
			return nil, errors.New("container name and image are required")
		}
		updates = append(updates, containerImageItem{Name: strings.TrimSpace(item.Name), Image: strings.TrimSpace(item.Image)})
	}
	if err := client.UpdateWorkloadContainerImages(workload.Kind, workload.NamespaceName, workload.Name, updates); err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	if err := s.refreshWorkloadState(cluster.ID, workload.Kind, workload.NamespaceName, workload.Name); err != nil {
		return nil, err
	}
	return map[string]any{
		"workload_id":  workload.ID,
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    workload.NamespaceName,
		"kind":         workload.Kind,
		"name":         workload.Name,
		"containers":   updates,
		"message":      "workload container images updated",
	}, nil
}

func (s *Service) GetStatefulSetSettings(id uint) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	if workload.Kind != "StatefulSet" {
		return nil, errors.New("current workload is not a StatefulSet")
	}
	settings, err := client.GetStatefulSetSettings(workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	return map[string]any{
		"workload_id":           workload.ID,
		"cluster_id":            cluster.ID,
		"cluster_name":          cluster.Name,
		"namespace":             workload.NamespaceName,
		"name":                  workload.Name,
		"service_name":          settings.ServiceName,
		"service_mode":          settings.ServiceMode,
		"pod_management_policy": settings.PodManagementPolicy,
		"rolling_partition":     settings.RollingPartition,
		"update_strategy":       settings.UpdateStrategy,
	}, nil
}

func parseExecCommand(command string) []string {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func executePodCommand(cluster *Cluster, namespace, podName, containerName string, commandArgs []string, timeout time.Duration) (string, error) {
	kubeconfigPath, err := writeTemporaryKubeconfig(*cluster)
	if err != nil {
		return "", err
	}
	defer os.Remove(kubeconfigPath)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{"--kubeconfig", kubeconfigPath, "exec", "-n", namespace, podName}
	if strings.TrimSpace(containerName) != "" {
		args = append(args, "-c", strings.TrimSpace(containerName))
	}
	args = append(args, "--")
	args = append(args, commandArgs...)

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if ctx.Err() == context.DeadlineExceeded {
		if text == "" {
			text = "command timed out"
		}
		return text, fmt.Errorf("command timed out after %s", timeout)
	}
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return text, fmt.Errorf("kubectl exec failed: %s", text)
	}
	return text, nil
}

func writeTemporaryKubeconfig(cluster Cluster) (string, error) {
	raw := extractStoredCredential(cluster.CredentialJSON)
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("cluster credential is empty")
	}
	content := raw
	if cluster.AuthType == "token" {
		credential, err := parseTokenCredential(cluster)
		if err != nil {
			return "", err
		}
		kubeconfig := map[string]any{
			"apiVersion": "v1",
			"kind":       "Config",
			"clusters": []map[string]any{{
				"name": "platform-center",
				"cluster": map[string]any{
					"server":                   cluster.APIEndpoint,
					"insecure-skip-tls-verify": credential.InsecureSkipVerify,
				},
			}},
			"contexts": []map[string]any{{
				"name": "platform-center",
				"context": map[string]any{
					"cluster": "platform-center",
					"user":    "platform-center",
				},
			}},
			"current-context": "platform-center",
			"users": []map[string]any{{
				"name": "platform-center",
				"user": map[string]any{
					"token": credential.Token,
				},
			}},
		}
		if strings.TrimSpace(credential.CACert) != "" {
			kubeconfig["clusters"].([]map[string]any)[0]["cluster"].(map[string]any)["certificate-authority-data"] = credential.CACert
		}
		data, err := yaml.Marshal(kubeconfig)
		if err != nil {
			return "", err
		}
		content = string(data)
	} else if cluster.AuthType == "kubeconfig" {
		if sanitized, err := sanitizeKubeconfigForKubectl(raw); err == nil {
			content = sanitized
		}
	}
	path := filepath.Join(os.TempDir(), fmt.Sprintf("platform-center-%d-%d-kubeconfig.yaml", cluster.ID, time.Now().UnixNano()))
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func sanitizeKubeconfigForKubectl(raw string) (string, error) {
	var payload map[string]any
	if err := yaml.Unmarshal([]byte(raw), &payload); err != nil {
		return "", err
	}
	clusterItems, ok := payload["clusters"].([]any)
	if ok {
		for _, item := range clusterItems {
			clusterItem, ok := item.(map[string]any)
			if !ok {
				continue
			}
			clusterData, ok := clusterItem["cluster"].(map[string]any)
			if !ok {
				continue
			}
			if skip, ok := clusterData["insecure-skip-tls-verify"].(bool); ok && skip {
				delete(clusterData, "certificate-authority-data")
				delete(clusterData, "certificate-authority")
			}
		}
	}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Service) UpdateStatefulSetSettings(id uint, input UpdateStatefulSetSettingsInput) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	if workload.Kind != "StatefulSet" {
		return nil, errors.New("current workload is not a StatefulSet")
	}
	if err := client.UpdateStatefulSetSettings(workload.NamespaceName, workload.Name, input.ServiceMode, input.PodManagementPolicy, input.RollingPartition); err != nil {
		return nil, s.handleMissingRemoteWorkload(workload, err)
	}
	if err := s.refreshWorkloadState(cluster.ID, workload.Kind, workload.NamespaceName, workload.Name); err != nil {
		return nil, err
	}
	return map[string]any{
		"workload_id":           workload.ID,
		"cluster_id":            cluster.ID,
		"cluster_name":          cluster.Name,
		"namespace":             workload.NamespaceName,
		"name":                  workload.Name,
		"service_mode":          input.ServiceMode,
		"pod_management_policy": input.PodManagementPolicy,
		"rolling_partition":     input.RollingPartition,
		"message":               "statefulset settings updated",
	}, nil
}

func (s *Service) RestartWorkload(id uint) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	result, err := client.RestartWorkload(workload.Kind, workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, err
	}
	if err := s.refreshWorkloadState(cluster.ID, workload.Kind, workload.NamespaceName, workload.Name); err != nil {
		return nil, err
	}
	return map[string]any{
		"workload_id":   workload.ID,
		"cluster_id":    cluster.ID,
		"cluster_name":  cluster.Name,
		"namespace":     workload.NamespaceName,
		"kind":          workload.Kind,
		"name":          workload.Name,
		"deleted_pods":  result.DeletedPods,
		"deleted_count": len(result.DeletedPods),
		"message":       "workload pods restarted",
	}, nil
}

func (s *Service) RolloutRestartWorkload(id uint) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	if workload.Kind != "Deployment" && workload.Kind != "StatefulSet" {
		return nil, errors.New("rollout restart is only supported for Deployment and StatefulSet")
	}
	result, err := client.RolloutRestartWorkload(workload.Kind, workload.NamespaceName, workload.Name)
	if err != nil {
		return nil, err
	}
	if err := s.refreshWorkloadState(cluster.ID, workload.Kind, workload.NamespaceName, workload.Name); err != nil {
		return nil, err
	}
	return map[string]any{
		"workload_id":  workload.ID,
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    workload.NamespaceName,
		"kind":         workload.Kind,
		"name":         workload.Name,
		"restarted_at": result.RestartedAt,
		"message":      "workload rollout restarted",
	}, nil
}

func (s *Service) DeleteWorkload(id uint) (map[string]any, error) {
	workload, cluster, client, err := s.resolveWorkloadClient(id)
	if err != nil {
		return nil, err
	}
	if err := client.DeleteWorkload(workload.Kind, workload.NamespaceName, workload.Name); err != nil {
		return nil, err
	}
	if err := s.db.Delete(&Workload{}, workload.ID).Error; err != nil {
		return nil, err
	}
	if _, err := s.SyncWorkloads(cluster.ID); err != nil {
		return nil, err
	}
	return map[string]any{
		"workload_id":  workload.ID,
		"cluster_id":   cluster.ID,
		"cluster_name": cluster.Name,
		"namespace":    workload.NamespaceName,
		"kind":         workload.Kind,
		"name":         workload.Name,
		"message":      "workload deleted",
	}, nil
}
