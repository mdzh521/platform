package runtime

import (
	"context"
	"log"
	"net/http"
	"time"

	"cluster-enrollment-worker/internal/client/controlplane"
	"cluster-enrollment-worker/internal/config"
	"cluster-enrollment-worker/internal/worker"
)

type Runtime struct {
	runner *worker.Runner
	server *http.Server
}

func New(cfg config.Config) *Runtime {
	backend := controlplane.NewBackendAPI(cfg.BackendBaseURL, cfg.BackendToken)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", worker.HealthHandler())
	return &Runtime{
		runner: worker.NewRunner(cfg.PollInterval, worker.NewExecutor(cfg.WorkerName, backend)),
		server: &http.Server{Addr: cfg.ListenAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second},
	}
}

func (r *Runtime) Run(ctx context.Context) error {
	go func() {
		log.Printf("cluster-enrollment-worker listening on %s", r.server.Addr)
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("cluster-enrollment-worker http stopped: %v", err)
		}
	}()
	return r.runner.Run(ctx)
}
