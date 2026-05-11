package k8s

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

func buildWorkloadBundle(input CreateWorkloadBundleInput) ([]map[string]any, []string, error) {
	namespace := strings.TrimSpace(input.Namespace)
	name := strings.TrimSpace(input.Name)
	workloadType := coalesceString(strings.TrimSpace(input.WorkloadType), "Deployment")
	replicas := maxInt(input.Replicas, 1)
	serviceType := coalesceString(strings.TrimSpace(input.ServiceType), "ClusterIP")
	servicePort := input.ServicePort
	if input.CreateIngress && !input.CreateService {
		return nil, nil, errors.New("ingress requires service to be enabled")
	}
	if workloadType == "StatefulSet" && !input.CreateService {
		return nil, nil, errors.New("statefulset requires service to be enabled")
	}
	objects := make([]map[string]any, 0, 12)
	kinds := make([]string, 0, 12)

	labels := map[string]string{"app": name}
	if extraLabels, err := parseOptionalStringMap(input.LabelsText); err != nil {
		return nil, nil, err
	} else {
		for key, value := range extraLabels {
			labels[key] = value
		}
	}
	annotations, err := parseOptionalStringMap(input.AnnotationsText)
	if err != nil {
		return nil, nil, err
	}

	serviceAccountName := strings.TrimSpace(input.ServiceAccountName)
	if input.CreateServiceAccount {
		if serviceAccountName == "" {
			serviceAccountName = name
		}
		objects = append(objects, map[string]any{
			"apiVersion": "v1",
			"kind":       "ServiceAccount",
			"metadata": map[string]any{
				"name":      serviceAccountName,
				"namespace": namespace,
			},
		})
		kinds = append(kinds, "ServiceAccount")
	}

	volumeInputs := append([]WorkloadVolumeInput(nil), input.Volumes...)
	if len(volumeInputs) == 0 {
		if input.CreateConfigMap {
			volumeInputs = append(volumeInputs, WorkloadVolumeInput{
				Name:          "config-volume",
				Type:          "configMap",
				ConfigMapName: coalesceString(strings.TrimSpace(input.ConfigMapName), name+"-config"),
				ConfigMapData: input.ConfigMapData,
			})
		}
		if input.CreatePVC {
			volumeInputs = append(volumeInputs, WorkloadVolumeInput{
				Name:                "data-volume",
				Type:                "pvc",
				PVCName:             coalesceString(strings.TrimSpace(input.PVCName), name+"-data"),
				PVCStorageClassName: strings.TrimSpace(input.PVCStorageClassName),
				PVCSize:             coalesceString(strings.TrimSpace(input.PVCSize), "1Gi"),
			})
		}
	}

	volumes, extraObjects, extraKinds, err := buildVolumes(namespace, volumeInputs)
	if err != nil {
		return nil, nil, err
	}
	objects = append(objects, extraObjects...)
	kinds = append(kinds, extraKinds...)

	containerInputs := append([]WorkloadContainerInput(nil), input.Containers...)
	if len(containerInputs) == 0 {
		containerInputs = []WorkloadContainerInput{{
			Name:      name,
			Image:     input.Image,
			PortsText: fmt.Sprintf("%d", maxInt(input.ContainerPort, 80)),
		}}
	}
	containers, primaryPort, err := buildContainers(containerInputs, volumeInputs, input.ClaimTemplates, name, input.Image, input.ContainerPort)
	if err != nil {
		return nil, nil, err
	}
	initContainers, _, err := buildContainers(input.InitContainers, volumeInputs, input.ClaimTemplates, "init", "", 0)
	if err != nil {
		return nil, nil, err
	}
	if servicePort <= 0 {
		servicePort = primaryPort
	}
	if servicePort <= 0 {
		servicePort = 80
	}

	podSpec := map[string]any{
		"containers": containers,
	}
	if len(initContainers) > 0 {
		podSpec["initContainers"] = initContainers
	}
	if serviceAccountName != "" {
		podSpec["serviceAccountName"] = serviceAccountName
	}
	if len(volumes) > 0 {
		podSpec["volumes"] = volumes
	}
	if nodeSelector, err := parseOptionalStringMap(input.NodeSelectorText); err != nil {
		return nil, nil, err
	} else if len(nodeSelector) > 0 {
		podSpec["nodeSelector"] = nodeSelector
	}
	if tolerations, err := parseOptionalYAMLList(input.TolerationsYAML); err != nil {
		return nil, nil, err
	} else if len(tolerations) > 0 {
		podSpec["tolerations"] = tolerations
	}
	if affinity, err := parseOptionalYAMLMap(input.AffinityYAML); err != nil {
		return nil, nil, err
	} else if len(affinity) > 0 {
		podSpec["affinity"] = affinity
	}
	if dnsPolicy := strings.TrimSpace(input.DNSPolicy); dnsPolicy != "" {
		podSpec["dnsPolicy"] = dnsPolicy
	}
	if dnsConfig, err := parseOptionalYAMLMap(input.DNSConfigYAML); err != nil {
		return nil, nil, err
	} else if len(dnsConfig) > 0 {
		podSpec["dnsConfig"] = dnsConfig
	}
	if hostAliases, err := parseOptionalYAMLList(input.HostAliasesYAML); err != nil {
		return nil, nil, err
	} else if len(hostAliases) > 0 {
		podSpec["hostAliases"] = hostAliases
	}
	if podSecurityContext, err := parseOptionalYAMLMap(input.PodSecurityContextYAML); err != nil {
		return nil, nil, err
	} else if len(podSecurityContext) > 0 {
		podSpec["securityContext"] = podSecurityContext
	}
	if input.HostNetwork {
		podSpec["hostNetwork"] = true
	}
	if input.HostPID {
		podSpec["hostPID"] = true
	}
	if input.HostIPC {
		podSpec["hostIPC"] = true
	}
	if imagePullSecrets := buildImagePullSecrets(input.ImagePullSecretsText); len(imagePullSecrets) > 0 {
		podSpec["imagePullSecrets"] = imagePullSecrets
	}
	if topologySpread, err := parseOptionalYAMLList(input.TopologySpreadYAML); err != nil {
		return nil, nil, err
	} else if len(topologySpread) > 0 {
		podSpec["topologySpreadConstraints"] = topologySpread
	}
	if input.TerminationGracePeriod > 0 {
		podSpec["terminationGracePeriodSeconds"] = input.TerminationGracePeriod
	}

	workload, workloadKind, err := buildWorkloadObject(workloadType, namespace, name, labels, annotations, replicas, input, podSpec)
	if err != nil {
		return nil, nil, err
	}
	objects = append(objects, workload)
	kinds = append(kinds, workloadKind)

	if input.CreateService {
		serviceName := name
		serviceSpec := map[string]any{
			"selector": labels,
			"ports": []map[string]any{{
				"name":       "http",
				"port":       servicePort,
				"targetPort": primaryPort,
				"protocol":   "TCP",
			}},
		}
		if workloadType == "StatefulSet" {
			serviceName = coalesceString(strings.TrimSpace(input.ServiceName), name)
			if strings.EqualFold(strings.TrimSpace(input.StatefulSetServiceMode), "Headless") || strings.TrimSpace(input.StatefulSetServiceMode) == "" {
				serviceSpec["clusterIP"] = "None"
				serviceSpec["publishNotReadyAddresses"] = true
			} else {
				serviceSpec["type"] = serviceType
			}
		} else {
			serviceSpec["type"] = serviceType
		}
		service := map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name":      serviceName,
				"namespace": namespace,
			},
			"spec": serviceSpec,
		}
		objects = append(objects, service)
		kinds = append(kinds, "Service")
	}

	if input.CreateIngress {
		ingressPath := coalesceString(strings.TrimSpace(input.IngressPath), "/")
		ingress := map[string]any{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "Ingress",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"rules": []map[string]any{{
					"host": strings.TrimSpace(input.IngressHost),
					"http": map[string]any{
						"paths": []map[string]any{{
							"path":     ingressPath,
							"pathType": "Prefix",
							"backend": map[string]any{
								"service": map[string]any{
									"name": name,
									"port": map[string]any{
										"number": servicePort,
									},
								},
							},
						}},
					},
				}},
			},
		}
		if strings.TrimSpace(input.IngressClassName) != "" {
			ingress["spec"].(map[string]any)["ingressClassName"] = strings.TrimSpace(input.IngressClassName)
		}
		objects = append(objects, ingress)
		kinds = append(kinds, "Ingress")
	}
	return objects, kinds, nil
}

func buildVolumes(namespace string, items []WorkloadVolumeInput) ([]map[string]any, []map[string]any, []string, error) {
	volumes := make([]map[string]any, 0, len(items))
	objects := make([]map[string]any, 0, len(items))
	kinds := make([]string, 0, len(items))
	for index, item := range items {
		name := coalesceString(strings.TrimSpace(item.Name), fmt.Sprintf("volume-%d", index+1))
		kind := strings.TrimSpace(item.Type)
		if kind == "" {
			return nil, nil, nil, fmt.Errorf("volume %s missing type", name)
		}
		volume := map[string]any{"name": name}
		switch kind {
		case "configMap":
			configMapName := coalesceString(strings.TrimSpace(item.ConfigMapName), name)
			if strings.TrimSpace(item.ConfigMapData) != "" {
				data, err := parseKeyValueBlock(item.ConfigMapData)
				if err != nil {
					return nil, nil, nil, err
				}
				objects = append(objects, map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]any{
						"name":      configMapName,
						"namespace": namespace,
					},
					"data": data,
				})
				kinds = append(kinds, "ConfigMap")
			}
			volume["configMap"] = map[string]any{"name": configMapName}
		case "pvc":
			claimName := coalesceString(strings.TrimSpace(item.PVCName), name)
			if strings.TrimSpace(item.PVCSize) != "" {
				pvc := map[string]any{
					"apiVersion": "v1",
					"kind":       "PersistentVolumeClaim",
					"metadata": map[string]any{
						"name":      claimName,
						"namespace": namespace,
					},
					"spec": map[string]any{
						"accessModes": []string{"ReadWriteOnce"},
						"resources": map[string]any{
							"requests": map[string]any{
								"storage": strings.TrimSpace(item.PVCSize),
							},
						},
					},
				}
				if strings.TrimSpace(item.PVCStorageClassName) != "" {
					pvc["spec"].(map[string]any)["storageClassName"] = strings.TrimSpace(item.PVCStorageClassName)
				}
				objects = append(objects, pvc)
				kinds = append(kinds, "PersistentVolumeClaim")
			}
			volume["persistentVolumeClaim"] = map[string]any{"claimName": claimName}
		case "emptyDir":
			emptyDir := map[string]any{}
			if strings.TrimSpace(item.EmptyDirMedium) != "" {
				emptyDir["medium"] = strings.TrimSpace(item.EmptyDirMedium)
			}
			volume["emptyDir"] = emptyDir
		case "hostPath":
			volume["hostPath"] = map[string]any{"path": strings.TrimSpace(item.HostPath)}
		case "nfs":
			volume["nfs"] = map[string]any{
				"server": strings.TrimSpace(item.NFSServer),
				"path":   strings.TrimSpace(item.NFSPath),
			}
		case "secret":
			volume["secret"] = map[string]any{"secretName": strings.TrimSpace(item.SecretName)}
		default:
			return nil, nil, nil, fmt.Errorf("unsupported volume type: %s", kind)
		}
		volumes = append(volumes, volume)
	}
	return volumes, objects, kinds, nil
}

func buildContainers(items []WorkloadContainerInput, volumeDefs []WorkloadVolumeInput, claimTemplates []WorkloadClaimTemplateInput, defaultName, defaultImage string, defaultPort int) ([]map[string]any, int, error) {
	containers := make([]map[string]any, 0, len(items))
	firstPort := 0
	volumeNames := make(map[string]struct{}, len(volumeDefs)+len(claimTemplates))
	for index, item := range volumeDefs {
		name := coalesceString(strings.TrimSpace(item.Name), fmt.Sprintf("volume-%d", index+1))
		volumeNames[name] = struct{}{}
	}
	for index, item := range claimTemplates {
		name := coalesceString(strings.TrimSpace(item.Name), fmt.Sprintf("data-%d", index+1))
		volumeNames[name] = struct{}{}
	}
	for index, item := range items {
		name := coalesceString(strings.TrimSpace(item.Name), fmt.Sprintf("%s-%d", defaultName, index+1))
		image := coalesceString(strings.TrimSpace(item.Image), defaultImage)
		if image == "" {
			return nil, 0, fmt.Errorf("container %s missing image", name)
		}
		container := map[string]any{"name": name, "image": image}
		if imagePullPolicy := strings.TrimSpace(item.ImagePullPolicy); imagePullPolicy != "" {
			container["imagePullPolicy"] = imagePullPolicy
		}
		ports, first, err := parseContainerPorts(item.PortsText, defaultPort)
		if err != nil {
			return nil, 0, err
		}
		if len(ports) > 0 {
			container["ports"] = ports
		}
		if firstPort == 0 {
			firstPort = first
		}
		if commands := parseMultilineValues(item.CommandText); len(commands) > 0 {
			container["command"] = commands
		}
		if args := parseMultilineValues(item.ArgsText); len(args) > 0 {
			container["args"] = args
		}
		if workingDir := strings.TrimSpace(item.WorkingDir); workingDir != "" {
			container["workingDir"] = workingDir
		}
		if item.Stdin {
			container["stdin"] = true
		}
		if item.TTY {
			container["tty"] = true
		}
		if env, err := buildEnvVars(item.EnvText); err != nil {
			return nil, 0, err
		} else if len(env) > 0 {
			container["env"] = env
		}
		if envFrom, err := buildEnvFromSources(item.EnvFromText); err != nil {
			return nil, 0, err
		} else if len(envFrom) > 0 {
			container["envFrom"] = envFrom
		}
		if envValueFrom, err := buildEnvValueFromVars(item.EnvValueFromText); err != nil {
			return nil, 0, err
		} else if len(envValueFrom) > 0 {
			existing, _ := container["env"].([]map[string]any)
			container["env"] = append(existing, envValueFrom...)
		}
		if resources := buildContainerResources(item.CPURequest, item.MemoryRequest, item.CPULimit, item.MemoryLimit); len(resources) > 0 {
			container["resources"] = resources
		}
		if livenessProbe, err := parseOptionalYAMLMap(item.LivenessProbeYAML); err != nil {
			return nil, 0, err
		} else if len(livenessProbe) > 0 {
			container["livenessProbe"] = livenessProbe
		}
		if readinessProbe, err := parseOptionalYAMLMap(item.ReadinessProbeYAML); err != nil {
			return nil, 0, err
		} else if len(readinessProbe) > 0 {
			container["readinessProbe"] = readinessProbe
		}
		if startupProbe, err := parseOptionalYAMLMap(item.StartupProbeYAML); err != nil {
			return nil, 0, err
		} else if len(startupProbe) > 0 {
			container["startupProbe"] = startupProbe
		}
		if lifecycle, err := buildLifecycle(item.LifecyclePostStart, item.LifecyclePreStop); err != nil {
			return nil, 0, err
		} else if len(lifecycle) > 0 {
			container["lifecycle"] = lifecycle
		}
		if securityContext, err := parseOptionalYAMLMap(item.SecurityContextYAML); err != nil {
			return nil, 0, err
		} else if len(securityContext) > 0 {
			container["securityContext"] = securityContext
		}
		if mounts, err := buildContainerVolumeMounts(item.VolumeMountsText, volumeNames); err != nil {
			return nil, 0, err
		} else if len(mounts) > 0 {
			container["volumeMounts"] = mounts
		}
		containers = append(containers, container)
	}
	return containers, firstPort, nil
}

func buildContainerVolumeMounts(raw string, volumeNames map[string]struct{}) ([]map[string]any, error) {
	lines := parseMultilineValues(raw)
	mounts := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return nil, fmt.Errorf("invalid volume mount format %q, expected volume:/mount/path[:ro|rw]", line)
		}
		volumeName := strings.TrimSpace(parts[0])
		mountPath := strings.TrimSpace(parts[1])
		if volumeName == "" || mountPath == "" {
			return nil, fmt.Errorf("invalid volume mount format %q, volume name and mount path are required", line)
		}
		if _, ok := volumeNames[volumeName]; !ok {
			return nil, fmt.Errorf("volume mount references undefined volume %q", volumeName)
		}
		mount := map[string]any{"name": volumeName, "mountPath": mountPath}
		if len(parts) == 3 {
			mode := strings.ToLower(strings.TrimSpace(parts[2]))
			switch mode {
			case "", "rw":
			case "ro":
				mount["readOnly"] = true
			default:
				return nil, fmt.Errorf("invalid mount mode %q for volume %s, expected ro or rw", mode, volumeName)
			}
		}
		mounts = append(mounts, mount)
	}
	return mounts, nil
}

func buildEnvFromSources(raw string) ([]map[string]any, error) {
	lines := parseMultilineValues(raw)
	items := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid envFrom format %q, expected configMap:name or secret:name", line)
		}
		sourceType := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		if name == "" {
			return nil, fmt.Errorf("invalid envFrom format %q, source name is required", line)
		}
		switch sourceType {
		case "configMap":
			items = append(items, map[string]any{"configMapRef": map[string]any{"name": name}})
		case "secret":
			items = append(items, map[string]any{"secretRef": map[string]any{"name": name}})
		default:
			return nil, fmt.Errorf("unsupported envFrom source %q", sourceType)
		}
	}
	return items, nil
}

func buildEnvValueFromVars(raw string) ([]map[string]any, error) {
	lines := parseMultilineValues(raw)
	items := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid env valueFrom format %q, expected ENV=configMap:name:key or ENV=secret:name:key", line)
		}
		name := strings.TrimSpace(parts[0])
		if name == "" {
			return nil, fmt.Errorf("invalid env valueFrom format %q, env name is required", line)
		}
		refParts := strings.Split(parts[1], ":")
		if len(refParts) != 3 {
			return nil, fmt.Errorf("invalid env valueFrom format %q, expected ENV=configMap:name:key or ENV=secret:name:key", line)
		}
		sourceType := strings.TrimSpace(refParts[0])
		sourceName := strings.TrimSpace(refParts[1])
		sourceKey := strings.TrimSpace(refParts[2])
		if sourceName == "" || sourceKey == "" {
			return nil, fmt.Errorf("invalid env valueFrom format %q, source name and key are required", line)
		}
		valueFrom := map[string]any{}
		switch sourceType {
		case "configMap":
			valueFrom["configMapKeyRef"] = map[string]any{"name": sourceName, "key": sourceKey}
		case "secret":
			valueFrom["secretKeyRef"] = map[string]any{"name": sourceName, "key": sourceKey}
		default:
			return nil, fmt.Errorf("unsupported env valueFrom source %q", sourceType)
		}
		items = append(items, map[string]any{"name": name, "valueFrom": valueFrom})
	}
	return items, nil
}

func buildContainerResources(cpuRequest, memoryRequest, cpuLimit, memoryLimit string) map[string]any {
	requests := map[string]any{}
	limits := map[string]any{}
	if strings.TrimSpace(cpuRequest) != "" {
		requests["cpu"] = strings.TrimSpace(cpuRequest)
	}
	if strings.TrimSpace(memoryRequest) != "" {
		requests["memory"] = strings.TrimSpace(memoryRequest)
	}
	if strings.TrimSpace(cpuLimit) != "" {
		limits["cpu"] = strings.TrimSpace(cpuLimit)
	}
	if strings.TrimSpace(memoryLimit) != "" {
		limits["memory"] = strings.TrimSpace(memoryLimit)
	}
	resources := map[string]any{}
	if len(requests) > 0 {
		resources["requests"] = requests
	}
	if len(limits) > 0 {
		resources["limits"] = limits
	}
	return resources
}

func buildLifecycle(postStart, preStop string) (map[string]any, error) {
	lifecycle := map[string]any{}
	if handler, err := buildLifecycleHandler(postStart); err != nil {
		return nil, err
	} else if len(handler) > 0 {
		lifecycle["postStart"] = handler
	}
	if handler, err := buildLifecycleHandler(preStop); err != nil {
		return nil, err
	} else if len(handler) > 0 {
		lifecycle["preStop"] = handler
	}
	return lifecycle, nil
}

func buildLifecycleHandler(raw string) (map[string]any, error) {
	commands := parseMultilineValues(raw)
	if len(commands) == 0 {
		return map[string]any{}, nil
	}
	return map[string]any{"exec": map[string]any{"command": commands}}, nil
}

func buildImagePullSecrets(raw string) []map[string]any {
	items := parseMultilineValues(raw)
	refs := make([]map[string]any, 0, len(items))
	for _, item := range items {
		refs = append(refs, map[string]any{"name": item})
	}
	return refs
}

func buildVolumeClaimTemplates(items []WorkloadClaimTemplateInput) ([]map[string]any, error) {
	templates := make([]map[string]any, 0, len(items))
	for index, item := range items {
		name := coalesceString(strings.TrimSpace(item.Name), fmt.Sprintf("data-%d", index+1))
		size := coalesceString(strings.TrimSpace(item.Size), "1Gi")
		accessModes := parseMultilineValues(item.AccessModesText)
		if len(accessModes) == 0 {
			accessModes = []string{"ReadWriteOnce"}
		}
		labels, err := parseOptionalStringMap(item.LabelsText)
		if err != nil {
			return nil, err
		}
		annotations, err := parseOptionalStringMap(item.AnnotationsText)
		if err != nil {
			return nil, err
		}
		metadata := map[string]any{"name": name}
		if len(labels) > 0 {
			metadata["labels"] = stringMapToAny(labels)
		}
		if len(annotations) > 0 {
			metadata["annotations"] = stringMapToAny(annotations)
		}
		spec := map[string]any{
			"accessModes": accessModes,
			"resources": map[string]any{
				"requests": map[string]any{"storage": size},
			},
		}
		if storageClassName := strings.TrimSpace(item.StorageClassName); storageClassName != "" {
			spec["storageClassName"] = storageClassName
		}
		templates = append(templates, map[string]any{"metadata": metadata, "spec": spec})
	}
	return templates, nil
}

func buildWorkloadObject(workloadType, namespace, name string, labels, annotations map[string]string, replicas int, input CreateWorkloadBundleInput, podSpec map[string]any) (map[string]any, string, error) {
	metadata := map[string]any{"name": name, "namespace": namespace}
	if len(annotations) > 0 {
		metadata["annotations"] = annotations
	}
	labelMap := stringMapToAny(labels)
	template := map[string]any{
		"metadata": map[string]any{"labels": labelMap},
		"spec":     podSpec,
	}
	switch workloadType {
	case "Deployment":
		spec := map[string]any{"replicas": replicas, "selector": map[string]any{"matchLabels": labelMap}, "template": template}
		if strategy := buildUpdateStrategy(input.UpdateStrategy, input.MaxSurge, input.MaxUnavailable); len(strategy) > 0 {
			spec["strategy"] = strategy
		}
		return map[string]any{"apiVersion": "apps/v1", "kind": "Deployment", "metadata": metadata, "spec": spec}, "Deployment", nil
	case "StatefulSet":
		serviceName := coalesceString(strings.TrimSpace(input.ServiceName), name)
		spec := map[string]any{
			"serviceName": serviceName,
			"replicas":    replicas,
			"selector":    map[string]any{"matchLabels": labelMap},
			"template":    template,
		}
		if strategyType := strings.TrimSpace(input.UpdateStrategy); strategyType != "" {
			updateStrategy := map[string]any{"type": strategyType}
			if strategyType == "RollingUpdate" && input.StatefulSetPartition > 0 {
				updateStrategy["rollingUpdate"] = map[string]any{"partition": input.StatefulSetPartition}
			}
			spec["updateStrategy"] = updateStrategy
		}
		if strings.TrimSpace(input.PodManagementPolicy) != "" {
			spec["podManagementPolicy"] = strings.TrimSpace(input.PodManagementPolicy)
		}
		if claimTemplates, err := buildVolumeClaimTemplates(input.ClaimTemplates); err != nil {
			return nil, "", err
		} else if len(claimTemplates) > 0 {
			spec["volumeClaimTemplates"] = claimTemplates
		}
		return map[string]any{"apiVersion": "apps/v1", "kind": "StatefulSet", "metadata": metadata, "spec": spec}, "StatefulSet", nil
	case "DaemonSet":
		spec := map[string]any{"selector": map[string]any{"matchLabels": labelMap}, "template": template}
		if strategy := buildUpdateStrategy(input.UpdateStrategy, input.MaxSurge, input.MaxUnavailable); len(strategy) > 0 {
			spec["updateStrategy"] = strategy
		}
		return map[string]any{"apiVersion": "apps/v1", "kind": "DaemonSet", "metadata": metadata, "spec": spec}, "DaemonSet", nil
	case "Job":
		jobPodSpec := cloneShallowMap(podSpec)
		jobPodSpec["restartPolicy"] = "OnFailure"
		spec := map[string]any{
			"completions": replicas,
			"parallelism": replicas,
			"template": map[string]any{
				"metadata": template["metadata"],
				"spec":     jobPodSpec,
			},
		}
		return map[string]any{"apiVersion": "batch/v1", "kind": "Job", "metadata": metadata, "spec": spec}, "Job", nil
	case "CronJob":
		schedule := coalesceString(strings.TrimSpace(input.Schedule), "*/5 * * * *")
		cronPodSpec := cloneShallowMap(podSpec)
		cronPodSpec["restartPolicy"] = "OnFailure"
		spec := map[string]any{
			"schedule": schedule,
			"suspend":  input.Suspend,
			"jobTemplate": map[string]any{
				"spec": map[string]any{
					"template": map[string]any{
						"metadata": template["metadata"],
						"spec":     cronPodSpec,
					},
				},
			},
		}
		if strings.TrimSpace(input.ConcurrencyPolicy) != "" {
			spec["concurrencyPolicy"] = strings.TrimSpace(input.ConcurrencyPolicy)
		}
		if input.StartingDeadlineSeconds > 0 {
			spec["startingDeadlineSeconds"] = input.StartingDeadlineSeconds
		}
		if input.SuccessfulJobsHistory > 0 {
			spec["successfulJobsHistoryLimit"] = input.SuccessfulJobsHistory
		}
		if input.FailedJobsHistory > 0 {
			spec["failedJobsHistoryLimit"] = input.FailedJobsHistory
		}
		return map[string]any{"apiVersion": "batch/v1", "kind": "CronJob", "metadata": metadata, "spec": spec}, "CronJob", nil
	default:
		return nil, "", fmt.Errorf("unsupported workload type: %s", workloadType)
	}
}

func buildUpdateStrategy(strategyType, maxSurge, maxUnavailable string) map[string]any {
	strategyType = strings.TrimSpace(strategyType)
	if strategyType == "" {
		return nil
	}
	strategy := map[string]any{"type": strategyType}
	if strategyType == "RollingUpdate" {
		rolling := map[string]any{}
		if strings.TrimSpace(maxSurge) != "" {
			rolling["maxSurge"] = normalizeIntOrString(maxSurge)
		}
		if strings.TrimSpace(maxUnavailable) != "" {
			rolling["maxUnavailable"] = normalizeIntOrString(maxUnavailable)
		}
		if len(rolling) > 0 {
			strategy["rollingUpdate"] = rolling
		}
	}
	return strategy
}

func parseManifestDocuments(raw, defaultNamespace string) ([]map[string]any, []string, error) {
	decoder := yaml.NewDecoder(strings.NewReader(raw))
	objects := make([]map[string]any, 0, 4)
	kinds := make([]string, 0, 4)
	for {
		var doc map[string]any
		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		if len(doc) == 0 {
			continue
		}
		kind, _ := doc["kind"].(string)
		if strings.TrimSpace(kind) == "" {
			return nil, nil, errors.New("manifest kind is required")
		}
		if shouldDefaultNamespace(kind) && strings.TrimSpace(defaultNamespace) != "" {
			metadata, _ := doc["metadata"].(map[string]any)
			if metadata == nil {
				metadata = map[string]any{}
				doc["metadata"] = metadata
			}
			if currentNamespace, _ := metadata["namespace"].(string); strings.TrimSpace(currentNamespace) == "" {
				metadata["namespace"] = strings.TrimSpace(defaultNamespace)
			}
		}
		objects = append(objects, doc)
		kinds = append(kinds, kind)
	}
	return objects, kinds, nil
}

func shouldDefaultNamespace(kind string) bool {
	switch kind {
	case "Namespace", "Node", "PersistentVolume", "StorageClass", "ClusterRole", "ClusterRoleBinding", "CustomResourceDefinition":
		return false
	default:
		return true
	}
}

func parseKeyValueBlock(raw string) (map[string]string, error) {
	out := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid config line: %s", line)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("invalid config key in line: %s", line)
		}
		out[key] = parts[1]
	}
	return out, nil
}

func parseOptionalStringMap(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]string{}, nil
	}
	return parseKeyValueBlock(raw)
}

func parseOptionalYAMLMap(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	if err := yaml.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseOptionalYAMLList(raw string) ([]map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return []map[string]any{}, nil
	}
	var out []map[string]any
	if err := yaml.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseMultilineValues(raw string) []string {
	items := make([]string, 0)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			items = append(items, line)
		}
	}
	return items
}

func parseContainerPorts(raw string, fallback int) ([]map[string]any, int, error) {
	values := make([]int, 0)
	text := strings.TrimSpace(raw)
	if text == "" && fallback > 0 {
		values = append(values, fallback)
	}
	for _, item := range strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == '\n' || r == ' ' }) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		port, err := strconv.Atoi(item)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid container port: %s", item)
		}
		values = append(values, port)
	}
	ports := make([]map[string]any, 0, len(values))
	for _, port := range values {
		ports = append(ports, map[string]any{"containerPort": port})
	}
	if len(values) == 0 {
		return ports, 0, nil
	}
	return ports, values[0], nil
}

func parseServicePorts(raw string) ([]map[string]any, error) {
	ports := make([]map[string]any, 0)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid service port line: %s", line)
		}
		port, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid service port: %s", strings.TrimSpace(parts[0]))
		}
		target := strings.TrimSpace(parts[1])
		if target == "" {
			return nil, fmt.Errorf("invalid target port in line: %s", line)
		}
		entry := map[string]any{
			"port":       port,
			"targetPort": parsePortOrString(target),
			"protocol":   "TCP",
		}
		if len(parts) >= 3 && strings.TrimSpace(parts[2]) != "" {
			entry["name"] = strings.TrimSpace(parts[2])
		}
		if len(parts) >= 4 && strings.TrimSpace(parts[3]) != "" {
			entry["protocol"] = strings.ToUpper(strings.TrimSpace(parts[3]))
		}
		if len(parts) >= 5 && strings.TrimSpace(parts[4]) != "" {
			nodePort, err := strconv.Atoi(strings.TrimSpace(parts[4]))
			if err != nil {
				return nil, fmt.Errorf("invalid nodePort in line: %s", line)
			}
			entry["nodePort"] = nodePort
		}
		ports = append(ports, entry)
	}
	return ports, nil
}

func parseIngressRules(raw string) ([]map[string]any, error) {
	rules := make([]map[string]any, 0)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			return nil, fmt.Errorf("invalid ingress rule line: %s", line)
		}
		host := strings.TrimSpace(parts[0])
		path := strings.TrimSpace(parts[1])
		pathType := strings.TrimSpace(parts[2])
		service := strings.TrimSpace(parts[3])
		servicePort := strings.TrimSpace(parts[4])
		if host == "" || service == "" || servicePort == "" {
			return nil, fmt.Errorf("invalid ingress rule line: %s", line)
		}
		rules = append(rules, map[string]any{
			"host": host,
			"http": map[string]any{
				"paths": []map[string]any{{
					"path":     firstNonEmpty(path, "/"),
					"pathType": firstNonEmpty(pathType, "Prefix"),
					"backend": map[string]any{
						"service": map[string]any{
							"name": service,
							"port": map[string]any{
								"number": parsePortOrString(servicePort),
							},
						},
					},
				}},
			},
		})
	}
	return rules, nil
}

func parseIngressTLS(raw string) ([]map[string]any, error) {
	items := make([]map[string]any, 0)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid ingress tls line: %s", line)
		}
		secretName := strings.TrimSpace(parts[0])
		if secretName == "" {
			return nil, fmt.Errorf("invalid ingress tls secret in line: %s", line)
		}
		hosts := make([]string, 0)
		for _, host := range strings.Split(parts[1], "|") {
			host = strings.TrimSpace(host)
			if host != "" {
				hosts = append(hosts, host)
			}
		}
		items = append(items, map[string]any{"secretName": secretName, "hosts": hosts})
	}
	return items, nil
}

func parsePortOrString(value string) any {
	if number, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		return number
	}
	return strings.TrimSpace(value)
}

func buildEnvVars(raw string) ([]map[string]any, error) {
	envMap, err := parseOptionalStringMap(raw)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(envMap))
	for key, value := range envMap {
		items = append(items, map[string]any{"name": key, "value": value})
	}
	return items, nil
}
