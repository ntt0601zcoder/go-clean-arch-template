// Package outbound declares the driven ports: interfaces the core depends on
// and adapters implement (repositories, cache, lock, limiter, tx manager).
// These are the only types the use cases know about for I/O.
package outbound

import (
	"context"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
)

// AccountRepository persists accounts. Pick ONE implementation per app and wire
// it in the composition root (internal/apps); the template ships gorm, pgx+sqlc
// and mongo examples and the developer chooses. Implementations MUST honour a
// transaction carried in ctx.
type AccountRepository interface {
	Create(ctx context.Context, e *domain.Account) (string, error)
	GetByID(ctx context.Context, id string) (*domain.Account, error)
	GetByEmail(ctx context.Context, email string) (*domain.Account, error)
	Update(ctx context.Context, p *domain.UpdateAccountParams) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f *domain.ListAccountFilter) ([]domain.Account, int, error)
}
