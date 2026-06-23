package sqlcrepo

import (
	"github.com/jackc/pgx/v5/pgtype"

	sqlcgen "github.com/ntt0601zcoder/go-clean-arch-template/internal/adapter/repo/sqlc/gen"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
)

// toDomain converts a sqlc row into the core entity, flattening pgtype wrappers
// and the nullable deleted_at column back into plain Go values.
func toDomain(row sqlcgen.Account) *domain.Account {
	a := &domain.Account{
		ID:           row.ID,
		Email:        row.Email,
		FirstName:    row.FirstName,
		LastName:     row.LastName,
		PasswordHash: row.PasswordHash,
		Status:       domain.AccountStatus(row.Status),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
	if row.DeletedAt.Valid {
		t := row.DeletedAt.Time
		a.DeletedAt = &t
	}
	return a
}

// toCreateParams builds the insert params from the core entity, wrapping the
// timestamps in pgtype.Timestamptz so pgx encodes them correctly.
func toCreateParams(e *domain.Account) sqlcgen.CreateAccountParams {
	return sqlcgen.CreateAccountParams{
		ID:           e.ID,
		Email:        e.Email,
		FirstName:    e.FirstName,
		LastName:     e.LastName,
		PasswordHash: e.PasswordHash,
		Status:       int32(e.Status),
		CreatedAt:    pgtype.Timestamptz{Time: e.CreatedAt, Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: e.UpdatedAt, Valid: true},
	}
}

// toUpdateParams builds the partial-update params. A nil pointer leaves the
// column untouched (the query COALESCEs against the existing value).
func toUpdateParams(p *domain.UpdateAccountParams) sqlcgen.UpdateAccountParams {
	out := sqlcgen.UpdateAccountParams{
		ID:        p.ID,
		FirstName: p.FirstName,
		LastName:  p.LastName,
		UpdatedAt: pgtype.Timestamptz{Time: p.UpdatedAt, Valid: true},
	}
	if p.Status != nil {
		s := int32(*p.Status)
		out.Status = &s
	}
	return out
}

// statusFilter converts the optional domain status filter into the *int32 the
// generated query expects (nil => no constraint).
func statusFilter(s *domain.AccountStatus) *int32 {
	if s == nil {
		return nil
	}
	v := int32(*s)
	return &v
}
