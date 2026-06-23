package cache

import (
	"time"

	lib_cache "github.com/eko/gocache/lib/v4/cache"
	gocachestore "github.com/eko/gocache/store/go_cache/v4"
	gocachelib "github.com/patrickmn/go-cache"
)

// NewMemoryManager builds a Manager backed by a process-local in-memory store.
// defaultTTL sets the default item expiration; the underlying go-cache purges
// expired items on an interval of twice the default TTL.
func NewMemoryManager[T any](defaultTTL time.Duration) *Manager[T] {
	return &Manager[T]{
		cache: lib_cache.New[T](gocachestore.NewGoCache(gocachelib.New(defaultTTL, defaultTTL*2))),
	}
}
