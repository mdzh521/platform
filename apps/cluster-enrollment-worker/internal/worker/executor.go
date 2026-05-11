package worker

import (
	"context"

	"cluster-enrollment-worker/internal/domain"
	"cluster-enrollment-worker/internal/handlers"
)

type Executor struct {
	workerName string
	backend    domain.ControlPlane
	registry   handlers.Registry
}

func NewExecutor(workerName string, backend domain.ControlPlane) *Executor {
	return &Executor{
		workerName: workerName,
		backend:    backend,
		registry:   handlers.NewRegistry(),
	}
}

func (e *Executor) Execute(ctx context.Context) error {
	claim, err := e.backend.Claim(ctx, e.workerName)
	if err != nil || claim == nil {
		return err
	}
	result := e.registry.Enroll(ctx, claim)
	result.WorkerName = e.workerName
	return e.backend.Report(ctx, claim.ResourceID, result)
}
