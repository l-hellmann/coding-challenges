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
	// Use mutex to safely read from the registry map
	m.mu.Lock()
	l, exists := m.registry[id]
	m.mu.Unlock()

	// Step 2: If lock exists, wait for it to be released or context to be cancelled
	if exists {
		select {
		case <-ctx.Done():
			// Context was cancelled while waiting, return the cancellation error
			return nil, ctx.Err()
		case <-l.wait:
			// Lock was released, continue to try acquiring it
		}
	}

	// Step 3: Attempt to acquire the lock in a loop
	// This loop handles race conditions where multiple goroutines try to acquire the same lock
	for {
		// Step 3a: Check again if lock exists (race condition protection)
		m.mu.Lock()
		l, exists = m.registry[id]
		if exists {
			// Another goroutine acquired the lock between our checks, try again
			m.mu.Unlock()
			continue
		}
		
		// Step 3b: Create a new lock since none exists
		l = &lock{
			wait: make(chan struct{}), // Channel that will be closed when lock is released
			remove: func() {
				// Cleanup function to remove this lock from the registry
				m.mu.Lock()
				defer m.mu.Unlock()
				delete(m.registry, id)
			},
		}
		
		// Step 3c: Register the new lock in the registry
		m.registry[id] = l
		m.mu.Unlock()
		
		// Step 3d: Successfully acquired the lock, exit the loop
		break
	}

	// Step 4: Return the acquired lock
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
