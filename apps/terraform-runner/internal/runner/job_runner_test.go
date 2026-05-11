package runner

import (
	"context"
	"sync"
	"testing"
	"time"

	"terraform-runner/internal/domain"
)

type fakeBackend struct {
	mu           sync.Mutex
	heartbeats   []string
	reported     []domain.JobResult
	appendedLogs []string
}

func (f *fakeBackend) ClaimNextJob(context.Context, string) (*domain.JobClaim, error) {
	return nil, nil
}

func (f *fakeBackend) Heartbeat(_ context.Context, _ uint, _ string, status string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.heartbeats = append(f.heartbeats, status)
	return nil
}

func (f *fakeBackend) AppendLog(_ context.Context, _ uint, _ string, stage, level, message string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.appendedLogs = append(f.appendedLogs, stage+":"+level+":"+message)
	return nil
}

func (f *fakeBackend) ReportResult(_ context.Context, _ string, result domain.JobResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reported = append(f.reported, result)
	return nil
}

type sleepExecutor struct {
	delay  time.Duration
	result JobExecutionResult
	err    error
}

func (s sleepExecutor) Execute(context.Context, JobExecution) (JobExecutionResult, error) {
	time.Sleep(s.delay)
	return s.result, s.err
}

func TestRunClaimSendsRepeatedHeartbeats(t *testing.T) {
	backend := &fakeBackend{}
	runner := NewJobRunner(JobRunnerConfig{
		RunnerName:        "test-runner",
		PollInterval:      time.Second,
		HeartbeatInterval: 20 * time.Millisecond,
		MaxParallelJobs:   1,
		Backend:           backend,
		Executor: sleepExecutor{
			delay: 120 * time.Millisecond,
			result: JobExecutionResult{
				Status:     "planned",
				LogExcerpt: "ok",
			},
		},
	})

	runner.slots <- struct{}{}
	runner.wg.Add(1)
	runner.runClaim(context.Background(), domain.JobClaim{
		JobID:         42,
		Action:        "plan",
		BlueprintCode: "aws-vpc-base",
	})

	backend.mu.Lock()
	defer backend.mu.Unlock()
	if len(backend.heartbeats) < 3 {
		t.Fatalf("expected repeated heartbeats, got %d", len(backend.heartbeats))
	}
	if len(backend.reported) != 1 {
		t.Fatalf("expected one reported result, got %d", len(backend.reported))
	}
	if backend.reported[0].Status != "planned" {
		t.Fatalf("unexpected result status: %s", backend.reported[0].Status)
	}
}
