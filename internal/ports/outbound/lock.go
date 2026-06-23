package outbound

import (
	"context"
	"errors"
	"time"
)

// ErrLockNotAcquired is returned by Acquire when the lock is already held.
var ErrLockNotAcquired = errors.New("lock: not acquired")

// DistributedLocker grants mutually-exclusive named locks across processes (the
// worker uses it so only one instance runs a scheduled job). The TTL bounds how
// long a crashed holder can keep the lock.
type DistributedLocker interface {
	// Acquire takes the lock for key; on success it returns a release func.
	// It returns ErrLockNotAcquired if the lock is held elsewhere.
	Acquire(ctx context.Context, key string, ttl time.Duration) (release func(ctx context.Context) error, err error)
}
