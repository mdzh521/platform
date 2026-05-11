package runtime

import (
	"context"
	"log"
	"net/http"
	"time"

	"terraform-runner/internal/client/controlplane"
	"terraform-runner/internal/config"
	"terraform-runner/internal/worker"
)

type App struct {
	config config.Config
	runner *worker.JobRunner
	server *http.Server
}

func New(cfg config.Config) *App {
	backend := controlplane.NewBackendAPI(cfg.BackendBaseURL, cfg.BackendToken)
	workspace := worker.NewWorkspaceManager(cfg.WorkspaceRoot, cfg.ProviderCacheDir, cfg.TemplateRoot, cfg.SuccessTTL, cfg.FailureTTL, cfg.DestroyedTTL)
	executor := worker.NewTerraformExecutor(workspace)

	jobRunner := worker.NewJobRunner(worker.JobRunnerConfig{
		RunnerName:        cfg.RunnerName,
		PollInterval:      cfg.PollInterval,
		HeartbeatInterval: cfg.HeartbeatInterval,
		MaxParallelJobs:   cfg.MaxParallelJobs,
		Backend:           backend,
		Executor:          executor,
		Cleanup:           workspace.CleanupExpired,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","service":"terraform-runner"}`))
	})

	return &App{
		config: cfg,
		runner: jobRunner,
		server: &http.Server{
			Addr:              cfg.ListenAddr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (a *App) Run(ctx context.Context) error {
	go func() {
		log.Printf("terraform-runner listening on %s", a.config.ListenAddr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server stopped: %v", err)
		}
	}()
	return a.runner.Run(ctx)
}
