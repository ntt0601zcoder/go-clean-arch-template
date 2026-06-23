// Package limiter provides a Redis-backed rate limiter that enforces the
// per-action policies declared in the domain (domain.RateLimitDefaults).
package limiter

import (
	"context"
	"fmt"

	redis_rate "github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// defaultRate is applied to actions without an explicit policy, so an unknown
// action still gets a sane ceiling rather than being unbounded.
const defaultRate = 60

// Limiter enforces business rate limits using a sliding-window algorithm backed
// by Redis. Policies are resolved once at construction from the domain defaults.
type Limiter struct {
	rl     *redis_rate.Limiter
	limits map[domain.RateLimitAction]redis_rate.Limit
}

// NewRedisLimiter builds a limiter over the given Redis client, translating the
// domain rate-limit defaults into the underlying library's policy type.
func NewRedisLimiter(client *redis.Client) *Limiter {
	limits := make(map[domain.RateLimitAction]redis_rate.Limit, len(domain.RateLimitDefaults))
	for action, cfg := range domain.RateLimitDefaults {
		limits[action] = redis_rate.Limit{
			Rate:   cfg.Rate,
			Burst:  cfg.Burst,
			Period: cfg.Period,
		}
	}
	return &Limiter{
		rl:     redis_rate.NewLimiter(client),
		limits: limits,
	}
}

// Limit decides whether one unit of req.Action for req.Key may proceed. An error
// signals a backend failure (callers may fail open); the verdict otherwise rides
// in the response.
func (l *Limiter) Limit(ctx context.Context, req *domain.LimitRequest) (*domain.LimitResponse, error) {
	limit, ok := l.limits[req.Action]
	if !ok {
		limit = redis_rate.PerMinute(defaultRate)
	}

	key := string(req.Action) + ":" + req.Key
	res, err := l.rl.Allow(ctx, key, limit)
	if err != nil {
		return nil, fmt.Errorf("limiter: allow %q: %w", key, err)
	}

	return &domain.LimitResponse{
		Allowed:    res.Allowed > 0,
		Remaining:  res.Remaining,
		RetryAfter: res.RetryAfter,
	}, nil
}

var _ outbound.RateLimiter = (*Limiter)(nil)
