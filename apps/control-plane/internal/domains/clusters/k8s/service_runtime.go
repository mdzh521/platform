package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend-center/internal/infra/cache"
)

const (
	clusterListCacheTTL     = 15 * time.Second
	clusterOverviewCacheTTL = 10 * time.Second
)

func (s *Service) clusterListCacheKey() string {
	return "k8s:clusters:list"
}

func (s *Service) clusterOverviewCacheKey(id uint) string {
	return fmt.Sprintf("k8s:clusters:%d:overview", id)
}

func withCachedJSON[T any](ctx context.Context, store cache.Store, key string, ttl time.Duration, loader func() (T, error)) (T, error) {
	var zero T
	if store != nil {
		if raw, ok, err := store.Get(ctx, key); err == nil && ok && raw != "" {
			var cached T
			if err := json.Unmarshal([]byte(raw), &cached); err == nil {
				return cached, nil
			}
		}
	}
	value, err := loader()
	if err != nil {
		return zero, err
	}
	if store != nil {
		if payload, err := json.Marshal(value); err == nil {
			_ = store.Set(ctx, key, string(payload), ttl)
		}
	}
	return value, nil
}

func (s *Service) invalidateClusterCaches(ctx context.Context, clusterID uint) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Delete(ctx, s.clusterListCacheKey())
	if clusterID > 0 {
		_ = s.cache.Delete(ctx, s.clusterOverviewCacheKey(clusterID))
	}
}

func (s *Service) publishEvent(ctx context.Context, topic string, payload map[string]any) {
	if s.queue == nil {
		return
	}
	payload["topic"] = topic
	payload["emitted_at"] = time.Now().UTC().Format(time.RFC3339)
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = s.queue.Publish(ctx, topic, body)
}
