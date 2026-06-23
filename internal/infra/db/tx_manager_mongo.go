package db

import (
	"context"

	trmmongo "github.com/avito-tech/go-transaction-manager/drivers/mongo/v2"
	trmcontext "github.com/avito-tech/go-transaction-manager/trm/v2/context"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// MongoTxManager implements outbound.TxManager for MongoDB. The avito factory
// opens a session, starts a transaction and injects the session into the context
// it passes to fn (via mongo.NewSessionContext), so the mongo repository simply
// runs its collection operations with that context — they automatically join the
// transaction.
//
// NOTE: multi-document transactions require a replica set (or mongos). The
// docker-compose runs mongo as a single-node replica set (rs0).
type MongoTxManager struct {
	db      *mongo.Database
	manager *manager.Manager
}

// NewMongoTxManager builds the manager over the client behind db.
func NewMongoTxManager(db *mongo.Database) *MongoTxManager {
	return &MongoTxManager{
		db: db,
		manager: manager.Must(
			trmmongo.NewDefaultFactory(db.Client()),
			manager.WithCtxManager(trmcontext.DefaultManager),
		),
	}
}

// WithTx runs fn inside a mongo transaction.
func (m *MongoTxManager) WithTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	return m.manager.Do(ctx, fn)
}

// Database returns the underlying database (the repo derives collections from it;
// the active session, if any, travels in ctx).
func (m *MongoTxManager) Database() *mongo.Database { return m.db }

var _ outbound.TxManager = (*MongoTxManager)(nil)
