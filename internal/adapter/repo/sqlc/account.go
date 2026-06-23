// Package sqlcrepo implements outbound.AccountRepository over PostgreSQL using
// pgx and sqlc-generated queries. It maps pgx driver errors to apperr sentinels
// and honours the ambient transaction carried in ctx via the PgxTxManager.
package sqlcrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	sqlcgen "github.com/ntt0601zcoder/go-clean-arch-template/internal/adapter/repo/sqlc/gen"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/apperr"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/db"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// uniqueViolationCode is the PostgreSQL SQLSTATE for a unique_violation.
const uniqueViolationCode = "23505"

// AccountRepository is the pgx+sqlc backed account store. It builds its Queries
// per call against the connection resolved from the transaction manager, so it
// transparently participates in any in-flight transaction.
type AccountRepository struct {
	tx *db.PgxTxManager
}

// NewAccountRepository wires the repository to the pgx transaction manager.
func NewAccountRepository(tx *db.PgxTxManager) *AccountRepository {
	return &AccountRepository{tx: tx}
}

// queries returns a Queries bound to the active transaction or pool for ctx.
func (r *AccountRepository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(r.tx.Conn(ctx))
}

// Create inserts a new account and returns its id. A unique-constraint
// violation is surfaced as apperr.ErrAccountAlreadyExists.
func (r *AccountRepository) Create(ctx context.Context, e *domain.Account) (string, error) {
	if err := r.queries(ctx).CreateAccount(ctx, toCreateParams(e)); err != nil {
		return "", mapWriteErr(err)
	}
	return e.ID, nil
}

// GetByID returns the account with id, or apperr.ErrAccountNotFound.
func (r *AccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	row, err := r.queries(ctx).GetAccountByID(ctx, id)
	if err != nil {
		return nil, mapReadErr(err)
	}
	return toDomain(row), nil
}

// GetByEmail returns the account with email, or apperr.ErrAccountNotFound.
func (r *AccountRepository) GetByEmail(ctx context.Context, email string) (*domain.Account, error) {
	row, err := r.queries(ctx).GetAccountByEmail(ctx, email)
	if err != nil {
		return nil, mapReadErr(err)
	}
	return toDomain(row), nil
}

// Update applies a partial profile update. Zero affected rows means the account
// does not exist, surfaced as apperr.ErrAccountNotFound.
func (r *AccountRepository) Update(ctx context.Context, p *domain.UpdateAccountParams) error {
	rows, err := r.queries(ctx).UpdateAccount(ctx, toUpdateParams(p))
	if err != nil {
		return mapWriteErr(err)
	}
	if rows == 0 {
		return apperr.ErrAccountNotFound
	}
	return nil
}

// Delete removes an account by id. Zero affected rows means it was not present.
func (r *AccountRepository) Delete(ctx context.Context, id string) error {
	rows, err := r.queries(ctx).DeleteAccount(ctx, id)
	if err != nil {
		return apperr.ErrInternal.Cause(fmt.Errorf("delete account: %w", err))
	}
	if rows == 0 {
		return apperr.ErrAccountNotFound
	}
	return nil
}

// List returns a filtered page of accounts together with the total count
// matching the same filter (ignoring limit/offset).
func (r *AccountRepository) List(ctx context.Context, f *domain.ListAccountFilter) ([]domain.Account, int, error) {
	q := r.queries(ctx)
	status := statusFilter(f.Status)

	rows, err := q.ListAccounts(ctx, sqlcgen.ListAccountsParams{
		Status:    status,
		RowLimit:  int32(f.Limit),
		RowOffset: int32(f.Offset),
	})
	if err != nil {
		return nil, 0, apperr.ErrInternal.Cause(fmt.Errorf("list accounts: %w", err))
	}

	total, err := q.CountAccounts(ctx, status)
	if err != nil {
		return nil, 0, apperr.ErrInternal.Cause(fmt.Errorf("count accounts: %w", err))
	}

	accounts := make([]domain.Account, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, *toDomain(row))
	}
	return accounts, int(total), nil
}

// mapReadErr maps a single-row read error: no rows => ErrAccountNotFound,
// anything else => wrapped ErrInternal.
func mapReadErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.ErrAccountNotFound.Cause(err)
	}
	return apperr.ErrInternal.Cause(err)
}

// mapWriteErr maps a write error: unique violation => ErrAccountAlreadyExists,
// anything else => wrapped ErrInternal.
func mapWriteErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
		return apperr.ErrAccountAlreadyExists.Cause(err)
	}
	return apperr.ErrInternal.Cause(err)
}

var _ outbound.AccountRepository = (*AccountRepository)(nil)
