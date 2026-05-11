package k8s

import "net/http"

type namespaceRemoteItem struct {
	Name   string
	Labels map[string]string
}

type versionPayload struct {
	GitVersion string `json:"gitVersion"`
}

type namespacesPayload struct {
	Items []struct {
		Metadata struct {
			Name   string            `json:"name"`
			Labels map[string]string `json:"labels"`
		} `json:"metadata"`
	} `json:"items"`
}

type namespacesOverviewPayload struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Phase string `json:"phase"`
		} `json:"status"`
	} `json:"items"`
}

type deploymentsPayload struct {
	Items []struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Replicas int `json:"replicas"`
			Template struct {
				Spec struct {
					Containers []struct {
						Image string `json:"image"`
					} `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
		Status struct {
			ReadyReplicas int `json:"readyReplicas"`
		} `json:"status"`
	} `json:"items"`
}

type daemonSetsPayload struct {
	Items []struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Template struct {
				Spec struct {
					Containers []struct {
						Image string `json:"image"`
					} `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
		Status struct {
			DesiredNumberScheduled int `json:"desiredNumberScheduled"`
			NumberReady            int `json:"numberReady"`
		} `json:"status"`
	} `json:"items"`
}

type daemonSetDetailPayload struct {
	Spec struct {
		Selector struct {
			MatchLabels map[string]string `json:"matchLabels"`
		} `json:"selector"`
	} `json:"spec"`
}

type deploymentDetailPayload struct {
	Spec struct {
		Selector struct {
			MatchLabels map[string]string `json:"matchLabels"`
		} `json:"selector"`
	} `json:"spec"`
}

type statefulSetsPayload struct {
	Items []struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Replicas int `json:"replicas"`
			Template struct {
				Spec struct {
					Containers []struct {
						Image string `json:"image"`
					} `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
		Status struct {
			ReadyReplicas int `json:"readyReplicas"`
		} `json:"status"`
	} `json:"items"`
}

type jobsPayload struct {
	Items []struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Parallelism int `json:"parallelism"`
			Completions int `json:"completions"`
			Template    struct {
				Spec struct {
					Containers []struct {
						Image string `json:"image"`
					} `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
		Status struct {
			Active    int `json:"active"`
			Succeeded int `json:"succeeded"`
			Failed    int `json:"failed"`
		} `json:"status"`
	} `json:"items"`
}

type jobDetailPayload struct {
	Spec struct {
		Selector struct {
			MatchLabels map[string]string `json:"matchLabels"`
		} `json:"selector"`
	} `json:"spec"`
}

type cronJobsPayload struct {
	Items []struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Schedule    string `json:"schedule"`
			Suspend     bool   `json:"suspend"`
			JobTemplate struct {
				Spec struct {
					Template struct {
						Spec struct {
							Containers []struct {
								Image string `json:"image"`
							} `json:"containers"`
						} `json:"spec"`
					} `json:"template"`
				} `json:"spec"`
			} `json:"jobTemplate"`
		} `json:"spec"`
		Status struct {
			Active []struct {
				Name string `json:"name"`
			} `json:"active"`
		} `json:"status"`
	} `json:"items"`
}

type statefulSetDetailPayload struct {
	Metadata struct {
		Generation int64 `json:"generation"`
	} `json:"metadata"`
	Spec struct {
		Replicas int `json:"replicas"`
		Selector struct {
			MatchLabels map[string]string `json:"matchLabels"`
		} `json:"selector"`
		UpdateStrategy struct {
			Type string `json:"type"`
		} `json:"updateStrategy"`
	} `json:"spec"`
	Status struct {
		ObservedGeneration int64  `json:"observedGeneration"`
		ReadyReplicas      int    `json:"readyReplicas"`
		CurrentReplicas    int    `json:"currentReplicas"`
		UpdatedReplicas    int    `json:"updatedReplicas"`
		CurrentRevision    string `json:"currentRevision"`
		UpdateRevision     string `json:"updateRevision"`
	} `json:"status"`
}

type deploymentRolloutPayload struct {
	Metadata struct {
		Generation int64 `json:"generation"`
	} `json:"metadata"`
	Spec struct {
		Replicas int `json:"replicas"`
		Strategy struct {
			Type          string `json:"type"`
			RollingUpdate struct {
				MaxSurge       any `json:"maxSurge"`
				MaxUnavailable any `json:"maxUnavailable"`
			} `json:"rollingUpdate"`
		} `json:"strategy"`
		Selector struct {
			MatchLabels map[string]string `json:"matchLabels"`
		} `json:"selector"`
	} `json:"spec"`
	Status struct {
		ObservedGeneration  int64 `json:"observedGeneration"`
		ReadyReplicas       int   `json:"readyReplicas"`
		UpdatedReplicas     int   `json:"updatedReplicas"`
		AvailableReplicas   int   `json:"availableReplicas"`
		UnavailableReplicas int   `json:"unavailableReplicas"`
		Conditions          []struct {
			Type    string `json:"type"`
			Status  string `json:"status"`
			Reason  string `json:"reason"`
			Message string `json:"message"`
		} `json:"conditions"`
	} `json:"status"`
}

type replicaSetsPayload struct {
	Items []struct {
		Metadata struct {
			Name              string            `json:"name"`
			CreationTimestamp string            `json:"creationTimestamp"`
			Annotations       map[string]string `json:"annotations"`
		} `json:"metadata"`
		Spec struct {
			Replicas int `json:"replicas"`
		} `json:"spec"`
		Status struct {
			ReadyReplicas     int `json:"readyReplicas"`
			AvailableReplicas int `json:"availableReplicas"`
		} `json:"status"`
	} `json:"items"`
}

type podsPayload struct {
	Items []struct {
		Metadata struct {
			Name      string            `json:"name"`
			Namespace string            `json:"namespace"`
			Labels    map[string]string `json:"labels"`
		} `json:"metadata"`
		Spec struct {
			Containers []struct {
				Image string `json:"image"`
			} `json:"containers"`
		} `json:"spec"`
		Status struct {
			Phase string `json:"phase"`
		} `json:"status"`
	} `json:"items"`
}

type podDetailPayload struct {
	Metadata struct {
		Name      string            `json:"name"`
		Namespace string            `json:"namespace"`
		Labels    map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		Containers []struct {
			Image string `json:"image"`
		} `json:"containers"`
	} `json:"spec"`
	Status struct {
		Phase string `json:"phase"`
	} `json:"status"`
}

type eventsPayload struct {
	Items []struct {
		Metadata struct {
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		InvolvedObject struct {
			Kind      string `json:"kind"`
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"involvedObject"`
		Reason         string `json:"reason"`
		Message        string `json:"message"`
		Type           string `json:"type"`
		LastTimestamp  string `json:"lastTimestamp"`
		EventTime      string `json:"eventTime"`
		FirstTimestamp string `json:"firstTimestamp"`
		Count          int    `json:"count"`
		Source         struct {
			Component string `json:"component"`
		} `json:"source"`
	} `json:"items"`
}

type nodesPayload struct {
	Items []struct {
		Metadata struct {
			Name              string            `json:"name"`
			CreationTimestamp string            `json:"creationTimestamp"`
			Labels            map[string]string `json:"labels"`
		} `json:"metadata"`
		Spec struct {
			PodCIDR string `json:"podCIDR"`
		} `json:"spec"`
		Status struct {
			NodeInfo struct {
				KubeletVersion string `json:"kubeletVersion"`
				OSImage        string `json:"osImage"`
			} `json:"nodeInfo"`
			Addresses []struct {
				Type    string `json:"type"`
				Address string `json:"address"`
			} `json:"addresses"`
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
		} `json:"status"`
	} `json:"items"`
}

type resourceQuotasPayload struct {
	Items []struct{} `json:"items"`
}

type limitRangesPayload struct {
	Items []struct{} `json:"items"`
}

type servicesPayload struct {
	Items []struct {
		Metadata struct {
			Name              string `json:"name"`
			CreationTimestamp string `json:"creationTimestamp"`
		} `json:"metadata"`
		Spec struct {
			ClusterIP string            `json:"clusterIP"`
			Type      string            `json:"type"`
			Selector  map[string]string `json:"selector"`
			Ports     []struct {
				Port       int `json:"port"`
				TargetPort any `json:"targetPort"`
			} `json:"ports"`
		} `json:"spec"`
	} `json:"items"`
}

type ingressesPayload struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			LoadBalancer struct {
				Ingress []struct {
					IP       string `json:"ip"`
					Hostname string `json:"hostname"`
				} `json:"ingress"`
			} `json:"loadBalancer"`
		} `json:"status"`
		Spec struct {
			Rules []struct {
				Host string `json:"host"`
				HTTP struct {
					Paths []struct {
						Path    string `json:"path"`
						Backend struct {
							Service struct {
								Name string `json:"name"`
								Port struct {
									Number int    `json:"number"`
									Name   string `json:"name"`
								} `json:"port"`
							} `json:"service"`
						} `json:"backend"`
					} `json:"paths"`
				} `json:"http"`
			} `json:"rules"`
		} `json:"spec"`
	} `json:"items"`
}

type endpointsPayload struct {
	Subsets []struct {
		Addresses []struct {
			IP string `json:"ip"`
		} `json:"addresses"`
		Ports []struct {
			Port int `json:"port"`
		} `json:"ports"`
	} `json:"subsets"`
}

type serviceDetailPayload struct {
	Metadata struct {
		Name              string            `json:"name"`
		CreationTimestamp string            `json:"creationTimestamp"`
		Annotations       map[string]string `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		Type            string            `json:"type"`
		ClusterIP       string            `json:"clusterIP"`
		SessionAffinity string            `json:"sessionAffinity"`
		Selector        map[string]string `json:"selector"`
		ExternalIPs     []string          `json:"externalIPs"`
		Ports           []struct {
			Name       string `json:"name"`
			Protocol   string `json:"protocol"`
			Port       int    `json:"port"`
			TargetPort any    `json:"targetPort"`
			NodePort   int    `json:"nodePort"`
		} `json:"ports"`
	} `json:"spec"`
}

type ingressDetailPayload struct {
	Metadata struct {
		Name              string            `json:"name"`
		Annotations       map[string]string `json:"annotations"`
		CreationTimestamp string            `json:"creationTimestamp"`
	} `json:"metadata"`
	Status struct {
		LoadBalancer struct {
			Ingress []struct {
				IP       string `json:"ip"`
				Hostname string `json:"hostname"`
			} `json:"ingress"`
		} `json:"loadBalancer"`
	} `json:"status"`
	Spec struct {
		IngressClassName string `json:"ingressClassName"`
		DefaultBackend   struct {
			Service struct {
				Name string `json:"name"`
				Port struct {
					Number int    `json:"number"`
					Name   string `json:"name"`
				} `json:"port"`
			} `json:"service"`
		} `json:"defaultBackend"`
		TLS []struct {
			Hosts      []string `json:"hosts"`
			SecretName string   `json:"secretName"`
		} `json:"tls"`
		Rules []struct {
			Host string `json:"host"`
			HTTP struct {
				Paths []struct {
					Path     string `json:"path"`
					PathType string `json:"pathType"`
					Backend  struct {
						Service struct {
							Name string `json:"name"`
							Port struct {
								Number int    `json:"number"`
								Name   string `json:"name"`
							} `json:"port"`
						} `json:"service"`
					} `json:"backend"`
				} `json:"paths"`
			} `json:"http"`
		} `json:"rules"`
	} `json:"spec"`
}

type configMapsPayload struct {
	Items []struct {
		Metadata struct {
			Name              string `json:"name"`
			CreationTimestamp string `json:"creationTimestamp"`
		} `json:"metadata"`
		Immutable bool              `json:"immutable"`
		Data      map[string]string `json:"data"`
	} `json:"items"`
}

type secretsPayload struct {
	Items []struct {
		Metadata struct {
			Name              string `json:"name"`
			CreationTimestamp string `json:"creationTimestamp"`
		} `json:"metadata"`
		Type string            `json:"type"`
		Data map[string]string `json:"data"`
	} `json:"items"`
}

type configMapDetailPayload struct {
	Metadata struct {
		Name              string            `json:"name"`
		CreationTimestamp string            `json:"creationTimestamp"`
		Annotations       map[string]string `json:"annotations"`
	} `json:"metadata"`
	Immutable bool              `json:"immutable"`
	Data      map[string]string `json:"data"`
}

type secretDetailPayload struct {
	Metadata struct {
		Name              string            `json:"name"`
		CreationTimestamp string            `json:"creationTimestamp"`
		Annotations       map[string]string `json:"annotations"`
	} `json:"metadata"`
	Type string            `json:"type"`
	Data map[string]string `json:"data"`
}

type tokenCredential struct {
	Token              string `json:"token"`
	CACert             string `json:"ca_cert"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
}

type execEnvItem struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type kubeConfig struct {
	CurrentContext string `yaml:"current-context"`
	Clusters       []struct {
		Name    string `yaml:"name"`
		Cluster struct {
			Server                   string `yaml:"server"`
			CertificateAuthorityData string `yaml:"certificate-authority-data"`
			InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
	Contexts []struct {
		Name    string `yaml:"name"`
		Context struct {
			Cluster string `yaml:"cluster"`
			User    string `yaml:"user"`
		} `yaml:"context"`
	} `yaml:"contexts"`
	Users []struct {
		Name string `yaml:"name"`
		User struct {
			Token                 string `yaml:"token"`
			ClientCertificate     string `yaml:"client-certificate"`
			ClientCertificateData string `yaml:"client-certificate-data"`
			ClientKey             string `yaml:"client-key"`
			ClientKeyData         string `yaml:"client-key-data"`
			Exec                  struct {
				Command string        `yaml:"command"`
				Args    []string      `yaml:"args"`
				Env     []execEnvItem `yaml:"env"`
			} `yaml:"exec"`
			AuthProvider struct {
				Name string `yaml:"name"`
			} `yaml:"auth-provider"`
		} `yaml:"user"`
	} `yaml:"users"`
}

type execCredentialOutput struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}

type clusterClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

type workloadRemoteItem struct {
	NamespaceName string
	Kind          string
	Name          string
	ReadyReplicas int
	Replicas      int
	Image         string
	Status        string
}

type relatedPodItem struct {
	Name      string
	Namespace string
	Status    string
	Image     string
}

type rolloutReplicaSetItem struct {
	Name              string
	Replicas          int
	ReadyReplicas     int
	AvailableReplicas int
	Revision          string
	CreationTimestamp string
}

type rolloutDetail struct {
	Strategy            string
	MaxSurge            string
	MaxUnavailable      string
	ObservedGeneration  int64
	Generation          int64
	ReadyReplicas       int
	UpdatedReplicas     int
	AvailableReplicas   int
	UnavailableReplicas int
	CurrentReplicas     int
	DesiredReplicas     int
	CurrentRevision     string
	UpdateRevision      string
	Conditions          []map[string]any
	ReplicaSets         []rolloutReplicaSetItem
}

type relatedServiceItem struct {
	Name      string
	Type      string
	ClusterIP string
	Selector  []string
	Ports     []string
	Endpoints []string
}

type relatedIngressItem struct {
	Name      string
	Hosts     []string
	Paths     []string
	Backends  []string
	Addresses []string
}

type relatedPersistentVolumeClaimItem struct {
	Name             string
	Status           string
	StorageClassName string
	VolumeName       string
	RequestedStorage string
	Capacity         string
	AccessModes      []string
	ClaimTemplate    string
	MountedBy        []map[string]any
}

type statefulSetClaimTemplateItem struct {
	Name             string
	StorageClassName string
	RequestedStorage string
	AccessModes      []string
	Labels           []string
	Annotations      []string
}

type workloadResourceRefs struct {
	ConfigMaps     []string
	Secrets        []string
	PersistentPVCs []string
	ServiceAccount string
	ClaimTemplates []statefulSetClaimTemplateItem
}

type serviceDetailItem struct {
	Name            string
	Type            string
	ClusterIP       string
	SessionAffinity string
	Selector        []string
	ExternalIPs     []string
	Ports           []map[string]any
	Endpoints       []string
	SelectedBy      []map[string]any
	Annotations     map[string]string
	CreatedAt       string
}

type ingressDetailItem struct {
	Name            string
	IngressClass    string
	Addresses       []string
	DefaultBackend  string
	TLS             []map[string]any
	Rules           []map[string]any
	BackendServices []map[string]any
	Annotations     map[string]string
	CreatedAt       string
}

type namespaceResourceItem struct {
	Kind      string
	Name      string
	Namespace string
	Summary   string
	Status    string
	CreatedAt string
}

type configMapDetailItem struct {
	Name         string
	Immutable    bool
	Keys         []string
	DataCount    int
	CreatedAt    string
	Entries      []map[string]any
	ReferencedBy []map[string]any
	Annotations  map[string]string
}

type secretDetailItem struct {
	Name         string
	Type         string
	Keys         []string
	DataCount    int
	CreatedAt    string
	KeySizes     []map[string]any
	ReferencedBy []map[string]any
	Annotations  map[string]string
}

type serviceAccountDetailItem struct {
	Name             string
	Secrets          []string
	ImagePullSecrets []string
	Annotations      map[string]string
	CreatedAt        string
	ReferencedBy     []map[string]any
}

type persistentVolumeClaimDetailItem struct {
	Name             string
	Status           string
	VolumeName       string
	StorageClassName string
	AccessModes      []string
	Capacity         string
	RequestedStorage string
	Annotations      map[string]string
	CreatedAt        string
	MountedBy        []map[string]any
}

type serviceUpdateSpec struct {
	Type            string
	SessionAffinity string
	Selector        map[string]string
	ExternalIPs     []string
	Ports           []map[string]any
}

type secretUpdateSpec struct {
	Entries     map[string]string
	RemoveKeys  []string
	Annotations map[string]string
}

type ingressUpdateSpec struct {
	IngressClass string
	Annotations  map[string]string
	Rules        []map[string]any
	TLS          []map[string]any
}

type resourceQuotaUpdateSpec struct {
	Hard   map[string]string
	Scopes []string
}

type limitRangeUpdateSpec struct {
	Limits []map[string]any
}

type serviceAccountUpdateSpec struct {
	ImagePullSecrets []string
	Annotations      map[string]string
}

type persistentVolumeClaimUpdateSpec struct {
	RequestedStorage string
	Annotations      map[string]string
}

type resourceQuotaDetailItem struct {
	Name              string
	CreatedAt         string
	Hard              map[string]string
	Used              map[string]string
	Scopes            []string
	ImpactedWorkloads []map[string]any
	Annotations       map[string]string
}

type limitRangeDetailItem struct {
	Name              string
	CreatedAt         string
	Limits            []map[string]any
	ImpactedWorkloads []map[string]any
	Annotations       map[string]string
}

type scaleResult struct {
	Replicas int
}

type restartResult struct {
	DeletedPods []string
}

type rolloutRestartResult struct {
	RestartedAt string
}

type statefulSetSettingsItem struct {
	ServiceName         string
	ServiceMode         string
	PodManagementPolicy string
	RollingPartition    int
	UpdateStrategy      string
}

type containerImageItem struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type clusterNodeItem struct {
	Name           string
	Ready          bool
	Roles          []string
	InternalIP     string
	PodCIDR        string
	KubeletVersion string
	OSImage        string
	CreatedAt      string
}

type clusterEventItem struct {
	Type         string
	Namespace    string
	InvolvedKind string
	InvolvedName string
	Reason       string
	Message      string
	Component    string
	Count        int
	Timestamp    string
}

type namespaceGovernanceItem struct {
	Name               string
	Status             string
	ResourceQuotaCount int
	LimitRangeCount    int
}
