package worker

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Runner struct {
	pollInterval time.Duration
	executor     *Executor
}

func NewRunner(pollInterval time.Duration, executor *Executor) *Runner {
	return &Runner{
		pollInterval: pollInterval,
		executor:     executor,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.executor.Execute(ctx); err != nil {
				log.Printf("resource sync poll failed: %v", err)
			}
		}
	}
}

func HealthHandler(service string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","service":"%s"}`, service)))
	}
}
