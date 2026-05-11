package cache

import (
	"context"
	"errors"
	"time"

	"backend-center/internal/config"
)

type Store interface {
	Driver() string
	Ping(context.Context) error
	Get(context.Context, string) (string, bool, error)
	Set(context.Context, string, string, time.Duration) error
	Delete(context.Context, string) error
	Close() error
}

func New(cfg config.CacheConfig) (Store, error) {
	switch cfg.Driver {
	case "", "memory":
		return NewMemoryStore(), nil
	case "redis":
		if cfg.RedisURL == "" {
			return nil, errors.New("cache redis driver requires CACHE_REDIS_URL")
		}
		return NewRedisStore(cfg.RedisURL)
	default:
		return nil, errors.New("unsupported cache driver: " + cfg.Driver)
	}
}
