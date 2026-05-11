package worker

import (
	"context"
	"testing"

	"cluster-addon-worker/internal/domain"
)

func TestInstallAddonRejectsUnsupportedInstallMode(t *testing.T) {
	t.Parallel()

	result := installAddon(context.Background(), &domain.Claim{
		AddonKey:    "aws_load_balancer_controller",
		InstallMode: "manual",
	})
	if result.Status != "failed" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	if result.ErrorMessage != "unsupported addon install mode" {
		t.Fatalf("unexpected error: %s", result.ErrorMessage)
	}
}

func TestInstallAddonRejectsUnsupportedAddonKey(t *testing.T) {
	t.Parallel()

	result := installAddon(context.Background(), &domain.Claim{
		AddonKey:    "unknown-addon",
		InstallMode: "platform_managed_helm",
	})
	if result.Status != "failed" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	if result.ErrorMessage != "unsupported addon key" {
		t.Fatalf("unexpected error: %s", result.ErrorMessage)
	}
}
