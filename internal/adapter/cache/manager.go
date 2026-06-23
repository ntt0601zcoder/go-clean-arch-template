// Package cache provides the eko/gocache-backed implementation of the
// outbound.CacheManager[T] port. A single Manager[T] type fronts both the
// distributed (Redis) and process-local (in-memory) backends; the constructor
// selected at wiring time decides which store is used, so callers never depend
// on the concrete backend.
package cache

import (
	"context"
	"time"

	lib_cache "github.com/eko/gocache/lib/v4/cache"
	lib_store "github.com/eko/gocache/lib/v4/store"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// Manager adapts an eko/gocache typed cache to outbound.CacheManager[T]. The
// embedded *lib_cache.Cache[T] is backend-agnostic, so the same Manager serves
// either Redis or the in-memory store depending on how it was constructed.
type Manager[T any] struct {
	cache *lib_cache.Cache[T]
}

// Get returns the cached value for key. A cache miss surfaces as a non-nil
// error from the underlying store (eko does not distinguish miss from failure).
func (m *Manager[T]) Get(ctx context.Context, key string) (T, error) {
	return m.cache.Get(ctx, key)
}

// GetOnce implements cache-aside: it returns the cached value if present,
// otherwise it computes the value via loader, stores it with the given options
// and returns it. A failed lookup is treated as a miss and triggers loader.
func (m *Manager[T]) GetOnce(
	ctx context.Context,
	key string,
	loader func(ctx context.Context) (T, error),
	options ...lib_store.Option,
) (T, error) {
	if value, err := m.cache.Get(ctx, key); err == nil {
		return value, nil
	}

	value, err := loader(ctx)
	if err != nil {
		var zero T
		return zero, err
	}

	if err := m.cache.Set(ctx, key, value, options...); err != nil {
		var zero T
		return zero, err
	}

	return value, nil
}

// Set stores value under key with optional store options (e.g. WithExpiration).
func (m *Manager[T]) Set(ctx context.Context, key string, value T, options ...lib_store.Option) error {
	return m.cache.Set(ctx, key, value, options...)
}

// SetMulti stores each key/value pair under a shared expiration. It writes
// entries one at a time and stops at the first failure.
func (m *Manager[T]) SetMulti(ctx context.Context, keys []string, values []T, expiration time.Duration) error {
	for i, key := range keys {
		if err := m.cache.Set(ctx, key, values[i], lib_store.WithExpiration(expiration)); err != nil {
			return err
		}
	}
	return nil
}

// Delete evicts key from the cache.
func (m *Manager[T]) Delete(ctx context.Context, key string) error {
	return m.cache.Delete(ctx, key)
}

var _ outbound.CacheManager[any] = (*Manager[any])(nil)
