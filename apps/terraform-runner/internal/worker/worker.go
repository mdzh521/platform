package worker

import (
	"time"

	"terraform-runner/internal/runner"
)

type JobRunner = runner.JobRunner
type JobRunnerConfig = runner.JobRunnerConfig
type WorkspaceManager = runner.WorkspaceManager
type TerraformExecutor = runner.TerraformExecutor

func NewJobRunner(cfg JobRunnerConfig) *JobRunner {
	return runner.NewJobRunner(cfg)
}

func NewWorkspaceManager(root, providerCacheDir, templateRoot string, successTTL, failureTTL, destroyedTTL time.Duration) *WorkspaceManager {
	return runner.NewWorkspaceManager(root, providerCacheDir, templateRoot, successTTL, failureTTL, destroyedTTL)
}

func NewTerraformExecutor(workspaces *WorkspaceManager) *TerraformExecutor {
	return runner.NewTerraformExecutor(workspaces)
}
