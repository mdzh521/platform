package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTemplateDirUsesTemplatePathFromBackend(t *testing.T) {
	manager := NewWorkspaceManager("/tmp/jobs", "/tmp/cache", "/app/templates", 0, 0, 0)

	got := manager.ResolveTemplateDir("terraform-runner/templates/aws-vpc-base", "ignored-code")
	want := "/app/templates/aws-vpc-base"
	if got != want {
		t.Fatalf("unexpected template resolution: got %q want %q", got, want)
	}
}

func TestResolveTemplateDirFallsBackToBlueprintCode(t *testing.T) {
	manager := NewWorkspaceManager("/tmp/jobs", "/tmp/cache", "/app/templates", 0, 0, 0)

	got := manager.ResolveTemplateDir("", "aws-vpc-base")
	want := "/app/templates/aws-vpc-base"
	if got != want {
		t.Fatalf("unexpected fallback template resolution: got %q want %q", got, want)
	}
}

func TestPrepareWorkspacePreservesExistingTerraformState(t *testing.T) {
	root := t.TempDir()
	cacheDir := filepath.Join(root, "cache")
	workspace := filepath.Join(root, "job-55")
	manager := NewWorkspaceManager(filepath.Join(root, "jobs"), cacheDir, "/app/templates", 0, 0, 0)

	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	statePath := filepath.Join(workspace, "terraform.tfstate")
	if err := os.WriteFile(statePath, []byte("existing-state"), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}
	markerPath := filepath.Join(workspace, "marker.txt")
	if err := os.WriteFile(markerPath, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	if err := manager.PrepareWorkspace(workspace); err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected terraform state to be preserved: %v", err)
	}
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("expected workspace contents to be preserved when state exists: %v", err)
	}
}
