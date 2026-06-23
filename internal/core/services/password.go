package services

import (
	"context"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/hash"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/inbound"
)

// PasswordService implements inbound.PasswordService. It enforces the password
// strength rule and hashes with bcrypt (the single algorithm), so use cases
// never touch a crypto library directly.
type PasswordService struct{}

var _ inbound.PasswordService = (*PasswordService)(nil)

// NewPasswordService builds the password service.
func NewPasswordService() *PasswordService {
	return &PasswordService{}
}

// IsValidPassword applies the domain strength rule (minimum length).
func (s *PasswordService) IsValidPassword(_ context.Context, password string) (bool, error) {
	return len(password) >= domain.MinPasswordLength, nil
}

// Hash hashes password with bcrypt, used for all new writes.
func (s *PasswordService) Hash(_ context.Context, password string) ([]byte, error) {
	return hash.Hash(password)
}
