package db

import (
	"context"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// PgxTxManager implements outbound.TxManager for the pgx + sqlc backend and also
// exposes Conn so the sqlc repository can build its Queries against the active
// transaction or the pool.
type PgxTxManager struct {
	pool    *pgxpool.Pool
	manager *manager.Manager
	getter  *trmpgx.CtxGetter
}

// NewPgxTxManager builds the manager+conn provider over pool.
func NewPgxTxManager(pool *pgxpool.Pool) *PgxTxManager {
	return &PgxTxManager{
		pool:    pool,
		manager: manager.Must(trmpgx.NewDefaultFactory(pool)),
		getter:  trmpgx.DefaultCtxGetter,
	}
}

// WithTx runs fn inside a pgx transaction.
func (m *PgxTxManager) WithTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	return m.manager.Do(ctx, fn)
}

// Conn returns the in-flight pgx transaction if WithTx is on the stack, else the
// pool. The result satisfies sqlcgen.DBTX (Exec/Query/QueryRow).
func (m *PgxTxManager) Conn(ctx context.Context) trmpgx.Tr {
	return m.getter.DefaultTrOrDB(ctx, m.pool)
}

var _ outbound.TxManager = (*PgxTxManager)(nil)
