package runner

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"terraform-runner/internal/domain"
)

type Backend interface {
	ClaimNextJob(context.Context, string) (*domain.JobClaim, error)
	Heartbeat(context.Context, uint, string, string) error
	AppendLog(context.Context, uint, string, string, string, string) error
	ReportResult(context.Context, string, domain.JobResult) error
}

type Executor interface {
	Execute(context.Context, JobExecution) (JobExecutionResult, error)
}

type JobRunnerConfig struct {
	RunnerName        string
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
	MaxParallelJobs   int
	Backend           Backend
	Executor          Executor
	Cleanup           func() error
}

type JobRunner struct {
	config JobRunnerConfig
	name   string
	slots  chan struct{}
	wg     sync.WaitGroup
}

func NewJobRunner(cfg JobRunnerConfig) *JobRunner {
	name := cfg.RunnerName
	if name == "" {
		name = "terraform-runner"
	}
	maxParallel := cfg.MaxParallelJobs
	if maxParallel <= 0 {
		maxParallel = 1
	}
	return &JobRunner{
		config: cfg,
		name:   name,
		slots:  make(chan struct{}, maxParallel),
	}
}

func (r *JobRunner) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.wg.Wait()
			return ctx.Err()
		case <-ticker.C:
			if r.config.Cleanup != nil {
				if err := r.config.Cleanup(); err != nil {
					log.Printf("workspace cleanup failed: %v", err)
				}
			}
			if err := r.pollAvailable(ctx); err != nil {
				log.Printf("runner poll failed: %v", err)
			}
		}
	}
}

func (r *JobRunner) pollAvailable(ctx context.Context) error {
	for {
		if !r.tryAcquire() {
			return nil
		}
		claim, err := r.config.Backend.ClaimNextJob(ctx, r.name)
		if err != nil {
			r.release()
			return err
		}
		if claim == nil {
			r.release()
			return nil
		}
		r.wg.Add(1)
		go r.runClaim(ctx, *claim)
	}
}

func (r *JobRunner) runClaim(ctx context.Context, claim domain.JobClaim) {
	defer r.release()
	defer r.wg.Done()

	log.Printf("runner %s claimed job=%d action=%s blueprint=%s", r.name, claim.JobID, claim.Action, claim.BlueprintCode)
	phaseStatus := phaseStatusForAction(claim.Action)
	if err := r.config.Backend.Heartbeat(ctx, claim.JobID, r.name, phaseStatus); err != nil {
		log.Printf("heartbeat failed for job %d: %v", claim.JobID, err)
	}
	hbCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	go r.runHeartbeatLoop(hbCtx, claim.JobID, phaseStatus)

	result, err := r.config.Executor.Execute(ctx, JobExecution{
		JobID:         claim.JobID,
		Name:          claim.Name,
		Provider:      claim.Provider,
		Action:        claim.Action,
		AccountID:     claim.AccountID,
		BlueprintID:   claim.BlueprintID,
		NetworkPlanID: claim.NetworkPlanID,
		BlueprintCode: claim.BlueprintCode,
		TemplatePath:  claim.TemplatePath,
		Input:         claim.Input,
		Environment:   claim.Environment,
		WorkspacePath: claim.WorkspacePath,
	})
	if err != nil {
		stage := "runner"
		message := strings.TrimSpace(err.Error())
		if execErr, ok := err.(*ExecutionError); ok {
			if strings.TrimSpace(execErr.Stage) != "" {
				stage = execErr.Stage
			}
			if strings.TrimSpace(execErr.Message) != "" {
				message = strings.TrimSpace(execErr.Message)
			}
		}
		_ = r.config.Backend.AppendLog(ctx, claim.JobID, r.name, stage, "error", message)
		log.Printf("runner %s failed job=%d: %v", r.name, claim.JobID, err)
		if reportErr := r.config.Backend.ReportResult(ctx, r.name, domain.JobResult{
			JobID:        claim.JobID,
			Status:       "failed",
			ErrorMessage: message,
		}); reportErr != nil {
			log.Printf("runner %s report failed job=%d: %v", r.name, claim.JobID, reportErr)
		}
		return
	}
	for _, entry := range result.StageLogs {
		if strings.TrimSpace(entry.Message) == "" {
			continue
		}
		stage := strings.TrimSpace(entry.Stage)
		if stage == "" {
			stage = "runner"
		}
		level := strings.TrimSpace(entry.Level)
		if level == "" {
			level = "info"
		}
		_ = r.config.Backend.AppendLog(ctx, claim.JobID, r.name, stage, level, entry.Message)
	}
	log.Printf("runner %s completed job=%d status=%s", r.name, claim.JobID, result.Status)

	if err := r.config.Backend.ReportResult(ctx, r.name, domain.JobResult{
		JobID:       claim.JobID,
		Status:      result.Status,
		LogExcerpt:  result.LogExcerpt,
		Output:      result.Output,
		PlanSummary: result.PlanSummary,
	}); err != nil {
		log.Printf("runner %s report result failed job=%d: %v", r.name, claim.JobID, err)
	}
}

func (r *JobRunner) runHeartbeatLoop(ctx context.Context, jobID uint, status string) {
	interval := r.config.HeartbeatInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.config.Backend.Heartbeat(ctx, jobID, r.name, status); err != nil {
				log.Printf("heartbeat loop failed for job %d: %v", jobID, err)
			}
		}
	}
}

func (r *JobRunner) tryAcquire() bool {
	select {
	case r.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (r *JobRunner) release() {
	select {
	case <-r.slots:
	default:
	}
}

func (r *JobRunner) pollOnce(ctx context.Context) error {
	claim, err := r.config.Backend.ClaimNextJob(ctx, r.name)
	if err != nil || claim == nil {
		return err
	}
	r.runClaim(ctx, *claim)
	return nil
}

func phaseStatusForAction(action string) string {
	switch action {
	case "apply":
		return "applying"
	case "destroy":
		return "destroying"
	default:
		return "planning"
	}
}
