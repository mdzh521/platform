package queue

import (
	"context"
	"time"

	"backend-center/internal/infra/cache"
)

type RedisDispatcher struct {
	client  *cache.RedisStore
	timeout time.Duration
}

func NewRedisDispatcher(rawURL string, timeout time.Duration) (*RedisDispatcher, error) {
	client, err := cache.NewRedisStore(rawURL)
	if err != nil {
		return nil, err
	}
	return &RedisDispatcher{client: client, timeout: timeout}, nil
}

func (d *RedisDispatcher) Driver() string { return "redis" }

func (d *RedisDispatcher) Ping(ctx context.Context) error { return d.client.Ping(ctx) }

func (d *RedisDispatcher) Publish(ctx context.Context, topic string, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	return d.client.Push(ctx, queueKey(topic), string(payload))
}

func (d *RedisDispatcher) Close() error { return d.client.Close() }

func queueKey(topic string) string {
	return "queue:" + topic
}
