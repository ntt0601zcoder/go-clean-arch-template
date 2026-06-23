package cache

import (
	lib_cache "github.com/eko/gocache/lib/v4/cache"
	redisstore "github.com/eko/gocache/store/redis/v4"
	"github.com/redis/go-redis/v9"
)

// NewRedisManager builds a Manager backed by the distributed Redis store. Use
// it when cached values must be shared across all running instances.
func NewRedisManager[T any](client *redis.Client) *Manager[T] {
	return &Manager[T]{
		cache: lib_cache.New[T](redisstore.NewRedis(client)),
	}
}
