// Package lock provides distributed locking functionality to prevent concurrent access
// to shared resources in the signing service.
package lock

import (
	"context"
)

// Lock represents an acquired lock that can be released
type Lock interface {
	Unlock()
}

// Locker interface for distributed lock service could be done with redis
// Generic interface that can lock on any comparable type (e.g., UUID, string, int)
type Locker[I comparable] interface {
	Acquire(context.Context, I) (Lock, error)
}
