package runtime

import (
	"context"
	"log"
	"net/http"
	"time"

	"cluster-addon-worker/internal/client/controlplane"
	"cluster-addon-worker/internal/config"
	"cluster-addon-worker/internal/worker"
)

type Runtime struct {
	worker *worker.Worker
	server *http.Server
}

func New(cfg config.Config) *Runtime {
	backend := controlplane.NewBackendAPI(cfg.BackendBaseURL, cfg.BackendToken)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", worker.HealthHandler())
	return &Runtime{
		worker: worker.New(worker.Config{
			WorkerName:        cfg.WorkerName,
			PollInterval:      cfg.PollInterval,
			HeartbeatInterval: cfg.HeartbeatInterval,
			Backend:           backend,
		}),
		server: &http.Server{
			Addr:              cfg.ListenAddr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (r *Runtime) Run(ctx context.Context) error {
	go func() {
		log.Printf("cluster-addon-worker listening on %s", r.server.Addr)
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("cluster-addon-worker http stopped: %v", err)
		}
	}()
	return r.worker.Run(ctx)
}
