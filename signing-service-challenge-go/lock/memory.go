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
	// Step 1: Check if a lock already exists for this ID
	m.mu.Lock()
	l, exists := m.registry[id]
	m.mu.Unlock()

	for {
		if exists {
			// Step 2: If lock exists, wait for it to be released or context to be cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-l.wait:
			}
		}

		// Step 3: Re-check if lock still exists after waiting
		m.mu.Lock()
		l, exists = m.registry[id]
		if exists {
			// Step 4: If lock still exists, continue waiting
			m.mu.Unlock()
			continue
		}

		// Step 5: Create new lock with cleanup function
		l = &lock{
			wait: make(chan struct{}), // Channel that will be closed when lock is released
			remove: func() {
				m.mu.Lock()
				defer m.mu.Unlock()
				delete(m.registry, id)
			},
		}

		// Step 6: Register the new lock and break out of loop
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
