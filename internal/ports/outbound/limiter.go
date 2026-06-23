package outbound

import (
	"context"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
)

// RateLimiter enforces business-level rate limits ("max N registrations per
// minute per IP"). It returns the verdict in the response; an error means the
// backend itself failed (callers may choose to fail open).
type RateLimiter interface {
	Limit(ctx context.Context, req *domain.LimitRequest) (*domain.LimitResponse, error)
}
