package handlers

import (
	"context"
	"errors"
	"testing"

	"cluster-enrollment-worker/internal/domain"
)

type stubAdapter struct {
	name   string
	result domain.Result
	err    error
}

func (s stubAdapter) Name() string { return s.name }

func (s stubAdapter) Enroll(_ context.Context, _ *domain.Claim) (domain.Result, error) {
	return s.result, s.err
}

func TestAdapterRegistryFallsBackToNextAdapter(t *testing.T) {
	registry := Registry{
		adapters: map[string][]Provider{
			"aws": {
				stubAdapter{name: "aws-sdk", err: errAdapterNotConfigured},
				stubAdapter{name: "aws-cli", result: domain.Result{Status: "enrolled", ClusterName: "demo"}},
			},
		},
	}
	result := registry.Enroll(context.Background(), &domain.Claim{Provider: "aws"})
	if result.Status != "enrolled" {
		t.Fatalf("expected fallback adapter to succeed, got status=%q error=%q", result.Status, result.ErrorMessage)
	}
	if result.Metadata["adapter"] != "aws-cli" {
		t.Fatalf("expected adapter metadata to record fallback adapter, got %#v", result.Metadata["adapter"])
	}
}

func TestAdapterRegistryAggregatesFailures(t *testing.T) {
	registry := Registry{
		adapters: map[string][]Provider{
			"alicloud": {
				stubAdapter{name: "alicloud-openapi", err: errors.New("sdk disabled")},
				stubAdapter{name: "alicloud-cli", err: errors.New("command not found")},
			},
		},
	}
	result := registry.Enroll(context.Background(), &domain.Claim{Provider: "alicloud"})
	if result.Status != "error" {
		t.Fatalf("expected error result, got status=%q", result.Status)
	}
	if result.ErrorMessage == "" {
		t.Fatal("expected aggregated error message")
	}
}
