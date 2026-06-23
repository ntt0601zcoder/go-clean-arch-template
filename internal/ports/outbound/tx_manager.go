package outbound

import (
	"context"

	"gorm.io/gorm"
)

// Provider hands a repository the *gorm.DB bound to the current context: the
// in-flight transaction if TxManager.WithTx is on the stack, otherwise the pool.
// It is gorm-specific by design; the pgx and mongo backends use their own
// concrete provider types in internal/infra/db (a backend's repo lives in the
// adapter layer and may import infra directly).
type Provider interface {
	GetDB(ctx context.Context) *gorm.DB
}

// TxManager runs fn inside a single transaction, propagating the tx via the
// context passed to fn. This is the ONLY transaction abstraction the core sees,
// so it works the same whichever backend is wired (gorm/pgx/mongo).
type TxManager interface {
	WithTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
