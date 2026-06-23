// Package db wires database connections and the per-backend transaction
// managers. Each TxManager is backed by avito-tech/go-transaction-manager v2,
// which stashes the live transaction in the context; the matching provider pulls
// it back out so repositories run the same code inside and outside a tx.
package db

import (
	"context"

	trmgorm "github.com/avito-tech/go-transaction-manager/drivers/gorm/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"gorm.io/gorm"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// GormTxManager implements both outbound.TxManager and outbound.Provider for the
// GORM (PostgreSQL) backend.
type GormTxManager struct {
	db      *gorm.DB
	manager *manager.Manager
	getter  *trmgorm.CtxGetter
}

// NewGormTxManager builds the manager+provider over db. The same *gorm.DB is
// shared with the gorm repository so they use one connection pool.
func NewGormTxManager(db *gorm.DB) *GormTxManager {
	return &GormTxManager{
		db:      db,
		manager: manager.Must(trmgorm.NewDefaultFactory(db)),
		getter:  trmgorm.DefaultCtxGetter,
	}
}

// WithTx runs fn inside a gorm transaction (re-entrant: nested calls reuse it).
func (m *GormTxManager) WithTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	return m.manager.Do(ctx, fn)
}

// GetDB returns the tx-bound *gorm.DB if WithTx is on the stack, else the pool.
func (m *GormTxManager) GetDB(ctx context.Context) *gorm.DB {
	return m.getter.DefaultTrOrDB(ctx, m.db.WithContext(ctx))
}

var (
	_ outbound.TxManager = (*GormTxManager)(nil)
	_ outbound.Provider  = (*GormTxManager)(nil)
)
