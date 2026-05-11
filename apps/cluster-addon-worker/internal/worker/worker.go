package worker

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"cluster-addon-worker/internal/domain"
	"cluster-addon-worker/internal/worker/handlers"
)

type Backend interface {
	Claim(context.Context, string) (*domain.Claim, error)
	Heartbeat(context.Context, uint, string, string) error
	Report(context.Context, uint, domain.Result) error
}

type Config struct {
	WorkerName        string
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
	Backend           Backend
}

type Worker struct {
	config Config
}

func New(cfg Config) *Worker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 10 * time.Second
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 20 * time.Second
	}
	return &Worker{config: cfg}
}

func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","service":"cluster-addon-worker"}`))
	}
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			claim, err := w.config.Backend.Claim(ctx, w.config.WorkerName)
			if err != nil || claim == nil {
				if err != nil {
					log.Printf("cluster addon claim failed: %v", err)
				}
				continue
			}
			w.processClaim(ctx, claim)
		}
	}
}

func (w *Worker) processClaim(ctx context.Context, claim *domain.Claim) {
	_ = w.config.Backend.Heartbeat(ctx, claim.ExecutionID, w.config.WorkerName, "installing")
	hbCtx, cancel := context.WithCancel(ctx)
	go w.runHeartbeatLoop(hbCtx, claim.ExecutionID)

	result := installAddon(ctx, claim)
	cancel()
	result.WorkerName = w.config.WorkerName
	if result.Status != "succeeded" {
		log.Printf("cluster addon install failed: execution=%d addon=%s error=%s", claim.ExecutionID, claim.AddonKey, result.ErrorMessage)
	}
	if err := w.config.Backend.Report(ctx, claim.ExecutionID, result); err != nil {
		log.Printf("cluster addon report failed: %v", err)
	}
}

func (w *Worker) runHeartbeatLoop(ctx context.Context, executionID uint) {
	ticker := time.NewTicker(w.config.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.config.Backend.Heartbeat(ctx, executionID, w.config.WorkerName, "installing"); err != nil {
				log.Printf("cluster addon heartbeat failed: %v", err)
			}
		}
	}
}

func installAddon(ctx context.Context, claim *domain.Claim) domain.Result {
	if strings.TrimSpace(strings.ToLower(claim.InstallMode)) != "platform_managed_helm" {
		return domain.Result{Status: "failed", ErrorMessage: "unsupported addon install mode"}
	}
	switch strings.TrimSpace(strings.ToLower(claim.AddonKey)) {
	case "aws_load_balancer_controller":
		result, err := handlers.InstallAWSLoadBalancerController(ctx, claim)
		if err != nil {
			return domain.Result{Status: "failed", ErrorMessage: err.Error()}
		}
		return domain.Result{Status: "succeeded", Result: result}
	default:
		return domain.Result{Status: "failed", ErrorMessage: "unsupported addon key"}
	}
}
