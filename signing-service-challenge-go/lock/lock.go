package lock

import (
	"context"
)

type Lock interface {
	Unlock()
}

// Locker interface for distributed lock service could be done with redis
type Locker[I comparable] interface {
	Acquire(context.Context, I) (Lock, error)
}
