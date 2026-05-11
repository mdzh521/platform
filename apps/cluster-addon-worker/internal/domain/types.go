package domain

type Claim struct {
	ExecutionID uint              `json:"execution_id"`
	ClusterID   uint              `json:"cluster_id"`
	ClusterName string            `json:"cluster_name"`
	Provider    string            `json:"provider"`
	AccountID   uint              `json:"account_id"`
	Region      string            `json:"region"`
	AddonKey    string            `json:"addon_key"`
	AddonType   string            `json:"addon_type"`
	InstallMode string            `json:"install_mode"`
	ReleaseName string            `json:"release_name"`
	Namespace   string            `json:"namespace"`
	Contract    map[string]any    `json:"contract"`
	Environment map[string]string `json:"environment"`
}

type Result struct {
	WorkerName   string         `json:"worker_name"`
	Status       string         `json:"status"`
	ErrorMessage string         `json:"error_message,omitempty"`
	Result       map[string]any `json:"result,omitempty"`
}
