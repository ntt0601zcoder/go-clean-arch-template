package gormrepo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/apperr"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// AccountRepository is the GORM-backed AccountRepository. It holds no *gorm.DB
// of its own; every method asks the Provider for the DB bound to the current
// context so writes participate in an in-flight TxManager.WithTx transaction.
type AccountRepository struct {
	provider outbound.Provider
}

// NewAccountRepository wires the repository to a gorm Provider (typically the
// infra/db.GormTxManager, which is both Provider and TxManager).
func NewAccountRepository(provider outbound.Provider) *AccountRepository {
	return &AccountRepository{provider: provider}
}

var _ outbound.AccountRepository = (*AccountRepository)(nil)

// Create inserts a new account and returns its id. A unique-constraint
// violation (e.g. duplicate email) is mapped to ErrAccountAlreadyExists.
func (r *AccountRepository) Create(ctx context.Context, e *domain.Account) (string, error) {
	db := r.provider.GetDB(ctx)
	m := toModel(e)
	if err := db.WithContext(ctx).Create(m).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return "", apperr.ErrAccountAlreadyExists.Cause(err)
		}
		return "", apperr.ErrInternal.Cause(err)
	}
	return m.ID, nil
}

// GetByID loads a single account by primary key.
func (r *AccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	return r.getOne(ctx, "id = ?", id)
}

// GetByEmail loads a single account by its unique email.
func (r *AccountRepository) GetByEmail(ctx context.Context, email string) (*domain.Account, error) {
	return r.getOne(ctx, "email = ?", email)
}

// getOne centralises the single-row fetch and the not-found mapping shared by
// the GetBy* methods.
func (r *AccountRepository) getOne(ctx context.Context, query string, args ...any) (*domain.Account, error) {
	db := r.provider.GetDB(ctx)
	var m AccountModel
	if err := db.WithContext(ctx).Where(query, args...).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.ErrAccountNotFound.Cause(err)
		}
		return nil, apperr.ErrInternal.Cause(err)
	}
	return m.toDomain(), nil
}

// Update applies a partial update: only the non-nil pointer fields of p are
// written, alongside updated_at. A zero RowsAffected means the row did not
// exist, surfaced as ErrAccountNotFound.
func (r *AccountRepository) Update(ctx context.Context, p *domain.UpdateAccountParams) error {
	db := r.provider.GetDB(ctx)

	updates := map[string]any{"updated_at": p.UpdatedAt}
	if p.FirstName != nil {
		updates["first_name"] = *p.FirstName
	}
	if p.LastName != nil {
		updates["last_name"] = *p.LastName
	}
	if p.Status != nil {
		updates["status"] = int(*p.Status)
	}

	res := db.WithContext(ctx).Model(&AccountModel{}).Where("id = ?", p.ID).Updates(updates)
	if err := res.Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperr.ErrAccountAlreadyExists.Cause(err)
		}
		return apperr.ErrInternal.Cause(err)
	}
	if res.RowsAffected == 0 {
		return apperr.ErrAccountNotFound
	}
	return nil
}

// Delete removes an account by id. A zero RowsAffected is treated as not found.
func (r *AccountRepository) Delete(ctx context.Context, id string) error {
	db := r.provider.GetDB(ctx)
	res := db.WithContext(ctx).Where("id = ?", id).Delete(&AccountModel{})
	if err := res.Error; err != nil {
		return apperr.ErrInternal.Cause(err)
	}
	if res.RowsAffected == 0 {
		return apperr.ErrAccountNotFound
	}
	return nil
}

// List returns a page of accounts matching the filter plus the total count of
// rows matching the same status constraint (ignoring limit/offset).
func (r *AccountRepository) List(ctx context.Context, f *domain.ListAccountFilter) ([]domain.Account, int, error) {
	db := r.provider.GetDB(ctx)

	// A fresh Session per query keeps the Count clause from leaking into the
	// subsequent Find (and vice versa) since both branch from the same base.
	base := db.WithContext(ctx).Model(&AccountModel{})
	if f != nil && f.Status != nil {
		base = base.Where("status = ?", int(*f.Status))
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, apperr.ErrInternal.Cause(err)
	}

	query := base.Session(&gorm.Session{})
	if f != nil {
		if f.Limit > 0 {
			query = query.Limit(f.Limit)
		}
		if f.Offset > 0 {
			query = query.Offset(f.Offset)
		}
	}

	var rows []AccountModel
	if err := query.Find(&rows).Error; err != nil {
		return nil, 0, apperr.ErrInternal.Cause(err)
	}

	accounts := make([]domain.Account, 0, len(rows))
	for _, m := range rows {
		accounts = append(accounts, *m.toDomain())
	}
	return accounts, int(total), nil
}
