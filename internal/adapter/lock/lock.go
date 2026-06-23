// Package lock provides a Redis-backed distributed locker. Locks are stored as
// keys carrying a per-acquisition random token so that release only frees a
// lock still owned by the caller, never one a later holder has taken over.
package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// lockKeyPrefix namespaces lock keys away from other data sharing the Redis db.
const lockKeyPrefix = "lock:"

// releaseScript atomically deletes the lock key only when its value still
// matches the holder's token, making release safe and idempotent: a key already
// gone (expired or freed) simply yields 0.
var releaseScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
else
	return 0
end
`)

// RedisLocker grants mutually-exclusive named locks via Redis SET NX.
type RedisLocker struct {
	client *redis.Client
}

// NewRedisLocker builds a locker over the given Redis client.
func NewRedisLocker(client *redis.Client) *RedisLocker {
	return &RedisLocker{client: client}
}

// Acquire takes the lock for key with the given TTL. On success it returns a
// release func that frees only this acquisition; it returns
// outbound.ErrLockNotAcquired when the lock is already held elsewhere.
func (l *RedisLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (func(ctx context.Context) error, error) {
	token, err := newToken()
	if err != nil {
		return nil, fmt.Errorf("lock: generate token: %w", err)
	}

	redisKey := lockKeyPrefix + key
	ok, err := l.client.SetNX(ctx, redisKey, token, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("lock: setnx %q: %w", redisKey, err)
	}
	if !ok {
		return nil, outbound.ErrLockNotAcquired
	}

	release := func(ctx context.Context) error {
		if err := releaseScript.Run(ctx, l.client, []string{redisKey}, token).Err(); err != nil {
			return fmt.Errorf("lock: release %q: %w", redisKey, err)
		}
		return nil
	}
	return release, nil
}

// newToken returns a cryptographically random opaque token identifying a single
// lock acquisition.
func newToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

var _ outbound.DistributedLocker = (*RedisLocker)(nil)
