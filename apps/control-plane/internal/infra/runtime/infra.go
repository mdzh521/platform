package runtime

import (
	"context"
	"time"

	"backend-center/internal/infra/cache"
	"backend-center/internal/infra/queue"
)

type Infra struct {
	Cache cache.Store
	Queue queue.Dispatcher
}

type DriverHealth struct {
	Driver  string `json:"driver"`
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
}

type HealthReport struct {
	Status string       `json:"status"`
	Cache  DriverHealth `json:"cache"`
	Queue  DriverHealth `json:"queue"`
}

func (i *Infra) Health(ctx context.Context) HealthReport {
	report := HealthReport{
		Status: "ok",
		Cache:  driverHealth(ctx, i.Cache),
		Queue:  driverHealth(ctx, i.Queue),
	}
	if !report.Cache.Healthy || !report.Queue.Healthy {
		report.Status = "degraded"
	}
	return report
}

type pinger interface {
	Driver() string
	Ping(context.Context) error
}

func driverHealth(ctx context.Context, target pinger) DriverHealth {
	if target == nil {
		return DriverHealth{Driver: "disabled", Healthy: false, Error: "not configured"}
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := target.Ping(pingCtx)
	health := DriverHealth{Driver: target.Driver(), Healthy: err == nil}
	if err != nil {
		health.Error = err.Error()
	}
	return health
}
