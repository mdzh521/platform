package runtime

import (
	"context"
	"log"
	"net/http"
	"time"

	"resource-sync-worker/internal/client/controlplane"
	"resource-sync-worker/internal/config"
	"resource-sync-worker/internal/worker"
)

type Runtime struct {
	config config.Config
	runner *worker.Runner
	server *http.Server
}

func New(cfg config.Config) *Runtime {
	backend := controlplane.NewBackendAPI(cfg.BackendBaseURL, cfg.BackendToken)
	executor := worker.NewExecutor(cfg.WorkerName, backend)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", worker.HealthHandler("resource-sync-worker"))

	return &Runtime{
		config: cfg,
		runner: worker.NewRunner(cfg.PollInterval, executor),
		server: &http.Server{
			Addr:              cfg.ListenAddr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (r *Runtime) Run(ctx context.Context) error {
	go func() {
		log.Printf("resource-sync-worker listening on %s", r.config.ListenAddr)
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server stopped: %v", err)
		}
	}()
	return r.runner.Run(ctx)
}
