// Package mongorepo is the MongoDB-backed implementation of the account
// repository driven port. It uses mongo-driver v1 and relies on the active
// transaction session being carried in ctx by infra/db.MongoTxManager, so all
// collection operations simply pass ctx through.
package mongorepo

import (
	"time"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
)

// accountDoc is the BSON document shape for an account. It mirrors
// domain.Account but uses snake_case bson tags and stores the domain id as the
// Mongo _id so lookups by id stay primary-key fast.
type accountDoc struct {
	ID           string     `bson:"_id"`
	Email        string     `bson:"email"`
	FirstName    string     `bson:"first_name"`
	LastName     string     `bson:"last_name"`
	PasswordHash string     `bson:"password_hash"`
	Status       int        `bson:"status"`
	CreatedAt    time.Time  `bson:"created_at"`
	UpdatedAt    time.Time  `bson:"updated_at"`
	DeletedAt    *time.Time `bson:"deleted_at,omitempty"`
}

// toDoc converts a domain entity into its persistence document.
func toDoc(a *domain.Account) accountDoc {
	return accountDoc{
		ID:           a.ID,
		Email:        a.Email,
		FirstName:    a.FirstName,
		LastName:     a.LastName,
		PasswordHash: a.PasswordHash,
		Status:       int(a.Status),
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
		DeletedAt:    a.DeletedAt,
	}
}

// toDomain rehydrates a domain entity from its persistence document.
func (d accountDoc) toDomain() *domain.Account {
	return &domain.Account{
		ID:           d.ID,
		Email:        d.Email,
		FirstName:    d.FirstName,
		LastName:     d.LastName,
		PasswordHash: d.PasswordHash,
		Status:       domain.AccountStatus(d.Status),
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
		DeletedAt:    d.DeletedAt,
	}
}
