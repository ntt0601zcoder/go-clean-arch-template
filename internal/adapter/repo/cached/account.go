// Package cachedrepo provides a cache-aside decorator for the account
// repository. It wraps any outbound.AccountRepository, serving reads from a
// CacheManager and invalidating affected keys after writes, so callers get
// caching transparently without the underlying backend (gorm, pgx, mongo)
// knowing anything about it.
package cachedrepo

import (
	"context"
	"log/slog"
	"time"

	lib_store "github.com/eko/gocache/lib/v4/store"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// AccountRepository decorates an inner AccountRepository with a read-through
// cache. Reads populate the cache on miss; writes delegate to the inner repo
// and then best-effort invalidate the cached id key. Cache failures never break
// a write — they are logged and swallowed so persistence stays authoritative.
type AccountRepository struct {
	inner outbound.AccountRepository
	cache outbound.CacheManager[*domain.Account]
	ttl   time.Duration
	log   *slog.Logger
}

// NewAccountRepository wires the decorator to an inner repository and a typed
// cache, using ttl as the per-entry expiration for cached reads.
func NewAccountRepository(
	inner outbound.AccountRepository,
	cache outbound.CacheManager[*domain.Account],
	ttl time.Duration,
	log *slog.Logger,
) *AccountRepository {
	return &AccountRepository{inner: inner, cache: cache, ttl: ttl, log: log}
}

var _ outbound.AccountRepository = (*AccountRepository)(nil)

// GetByID serves the account from cache, loading it via the inner repo on miss.
func (r *AccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	return r.cache.GetOnce(ctx, domain.AccountCacheKeyByID(id),
		func(ctx context.Context) (*domain.Account, error) {
			return r.inner.GetByID(ctx, id)
		},
		lib_store.WithExpiration(r.ttl),
	)
}

// GetByEmail serves the account from cache, loading it via the inner repo on miss.
func (r *AccountRepository) GetByEmail(ctx context.Context, email string) (*domain.Account, error) {
	return r.cache.GetOnce(ctx, domain.AccountCacheKeyByEmail(email),
		func(ctx context.Context) (*domain.Account, error) {
			return r.inner.GetByEmail(ctx, email)
		},
		lib_store.WithExpiration(r.ttl),
	)
}

// Create delegates to the inner repo. A freshly created account is not yet
// cached, so nothing needs invalidating.
func (r *AccountRepository) Create(ctx context.Context, e *domain.Account) (string, error) {
	return r.inner.Create(ctx, e)
}

// Update persists the change and then evicts the cached id entry so the next
// read reloads the fresh row.
func (r *AccountRepository) Update(ctx context.Context, p *domain.UpdateAccountParams) error {
	if err := r.inner.Update(ctx, p); err != nil {
		return err
	}
	r.invalidate(ctx, domain.AccountCacheKeyByID(p.ID))
	return nil
}

// Delete removes the account and then evicts its cached id entry.
func (r *AccountRepository) Delete(ctx context.Context, id string) error {
	if err := r.inner.Delete(ctx, id); err != nil {
		return err
	}
	r.invalidate(ctx, domain.AccountCacheKeyByID(id))
	return nil
}

// List is not cached; it passes straight through to the inner repository.
func (r *AccountRepository) List(ctx context.Context, f *domain.ListAccountFilter) ([]domain.Account, int, error) {
	return r.inner.List(ctx, f)
}

// invalidate evicts a cache key best-effort: a failure is logged at warn and
// otherwise ignored so it cannot turn a successful write into a failed one.
func (r *AccountRepository) invalidate(ctx context.Context, key string) {
	if err := r.cache.Delete(ctx, key); err != nil {
		r.log.WarnContext(ctx, "cached account repo: failed to invalidate cache key",
			slog.String("key", key), slog.Any("error", err))
	}
}
