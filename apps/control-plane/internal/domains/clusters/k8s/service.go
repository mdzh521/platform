package k8s

import (
	"backend-center/internal/infra/cache"
	"backend-center/internal/infra/queue"
	"sync"

	"gorm.io/gorm"
)

// Service is the K8s application service entrypoint.
// Keep this file minimal: shared dependencies and constructor only.
// Cluster/workload/resource/exec/terminal logic must stay in the dedicated service_*.go files.
type Service struct {
	db            *gorm.DB
	cache         cache.Store
	queue         queue.Dispatcher
	shellSessions map[string]*shellSession
	shellMu       sync.RWMutex
}

func NewService(db *gorm.DB, cacheStore cache.Store, queueDispatcher queue.Dispatcher) *Service {
	return &Service{
		db:            db,
		cache:         cacheStore,
		queue:         queueDispatcher,
		shellSessions: map[string]*shellSession{},
	}
}
