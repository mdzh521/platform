package worker

import (
	"testing"
	"time"
)

func TestWrapperConstructorsExposeRunnerComponents(t *testing.T) {
	workspaces := NewWorkspaceManager("/tmp/jobs", "/tmp/cache", "/tmp/templates", time.Minute, time.Minute, time.Minute)
	if workspaces == nil {
		t.Fatal("expected workspace manager")
	}
	if NewTerraformExecutor(workspaces) == nil {
		t.Fatal("expected terraform executor")
	}
	if NewJobRunner(JobRunnerConfig{}) == nil {
		t.Fatal("expected job runner")
	}
}
