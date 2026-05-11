package queue

import (
	"context"
	"sync"
)

type memoryMessage struct {
	Topic   string
	Payload []byte
}

type MemoryDispatcher struct {
	mu       sync.Mutex
	messages []memoryMessage
}

func NewMemoryDispatcher() *MemoryDispatcher {
	return &MemoryDispatcher{messages: make([]memoryMessage, 0, 64)}
}

func (d *MemoryDispatcher) Driver() string { return "memory" }

func (d *MemoryDispatcher) Ping(context.Context) error { return nil }

func (d *MemoryDispatcher) Publish(_ context.Context, topic string, payload []byte) error {
	d.mu.Lock()
	d.messages = append(d.messages, memoryMessage{Topic: topic, Payload: append([]byte(nil), payload...)})
	d.mu.Unlock()
	return nil
}

func (d *MemoryDispatcher) Close() error { return nil }
