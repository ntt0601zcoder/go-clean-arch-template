package mongorepo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/apperr"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// collectionName is the Mongo collection holding account documents.
const collectionName = "accounts"

// AccountRepository persists accounts in MongoDB. The active transaction
// session (if any) is injected into ctx by infra/db.MongoTxManager.WithTx, so
// every method simply forwards ctx to the collection and automatically joins
// the surrounding transaction.
type AccountRepository struct {
	coll *mongo.Collection
}

// NewAccountRepository binds the repository to the "accounts" collection of the
// given database.
func NewAccountRepository(database *mongo.Database) *AccountRepository {
	return &AccountRepository{coll: database.Collection(collectionName)}
}

// Create inserts a new account and returns its id. A duplicate key (unique
// email/phone index) maps to apperr.ErrAccountAlreadyExists.
func (r *AccountRepository) Create(ctx context.Context, e *domain.Account) (string, error) {
	doc := toDoc(e)
	if _, err := r.coll.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return "", apperr.ErrAccountAlreadyExists.Cause(err)
		}
		return "", apperr.ErrInternal.Cause(err)
	}
	return doc.ID, nil
}

// GetByID fetches one account by its id.
func (r *AccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

// GetByEmail fetches one account by email.
func (r *AccountRepository) GetByEmail(ctx context.Context, email string) (*domain.Account, error) {
	return r.findOne(ctx, bson.M{"email": email})
}

// findOne runs a single-document lookup and maps a missing document to
// apperr.ErrAccountNotFound.
func (r *AccountRepository) findOne(ctx context.Context, filter bson.M) (*domain.Account, error) {
	var doc accountDoc
	if err := r.coll.FindOne(ctx, filter).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperr.ErrAccountNotFound.Cause(err)
		}
		return nil, apperr.ErrInternal.Cause(err)
	}
	return doc.toDomain(), nil
}

// Update applies a partial $set of only the provided (non-nil) fields. A zero
// matched count means the account does not exist.
func (r *AccountRepository) Update(ctx context.Context, p *domain.UpdateAccountParams) error {
	set := bson.M{"updated_at": p.UpdatedAt}
	if p.FirstName != nil {
		set["first_name"] = *p.FirstName
	}
	if p.LastName != nil {
		set["last_name"] = *p.LastName
	}
	if p.Status != nil {
		set["status"] = int(*p.Status)
	}

	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": p.ID}, bson.M{"$set": set})
	if err != nil {
		return apperr.ErrInternal.Cause(err)
	}
	if res.MatchedCount == 0 {
		return apperr.ErrAccountNotFound
	}
	return nil
}

// Delete removes an account by id. A zero deleted count means it was absent.
func (r *AccountRepository) Delete(ctx context.Context, id string) error {
	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return apperr.ErrInternal.Cause(err)
	}
	if res.DeletedCount == 0 {
		return apperr.ErrAccountNotFound
	}
	return nil
}

// List returns a page of accounts (newest first) plus the total count matching
// the filter. The status constraint is optional.
func (r *AccountRepository) List(ctx context.Context, f *domain.ListAccountFilter) ([]domain.Account, int, error) {
	filter := bson.M{}
	if f.Status != nil {
		filter["status"] = int(*f.Status)
	}

	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, apperr.ErrInternal.Cause(err)
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if f.Limit > 0 {
		opts.SetLimit(int64(f.Limit))
	}
	if f.Offset > 0 {
		opts.SetSkip(int64(f.Offset))
	}

	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, apperr.ErrInternal.Cause(err)
	}
	defer cur.Close(ctx)

	var docs []accountDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, 0, apperr.ErrInternal.Cause(err)
	}

	accounts := make([]domain.Account, 0, len(docs))
	for _, d := range docs {
		accounts = append(accounts, *d.toDomain())
	}
	return accounts, int(total), nil
}

// EnsureIndexes creates the unique email index and a status index. It is safe to
// call repeatedly and is intended to run once at startup.
func (r *AccountRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_email"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_status"),
		},
	}
	if _, err := r.coll.Indexes().CreateMany(ctx, models); err != nil {
		return apperr.ErrInternal.Cause(err)
	}
	return nil
}

var _ outbound.AccountRepository = (*AccountRepository)(nil)
