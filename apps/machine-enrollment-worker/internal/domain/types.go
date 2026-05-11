package domain

import "context"

type Claim struct {
	ResourceID   uint           `json:"resource_id"`
	Provider     string         `json:"provider"`
	AccountID    uint           `json:"account_id"`
	Region       string         `json:"region"`
	ResourceType string         `json:"resource_type"`
	ResourceName string         `json:"resource_name"`
	CloudID      string         `json:"cloud_id"`
	Metadata     map[string]any `json:"metadata"`
}

type Result struct {
	WorkerName   string `json:"worker_name"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type ControlPlane interface {
	Claim(context.Context, string) (*Claim, error)
	Report(context.Context, uint, Result) error
}
