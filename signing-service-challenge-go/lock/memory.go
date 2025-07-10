package lock

import (
	"context"
	"sync"
)

type memoryLocker[I comparable] struct {
	mu       sync.Mutex
	registry map[I]*lock
}

func NewMemoryLocker[I comparable]() Locker[I] {
	return &memoryLocker[I]{
		registry: make(map[I]*lock),
	}
}

func (m *memoryLocker[I]) Acquire(ctx context.Context, id I) (Lock, error) {
	m.mu.Lock()
	l, exists := m.registry[id]
	m.mu.Unlock()

	if exists {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-l.wait:
		}
	}

	for {
		m.mu.Lock()
		l, exists = m.registry[id]
		if exists {
			m.mu.Unlock()
			continue
		}
		l = &lock{
			wait: make(chan struct{}),
			remove: func() {
				m.mu.Lock()
				defer m.mu.Unlock()
				delete(m.registry, id)
			},
		}
		m.registry[id] = l
		m.mu.Unlock()
		break
	}

	return l, nil
}

type lock struct {
	wait   chan struct{}
	remove func()
}

func (l *lock) Unlock() {
	l.remove()
	close(l.wait)
}
