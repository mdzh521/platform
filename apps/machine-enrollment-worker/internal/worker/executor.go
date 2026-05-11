package worker

import (
	"context"

	"machine-enrollment-worker/internal/domain"
)

type Executor struct {
	workerName string
	backend    domain.ControlPlane
}

func NewExecutor(workerName string, backend domain.ControlPlane) *Executor {
	return &Executor{workerName: workerName, backend: backend}
}

func (e *Executor) Execute(ctx context.Context) error {
	claim, err := e.backend.Claim(ctx, e.workerName)
	if err != nil || claim == nil {
		return err
	}
	result := domain.Result{WorkerName: e.workerName, Status: "enrolled"}
	if claim.CloudID == "" && claim.Metadata["cloud_id"] == nil {
		result.Status = "error"
		result.ErrorMessage = "missing cloud resource id"
	}
	return e.backend.Report(ctx, claim.ResourceID, result)
}
