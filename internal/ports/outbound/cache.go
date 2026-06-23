package outbound

import (
	"context"
	"time"

	lib_store "github.com/eko/gocache/lib/v4/store"
)

// CacheManager is the single typed cache port. Both backends — Redis
// (distributed) and in-memory (process-local) — implement it, selected by
// config; callers never know which is wired. T is the cached value type.
//
// store.Option (from eko/gocache) is the one third-party type that leaks into
// this port, matching the reference; it carries per-call options such as TTL.
type CacheManager[T any] interface {
	// Get returns the cached value or an error (cache miss is an error).
	Get(ctx context.Context, key string) (T, error)
	// GetOnce returns the cached value, or computes it via loader, stores it and
	// returns it (cache-aside / single-flight friendly).
	GetOnce(ctx context.Context, key string, loader func(ctx context.Context) (T, error), options ...lib_store.Option) (T, error)
	// Set stores value under key with optional store options (e.g. WithExpiration).
	Set(ctx context.Context, key string, value T, options ...lib_store.Option) error
	// SetMulti stores several key/value pairs with one expiration.
	SetMulti(ctx context.Context, keys []string, values []T, expiration time.Duration) error
	// Delete evicts key.
	Delete(ctx context.Context, key string) error
}
