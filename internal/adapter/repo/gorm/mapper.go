package gormrepo

import "github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"

// toModel converts a domain entity into its GORM row, encoding the status enum
// as its underlying int.
func toModel(e *domain.Account) *AccountModel {
	if e == nil {
		return nil
	}
	return &AccountModel{
		ID:           e.ID,
		Email:        e.Email,
		FirstName:    e.FirstName,
		LastName:     e.LastName,
		PasswordHash: e.PasswordHash,
		Status:       int(e.Status),
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
		DeletedAt:    e.DeletedAt,
	}
}

// toDomain rebuilds the domain entity from a row, decoding the int status back
// into the AccountStatus enum.
func (m AccountModel) toDomain() *domain.Account {
	return &domain.Account{
		ID:           m.ID,
		Email:        m.Email,
		FirstName:    m.FirstName,
		LastName:     m.LastName,
		PasswordHash: m.PasswordHash,
		Status:       domain.AccountStatus(m.Status),
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		DeletedAt:    m.DeletedAt,
	}
}
