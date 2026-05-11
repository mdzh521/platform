package k8s

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

func (c *clusterClient) ListWorkloads() ([]workloadRemoteItem, error) {
	deployments, err := c.listDeployments()
	if err != nil {
		return nil, err
	}
	daemonSets, err := c.listDaemonSets()
	if err != nil {
		return nil, err
	}
	statefulSets, err := c.listStatefulSets()
	if err != nil {
		return nil, err
	}
	jobs, err := c.listJobs()
	if err != nil {
		return nil, err
	}
	cronJobs, err := c.listCronJobs()
	if err != nil {
		return nil, err
	}
	pods, err := c.listPods()
	if err != nil {
		return nil, err
	}
	items := make([]workloadRemoteItem, 0, len(deployments)+len(daemonSets)+len(statefulSets)+len(jobs)+len(cronJobs)+len(pods))
	items = append(items, deployments...)
	items = append(items, daemonSets...)
	items = append(items, statefulSets...)
	items = append(items, jobs...)
	items = append(items, cronJobs...)
	items = append(items, pods...)
	return items, nil
}

func (c *clusterClient) GetWorkloadObject(kind, namespace, name string) (map[string]any, error) {
	path, err := workloadResourcePath(kind, namespace, name)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := c.getJSON(path, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *clusterClient) GetWorkloadEvents(kind, namespace, name string) ([]map[string]any, error) {
	query := url.Values{}
	query.Set("fieldSelector", fmt.Sprintf("involvedObject.kind=%s,involvedObject.name=%s,involvedObject.namespace=%s", kind, name, namespace))
	var payload eventsPayload
	if err := c.getJSON("/api/v1/namespaces/"+namespace+"/events?"+query.Encode(), &payload); err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(payload.Items))
	for _, item := range payload.Items {
		timestamp := firstNonEmpty(item.EventTime, item.LastTimestamp, item.FirstTimestamp)
		items = append(items, map[string]any{
			"type":      item.Type,
			"reason":    item.Reason,
			"message":   item.Message,
			"component": item.Source.Component,
			"count":     item.Count,
			"timestamp": timestamp,
		})
	}
	return items, nil
}

func (c *clusterClient) GetPodLogs(namespace, name string, tailLines int) (string, error) {
	query := url.Values{}
	if tailLines > 0 {
		query.Set("tailLines", fmt.Sprintf("%d", tailLines))
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log", namespace, name)
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return c.getText(path)
}

func (c *clusterClient) GetRelatedPods(kind, namespace, name string) ([]relatedPodItem, error) {
	switch kind {
	case "Pod":
		pod, err := c.getPod(namespace, name)
		if err != nil {
			return nil, err
		}
		return []relatedPodItem{pod}, nil
	case "Deployment":
		selector, err := c.getDeploymentSelector(namespace, name)
		if err != nil {
			return nil, err
		}
		return c.listPodsBySelector(namespace, selector)
	case "DaemonSet":
		selector, err := c.getDaemonSetSelector(namespace, name)
		if err != nil {
			return nil, err
		}
		return c.listPodsBySelector(namespace, selector)
	case "StatefulSet":
		selector, err := c.getStatefulSetSelector(namespace, name)
		if err != nil {
			return nil, err
		}
		return c.listPodsBySelector(namespace, selector)
	case "Job":
		return c.listPodsByJob(namespace, name)
	case "CronJob":
		return nil, fmt.Errorf("current version does not list related pods directly for workload kind: %s", kind)
	default:
		return nil, fmt.Errorf("unsupported workload kind: %s", kind)
	}
}

func (c *clusterClient) GetWorkloadRollout(kind, namespace, name string) (*rolloutDetail, error) {
	switch kind {
	case "Deployment":
		return c.getDeploymentRollout(namespace, name)
	case "StatefulSet":
		return c.getStatefulSetRollout(namespace, name)
	default:
		return nil, fmt.Errorf("rollout is unsupported for workload kind: %s", kind)
	}
}

func (c *clusterClient) GetWorkloadResources(kind, namespace, name string) ([]relatedServiceItem, []relatedIngressItem, workloadResourceRefs, error) {
	selector, refs, err := c.getWorkloadResourceContext(kind, namespace, name)
	if err != nil {
		return nil, nil, workloadResourceRefs{}, err
	}
	services, err := c.listRelatedServices(namespace, selector)
	if err != nil {
		return nil, nil, workloadResourceRefs{}, err
	}
	ingresses, err := c.listRelatedIngresses(namespace, services)
	if err != nil {
		return nil, nil, workloadResourceRefs{}, err
	}
	return services, ingresses, refs, nil
}

func (c *clusterClient) GetWorkloadPersistentVolumeClaims(kind, namespace, name string, refs workloadResourceRefs) ([]relatedPersistentVolumeClaimItem, error) {
	var payload struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
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
		} `json:"items"`
	}
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/persistentvolumeclaims", namespace), &payload); err != nil {
		return nil, err
	}

	selected := make(map[string]string)
	for _, claimName := range refs.PersistentPVCs {
		if strings.TrimSpace(claimName) != "" {
			selected[claimName] = ""
		}
	}
	if kind == "StatefulSet" {
		for _, tmpl := range refs.ClaimTemplates {
			prefix := fmt.Sprintf("%s-%s-", tmpl.Name, name)
			for _, item := range payload.Items {
				if strings.HasPrefix(item.Metadata.Name, prefix) {
					selected[item.Metadata.Name] = tmpl.Name
				}
			}
		}
	}
	if len(selected) == 0 {
		return []relatedPersistentVolumeClaimItem{}, nil
	}

	items := make([]relatedPersistentVolumeClaimItem, 0, len(selected))
	for _, item := range payload.Items {
		claimTemplate, ok := selected[item.Metadata.Name]
		if !ok {
			continue
		}
		mountedBy, err := c.findWorkloadsByPersistentVolumeClaim(namespace, item.Metadata.Name)
		if err != nil {
			return nil, err
		}
		items = append(items, relatedPersistentVolumeClaimItem{
			Name:             item.Metadata.Name,
			Status:           firstNonEmpty(item.Status.Phase, "Pending"),
			StorageClassName: item.Spec.StorageClassName,
			VolumeName:       item.Spec.VolumeName,
			RequestedStorage: item.Spec.Resources.Requests["storage"],
			Capacity:         item.Status.Capacity["storage"],
			AccessModes:      uniqueStrings(item.Spec.AccessModes),
			ClaimTemplate:    claimTemplate,
			MountedBy:        mountedBy,
		})
	}
	slices.SortFunc(items, func(a, b relatedPersistentVolumeClaimItem) int {
		return strings.Compare(a.Name, b.Name)
	})
	return items, nil
}

func (c *clusterClient) ScaleWorkload(kind, namespace, name string, replicas int) (*scaleResult, error) {
	path, err := workloadResourcePath(kind, namespace, name)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]any{
		"spec": map[string]any{
			"replicas": replicas,
		},
	})
	if err != nil {
		return nil, err
	}
	if err := c.patchJSON(path, "application/merge-patch+json", body, nil); err != nil {
		return nil, err
	}
	return &scaleResult{Replicas: replicas}, nil
}

func (c *clusterClient) UpdateWorkloadPrimaryImage(kind, namespace, name, image string) error {
	object, err := c.GetWorkloadObject(kind, namespace, name)
	if err != nil {
		return err
	}
	containerName, err := primaryContainerName(kind, object)
	if err != nil {
		return err
	}
	path, err := workloadResourcePath(kind, namespace, name)
	if err != nil {
		return err
	}
	body, err := json.Marshal(workloadPrimaryImagePatch(kind, containerName, image))
	if err != nil {
		return err
	}
	return c.patchJSON(path, "application/strategic-merge-patch+json", body, nil)
}

func (c *clusterClient) GetWorkloadContainerImages(kind, namespace, name string) ([]containerImageItem, error) {
	object, err := c.GetWorkloadObject(kind, namespace, name)
	if err != nil {
		return nil, err
	}
	return workloadContainerImages(kind, object)
}

func (c *clusterClient) UpdateWorkloadContainerImages(kind, namespace, name string, containers []containerImageItem) error {
	path, err := workloadResourcePath(kind, namespace, name)
	if err != nil {
		return err
	}
	body, err := json.Marshal(workloadContainerImagesPatch(kind, containers))
	if err != nil {
		return err
	}
	return c.patchJSON(path, "application/strategic-merge-patch+json", body, nil)
}

func (c *clusterClient) GetStatefulSetSettings(namespace, name string) (*statefulSetSettingsItem, error) {
	object, err := c.GetWorkloadObject("StatefulSet", namespace, name)
	if err != nil {
		return nil, err
	}
	spec := extractMap(object, "spec")
	if spec == nil {
		return nil, errors.New("statefulset spec not found")
	}
	serviceName := stringValue(spec["serviceName"])
	serviceMode := "ClusterIP"
	updateStrategy := extractMap(spec, "updateStrategy")
	if strings.TrimSpace(serviceName) != "" {
		serviceObject, err := c.getServiceObject(namespace, serviceName)
		if err != nil && !strings.Contains(err.Error(), "404") {
			return nil, err
		}
		if serviceObject != nil {
			serviceSpec := extractMap(serviceObject, "spec")
			if firstNonEmpty(stringValue(serviceSpec["clusterIP"])) == "None" {
				serviceMode = "Headless"
			} else {
				serviceMode = firstNonEmpty(stringValue(serviceSpec["type"]), "ClusterIP")
			}
		}
	}
	rollingUpdate := extractMap(spec, "updateStrategy", "rollingUpdate")
	return &statefulSetSettingsItem{
		ServiceName:         serviceName,
		ServiceMode:         serviceMode,
		PodManagementPolicy: firstNonEmpty(stringValue(spec["podManagementPolicy"]), "OrderedReady"),
		RollingPartition:    intValue(rollingUpdate["partition"]),
		UpdateStrategy:      firstNonEmpty(stringValue(updateStrategy["type"]), "RollingUpdate"),
	}, nil
}

func (c *clusterClient) UpdateStatefulSetSettings(namespace, name, serviceMode, podManagementPolicy string, rollingPartition int) error {
	settings, err := c.GetStatefulSetSettings(namespace, name)
	if err != nil {
		return err
	}
	if strings.TrimSpace(podManagementPolicy) != "" && podManagementPolicy != settings.PodManagementPolicy {
		return errors.New("changing podManagementPolicy requires recreating the StatefulSet; current version keeps it read-only")
	}
	patch := map[string]any{
		"spec": map[string]any{
			"updateStrategy": map[string]any{
				"type": "RollingUpdate",
				"rollingUpdate": map[string]any{
					"partition": maxInt(rollingPartition, 0),
				},
			},
		},
	}
	path, err := workloadResourcePath("StatefulSet", namespace, name)
	if err != nil {
		return err
	}
	body, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	if err := c.patchJSON(path, "application/strategic-merge-patch+json", body, nil); err != nil {
		return err
	}
	targetMode := firstNonEmpty(serviceMode, settings.ServiceMode)
	if settings.ServiceName != "" && targetMode != settings.ServiceMode {
		if err := c.replaceStatefulSetServiceMode(namespace, settings.ServiceName, targetMode); err != nil {
			return err
		}
	}
	return nil
}

func (c *clusterClient) RestartWorkload(kind, namespace, name string) (*restartResult, error) {
	pods, err := c.GetRelatedPods(kind, namespace, name)
	if err != nil {
		return nil, err
	}
	if len(pods) == 0 {
		return nil, errors.New("no related pods found for current workload")
	}
	deleted := make([]string, 0, len(pods))
	for _, item := range pods {
		if err := c.delete(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", item.Namespace, item.Name)); err != nil {
			return nil, err
		}
		deleted = append(deleted, item.Name)
	}
	return &restartResult{DeletedPods: deleted}, nil
}

func (c *clusterClient) RolloutRestartWorkload(kind, namespace, name string) (*rolloutRestartResult, error) {
	path, err := workloadResourcePath(kind, namespace, name)
	if err != nil {
		return nil, err
	}
	restartedAt := time.Now().UTC().Format(time.RFC3339)
	body, err := json.Marshal(map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]any{
						"kubectl.kubernetes.io/restartedAt": restartedAt,
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if err := c.patchJSON(path, "application/strategic-merge-patch+json", body, nil); err != nil {
		return nil, err
	}
	return &rolloutRestartResult{RestartedAt: restartedAt}, nil
}

func (c *clusterClient) DeleteWorkload(kind, namespace, name string) error {
	path, err := workloadResourcePath(kind, namespace, name)
	if err != nil {
		return err
	}
	return c.delete(path)
}

func (c *clusterClient) listDeployments() ([]workloadRemoteItem, error) {
	var payload deploymentsPayload
	if err := c.getJSON("/apis/apps/v1/deployments", &payload); err != nil {
		return nil, err
	}
	items := make([]workloadRemoteItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, workloadRemoteItem{
			NamespaceName: item.Metadata.Namespace,
			Kind:          "Deployment",
			Name:          item.Metadata.Name,
			ReadyReplicas: item.Status.ReadyReplicas,
			Replicas:      item.Spec.Replicas,
			Image:         firstDeploymentImage(item.Spec.Template.Spec.Containers),
			Status:        workloadStatus(item.Status.ReadyReplicas, item.Spec.Replicas),
		})
	}
	return items, nil
}

func (c *clusterClient) listStatefulSets() ([]workloadRemoteItem, error) {
	var payload statefulSetsPayload
	if err := c.getJSON("/apis/apps/v1/statefulsets", &payload); err != nil {
		return nil, err
	}
	items := make([]workloadRemoteItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, workloadRemoteItem{
			NamespaceName: item.Metadata.Namespace,
			Kind:          "StatefulSet",
			Name:          item.Metadata.Name,
			ReadyReplicas: item.Status.ReadyReplicas,
			Replicas:      item.Spec.Replicas,
			Image:         firstStatefulSetImage(item.Spec.Template.Spec.Containers),
			Status:        workloadStatus(item.Status.ReadyReplicas, item.Spec.Replicas),
		})
	}
	return items, nil
}

func (c *clusterClient) listDaemonSets() ([]workloadRemoteItem, error) {
	var payload daemonSetsPayload
	if err := c.getJSON("/apis/apps/v1/daemonsets", &payload); err != nil {
		return nil, err
	}
	items := make([]workloadRemoteItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, workloadRemoteItem{
			NamespaceName: item.Metadata.Namespace,
			Kind:          "DaemonSet",
			Name:          item.Metadata.Name,
			ReadyReplicas: item.Status.NumberReady,
			Replicas:      item.Status.DesiredNumberScheduled,
			Image:         firstDaemonSetImage(item.Spec.Template.Spec.Containers),
			Status:        workloadStatus(item.Status.NumberReady, item.Status.DesiredNumberScheduled),
		})
	}
	return items, nil
}

func (c *clusterClient) listJobs() ([]workloadRemoteItem, error) {
	var payload jobsPayload
	if err := c.getJSON("/apis/batch/v1/jobs", &payload); err != nil {
		return nil, err
	}
	items := make([]workloadRemoteItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		desired := item.Spec.Completions
		if desired <= 0 {
			desired = item.Spec.Parallelism
		}
		if desired <= 0 {
			desired = 1
		}
		items = append(items, workloadRemoteItem{
			NamespaceName: item.Metadata.Namespace,
			Kind:          "Job",
			Name:          item.Metadata.Name,
			ReadyReplicas: item.Status.Succeeded,
			Replicas:      desired,
			Image:         firstJobImage(item.Spec.Template.Spec.Containers),
			Status:        jobStatus(item.Status.Active, item.Status.Succeeded, item.Status.Failed, desired),
		})
	}
	return items, nil
}

func (c *clusterClient) listCronJobs() ([]workloadRemoteItem, error) {
	var payload cronJobsPayload
	if err := c.getJSON("/apis/batch/v1/cronjobs", &payload); err != nil {
		return nil, err
	}
	items := make([]workloadRemoteItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		active := len(item.Status.Active)
		replicas := 1
		if active > replicas {
			replicas = active
		}
		items = append(items, workloadRemoteItem{
			NamespaceName: item.Metadata.Namespace,
			Kind:          "CronJob",
			Name:          item.Metadata.Name,
			ReadyReplicas: active,
			Replicas:      replicas,
			Image:         firstCronJobImage(item.Spec.JobTemplate.Spec.Template.Spec.Containers),
			Status:        cronJobStatus(item.Spec.Suspend, active),
		})
	}
	return items, nil
}

func (c *clusterClient) listPods() ([]workloadRemoteItem, error) {
	var payload podsPayload
	if err := c.getJSON("/api/v1/pods", &payload); err != nil {
		return nil, err
	}
	items := make([]workloadRemoteItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, workloadRemoteItem{
			NamespaceName: item.Metadata.Namespace,
			Kind:          "Pod",
			Name:          item.Metadata.Name,
			ReadyReplicas: 1,
			Replicas:      1,
			Image:         firstPodImage(item.Spec.Containers),
			Status:        item.Status.Phase,
		})
	}
	return items, nil
}

func (c *clusterClient) getDeploymentSelector(namespace, name string) (map[string]string, error) {
	var payload deploymentDetailPayload
	if err := c.getJSON(fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	return payload.Spec.Selector.MatchLabels, nil
}

func (c *clusterClient) getDaemonSetSelector(namespace, name string) (map[string]string, error) {
	var payload daemonSetDetailPayload
	if err := c.getJSON(fmt.Sprintf("/apis/apps/v1/namespaces/%s/daemonsets/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	return payload.Spec.Selector.MatchLabels, nil
}

func (c *clusterClient) getStatefulSetSelector(namespace, name string) (map[string]string, error) {
	var payload statefulSetDetailPayload
	if err := c.getJSON(fmt.Sprintf("/apis/apps/v1/namespaces/%s/statefulsets/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	return payload.Spec.Selector.MatchLabels, nil
}

func (c *clusterClient) getPod(namespace, name string) (relatedPodItem, error) {
	var payload podDetailPayload
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", namespace, name), &payload); err != nil {
		return relatedPodItem{}, err
	}
	return relatedPodItem{
		Name:      payload.Metadata.Name,
		Namespace: payload.Metadata.Namespace,
		Status:    payload.Status.Phase,
		Image:     firstPodImage(payload.Spec.Containers),
	}, nil
}

func (c *clusterClient) listPodsBySelector(namespace string, selector map[string]string) ([]relatedPodItem, error) {
	query := url.Values{}
	if encoded := labelSelector(selector); encoded != "" {
		query.Set("labelSelector", encoded)
	}
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods", namespace)
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var payload podsPayload
	if err := c.getJSON(path, &payload); err != nil {
		return nil, err
	}
	items := make([]relatedPodItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, relatedPodItem{
			Name:      item.Metadata.Name,
			Namespace: item.Metadata.Namespace,
			Status:    item.Status.Phase,
			Image:     firstPodImage(item.Spec.Containers),
		})
	}
	return items, nil
}

func (c *clusterClient) listPodsByJob(namespace, name string) ([]relatedPodItem, error) {
	query := url.Values{}
	query.Set("labelSelector", fmt.Sprintf("job-name=%s", name))
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods?%s", namespace, query.Encode())
	var payload podsPayload
	if err := c.getJSON(path, &payload); err != nil {
		return nil, err
	}
	items := make([]relatedPodItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, relatedPodItem{
			Name:      item.Metadata.Name,
			Namespace: item.Metadata.Namespace,
			Status:    item.Status.Phase,
			Image:     firstPodImage(item.Spec.Containers),
		})
	}
	return items, nil
}

func (c *clusterClient) getDeploymentRollout(namespace, name string) (*rolloutDetail, error) {
	var payload deploymentRolloutPayload
	if err := c.getJSON(fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	replicaSets, err := c.listReplicaSets(namespace, payload.Spec.Selector.MatchLabels)
	if err != nil {
		return nil, err
	}
	conditions := make([]map[string]any, 0, len(payload.Status.Conditions))
	for _, item := range payload.Status.Conditions {
		conditions = append(conditions, map[string]any{
			"type":    item.Type,
			"status":  item.Status,
			"reason":  item.Reason,
			"message": item.Message,
		})
	}
	return &rolloutDetail{
		Strategy:            firstNonEmpty(payload.Spec.Strategy.Type, "RollingUpdate"),
		MaxSurge:            stringifyAny(payload.Spec.Strategy.RollingUpdate.MaxSurge),
		MaxUnavailable:      stringifyAny(payload.Spec.Strategy.RollingUpdate.MaxUnavailable),
		ObservedGeneration:  payload.Status.ObservedGeneration,
		Generation:          payload.Metadata.Generation,
		ReadyReplicas:       payload.Status.ReadyReplicas,
		UpdatedReplicas:     payload.Status.UpdatedReplicas,
		AvailableReplicas:   payload.Status.AvailableReplicas,
		UnavailableReplicas: payload.Status.UnavailableReplicas,
		DesiredReplicas:     payload.Spec.Replicas,
		Conditions:          conditions,
		ReplicaSets:         replicaSets,
	}, nil
}

func (c *clusterClient) getStatefulSetRollout(namespace, name string) (*rolloutDetail, error) {
	var payload statefulSetDetailPayload
	if err := c.getJSON(fmt.Sprintf("/apis/apps/v1/namespaces/%s/statefulsets/%s", namespace, name), &payload); err != nil {
		return nil, err
	}
	return &rolloutDetail{
		Strategy:           firstNonEmpty(payload.Spec.UpdateStrategy.Type, "RollingUpdate"),
		ObservedGeneration: payload.Status.ObservedGeneration,
		Generation:         payload.Metadata.Generation,
		ReadyReplicas:      payload.Status.ReadyReplicas,
		UpdatedReplicas:    payload.Status.UpdatedReplicas,
		CurrentReplicas:    payload.Status.CurrentReplicas,
		DesiredReplicas:    payload.Spec.Replicas,
		CurrentRevision:    payload.Status.CurrentRevision,
		UpdateRevision:     payload.Status.UpdateRevision,
	}, nil
}

func (c *clusterClient) listReplicaSets(namespace string, selector map[string]string) ([]rolloutReplicaSetItem, error) {
	query := url.Values{}
	if encoded := labelSelector(selector); encoded != "" {
		query.Set("labelSelector", encoded)
	}
	path := fmt.Sprintf("/apis/apps/v1/namespaces/%s/replicasets", namespace)
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var payload replicaSetsPayload
	if err := c.getJSON(path, &payload); err != nil {
		return nil, err
	}
	items := make([]rolloutReplicaSetItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, rolloutReplicaSetItem{
			Name:              item.Metadata.Name,
			Replicas:          item.Spec.Replicas,
			ReadyReplicas:     item.Status.ReadyReplicas,
			AvailableReplicas: item.Status.AvailableReplicas,
			Revision:          item.Metadata.Annotations["deployment.kubernetes.io/revision"],
			CreationTimestamp: item.Metadata.CreationTimestamp,
		})
	}
	return items, nil
}

func (c *clusterClient) getWorkloadResourceContext(kind, namespace, name string) (map[string]string, workloadResourceRefs, error) {
	object, err := c.GetWorkloadObject(kind, namespace, name)
	if err != nil {
		return nil, workloadResourceRefs{}, err
	}
	spec, _ := object["spec"].(map[string]any)
	if spec == nil {
		return nil, workloadResourceRefs{}, errors.New("workload spec not found")
	}
	selector := extractStringMap(spec, "selector", "matchLabels")
	var podSpec map[string]any
	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet", "Job":
		podSpec = extractMap(spec, "template", "spec")
	case "CronJob":
		podSpec = extractMap(spec, "jobTemplate", "spec", "template", "spec")
	case "Pod":
		podSpec = spec
	default:
		return nil, workloadResourceRefs{}, fmt.Errorf("resources are unsupported for workload kind: %s", kind)
	}
	refs := collectWorkloadRefs(podSpec)
	if kind == "StatefulSet" {
		refs.ClaimTemplates = extractClaimTemplates(spec)
	}
	return selector, refs, nil
}

func (c *clusterClient) listRelatedServices(namespace string, selector map[string]string) ([]relatedServiceItem, error) {
	var payload servicesPayload
	if err := c.getJSON(fmt.Sprintf("/api/v1/namespaces/%s/services", namespace), &payload); err != nil {
		return nil, err
	}
	items := make([]relatedServiceItem, 0)
	for _, item := range payload.Items {
		if !selectorMatches(item.Spec.Selector, selector) {
			continue
		}
		ports := make([]string, 0, len(item.Spec.Ports))
		for _, port := range item.Spec.Ports {
			target := stringifyAny(port.TargetPort)
			if target == "-" || target == "" {
				target = fmt.Sprintf("%d", port.Port)
			}
			ports = append(ports, fmt.Sprintf("%d -> %s", port.Port, target))
		}
		endpoints, err := c.getServiceEndpoints(namespace, item.Metadata.Name)
		if err != nil {
			return nil, err
		}
		items = append(items, relatedServiceItem{
			Name:      item.Metadata.Name,
			Type:      firstNonEmpty(item.Spec.Type, "ClusterIP"),
			ClusterIP: firstNonEmpty(item.Spec.ClusterIP, "-"),
			Selector:  selectorPairs(item.Spec.Selector),
			Ports:     ports,
			Endpoints: endpoints,
		})
	}
	return items, nil
}

func (c *clusterClient) listRelatedIngresses(namespace string, services []relatedServiceItem) ([]relatedIngressItem, error) {
	var payload ingressesPayload
	if err := c.getJSON(fmt.Sprintf("/apis/networking.k8s.io/v1/namespaces/%s/ingresses", namespace), &payload); err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	serviceNames := make(map[string]struct{}, len(services))
	for _, item := range services {
		serviceNames[item.Name] = struct{}{}
	}
	items := make([]relatedIngressItem, 0)
	for _, item := range payload.Items {
		hosts := make([]string, 0)
		paths := make([]string, 0)
		backends := make([]string, 0)
		matched := false
		for _, rule := range item.Spec.Rules {
			if strings.TrimSpace(rule.Host) != "" {
				hosts = append(hosts, rule.Host)
			}
			for _, path := range rule.HTTP.Paths {
				serviceName := path.Backend.Service.Name
				if serviceName == "" {
					continue
				}
				backends = append(backends, serviceName)
				if _, ok := serviceNames[serviceName]; ok {
					matched = true
				}
				if strings.TrimSpace(path.Path) != "" {
					paths = append(paths, path.Path)
				}
			}
		}
		if !matched {
			continue
		}
		items = append(items, relatedIngressItem{
			Name:      item.Metadata.Name,
			Hosts:     uniqueStrings(hosts),
			Paths:     uniqueStrings(paths),
			Backends:  uniqueStrings(backends),
			Addresses: loadBalancerAddresses(item.Status.LoadBalancer.Ingress),
		})
	}
	return items, nil
}
