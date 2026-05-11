package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cluster-enrollment-worker/internal/domain"
)

var errAdapterNotConfigured = errors.New("adapter not configured")

type Provider interface {
	Name() string
	Enroll(context.Context, *domain.Claim) (domain.Result, error)
}

type Registry struct {
	adapters map[string][]Provider
}

func NewRegistry() Registry {
	return Registry{
		adapters: map[string][]Provider{
			"aws": {
				AWSSDKAdapter{},
				AWSCLIAdapter{},
			},
			"alicloud": {
				AlicloudOpenAPIAdapter{},
				AlicloudCLIAdapter{},
			},
		},
	}
}

func (r Registry) Enroll(ctx context.Context, claim *domain.Claim) domain.Result {
	provider := strings.ToLower(strings.TrimSpace(claim.Provider))
	adapters := r.adapters[provider]
	if len(adapters) == 0 {
		return domain.Result{Status: "error", ErrorMessage: "unsupported provider"}
	}
	var failures []string
	for _, adapter := range adapters {
		result, err := adapter.Enroll(ctx, claim)
		if err == nil {
			if result.Metadata == nil {
				result.Metadata = map[string]any{}
			}
			if _, exists := result.Metadata["adapter"]; !exists {
				result.Metadata["adapter"] = adapter.Name()
			}
			return result
		}
		failures = append(failures, fmt.Sprintf("%s: %s", adapter.Name(), err.Error()))
	}
	return domain.Result{Status: "error", ErrorMessage: strings.Join(failures, " | ")}
}

type AWSSDKAdapter struct{}

func (AWSSDKAdapter) Name() string { return "aws-sdk" }

type AWSCLIAdapter struct{}

func (AWSCLIAdapter) Name() string { return "aws-cli" }

func (AWSCLIAdapter) Enroll(ctx context.Context, claim *domain.Claim) (domain.Result, error) {
	result := enrollAWSCluster(ctx, claim)
	if result.Status == "error" && strings.TrimSpace(result.ErrorMessage) != "" {
		return domain.Result{}, errors.New(strings.TrimSpace(result.ErrorMessage))
	}
	return result, nil
}

type AlicloudOpenAPIAdapter struct{}

func (AlicloudOpenAPIAdapter) Name() string { return "alicloud-openapi" }

type AlicloudCLIAdapter struct{}

func (AlicloudCLIAdapter) Name() string { return "alicloud-cli" }

func (AlicloudCLIAdapter) Enroll(ctx context.Context, claim *domain.Claim) (domain.Result, error) {
	result := enrollAlicloudCluster(ctx, claim)
	if result.Status == "error" && strings.TrimSpace(result.ErrorMessage) != "" {
		return domain.Result{}, errors.New(strings.TrimSpace(result.ErrorMessage))
	}
	return result, nil
}
