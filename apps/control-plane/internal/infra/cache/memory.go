package cache

import (
	"context"
	"sync"
	"time"
)

type memoryItem struct {
	value     string
	expiresAt time.Time
}

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]memoryItem
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: map[string]memoryItem{}}
}

func (s *MemoryStore) Driver() string { return "memory" }

func (s *MemoryStore) Ping(context.Context) error { return nil }

func (s *MemoryStore) Get(_ context.Context, key string) (string, bool, error) {
	s.mu.RLock()
	item, ok := s.items[key]
	s.mu.RUnlock()
	if !ok {
		return "", false, nil
	}
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		s.mu.Lock()
		delete(s.items, key)
		s.mu.Unlock()
		return "", false, nil
	}
	return item.value, true, nil
}

func (s *MemoryStore) Set(_ context.Context, key, value string, ttl time.Duration) error {
	item := memoryItem{value: value}
	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}
	s.mu.Lock()
	s.items[key] = item
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) Close() error { return nil }
