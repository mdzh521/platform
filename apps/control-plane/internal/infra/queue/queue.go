package queue

import (
	"context"
	"errors"
	"time"

	"backend-center/internal/config"
)

type Dispatcher interface {
	Driver() string
	Ping(context.Context) error
	Publish(context.Context, string, []byte) error
	Close() error
}

func New(cfg config.QueueConfig) (Dispatcher, error) {
	switch cfg.Driver {
	case "", "memory":
		return NewMemoryDispatcher(), nil
	case "redis":
		if cfg.RedisURL == "" {
			return nil, errors.New("queue redis driver requires QUEUE_REDIS_URL")
		}
		return NewRedisDispatcher(cfg.RedisURL, 3*time.Second)
	default:
		return nil, errors.New("unsupported queue driver: " + cfg.Driver)
	}
}
