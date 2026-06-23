// Package account holds the AccountUseCase: basic CRUD for the account entity.
// It is core code: it depends only on ports, the domain and infra/apperr — never
// on transports or persistence drivers. The concrete repository/tx-manager are
// chosen in the composition root; the use case only sees the interfaces.
package account

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/apperr"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/inbound"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// AccountUseCase implements basic account CRUD against a single repository +
// transaction manager, applying password policy and a business rate limit on
// creation.
type AccountUseCase struct {
	repo      outbound.AccountRepository
	tx        outbound.TxManager
	limiter   outbound.RateLimiter
	passwords inbound.PasswordService
	log       *slog.Logger
}

// NewAccountUseCase wires the use case with its ports.
func NewAccountUseCase(
	repo outbound.AccountRepository,
	tx outbound.TxManager,
	limiter outbound.RateLimiter,
	passwords inbound.PasswordService,
	log *slog.Logger,
) *AccountUseCase {
	return &AccountUseCase{
		repo:      repo,
		tx:        tx,
		limiter:   limiter,
		passwords: passwords,
		log:       log,
	}
}

var _ inbound.AccountUseCase = (*AccountUseCase)(nil)

// Create validates, hashes and persists a new account inside a transaction that
// first guards against a duplicate email. It is rate limited per email to show a
// business-level limit.
func (uc *AccountUseCase) Create(ctx context.Context, req *domain.CreateAccountRequest) (*domain.Account, error) {
	verdict, err := uc.limiter.Limit(ctx, &domain.LimitRequest{
		Action: domain.RateLimitActionCreateAccount,
		Key:    strings.ToLower(req.Email),
	})
	if err != nil {
		return nil, apperr.ErrInternal.Cause(err)
	}
	if !verdict.Allowed {
		return nil, apperr.ErrTooManyRequests.WithMetadata("retry_after", verdict.RetryAfter.String())
	}

	if err := validateEmail(req.Email); err != nil {
		return nil, err
	}
	if err := uc.ensureStrongPassword(ctx, req.Password); err != nil {
		return nil, err
	}

	hashed, err := uc.passwords.Hash(ctx, req.Password)
	if err != nil {
		return nil, apperr.ErrInternal.Cause(err)
	}

	status := req.Status
	if !status.Valid() {
		status = domain.AccountStatusActive
	}

	now := time.Now().UTC()
	account := &domain.Account{
		ID:           uuid.NewString(),
		Email:        req.Email,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		PasswordHash: string(hashed),
		Status:       status,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.tx.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.ensureEmailFree(txCtx, account.Email); err != nil {
			return err
		}
		id, err := uc.repo.Create(txCtx, account)
		if err != nil {
			return err
		}
		account.ID = id
		return nil
	}); err != nil {
		return nil, err
	}

	uc.log.InfoContext(ctx, "account created", slog.String("account_id", account.ID))
	return account, nil
}

// Get fetches a single account by id.
func (uc *AccountUseCase) Get(ctx context.Context, id string) (*domain.Account, error) {
	if id == "" {
		return nil, apperr.ErrInvalidRequest.WithMessage("account id is required")
	}
	return uc.repo.GetByID(ctx, id)
}

// List returns accounts matching the filter and the total match count.
func (uc *AccountUseCase) List(ctx context.Context, filter domain.ListAccountFilter) ([]domain.Account, int, error) {
	return uc.repo.List(ctx, &filter)
}

// Update applies a partial profile update inside a transaction and returns the
// refreshed account.
func (uc *AccountUseCase) Update(ctx context.Context, req *domain.UpdateAccountRequest) (*domain.Account, error) {
	if req.ID == "" {
		return nil, apperr.ErrInvalidRequest.WithMessage("account id is required")
	}
	if req.Status != nil && !req.Status.Valid() {
		return nil, apperr.ErrInvalidStatus
	}

	var updated *domain.Account
	if err := uc.tx.WithTx(ctx, func(txCtx context.Context) error {
		params := &domain.UpdateAccountParams{
			ID:        req.ID,
			FirstName: req.FirstName,
			LastName:  req.LastName,
			Status:    req.Status,
			UpdatedAt: time.Now().UTC(),
		}
		if err := uc.repo.Update(txCtx, params); err != nil {
			return err
		}
		account, err := uc.repo.GetByID(txCtx, req.ID)
		if err != nil {
			return err
		}
		updated = account
		return nil
	}); err != nil {
		return nil, err
	}
	return updated, nil
}

// Delete removes an account inside a transaction.
func (uc *AccountUseCase) Delete(ctx context.Context, id string) error {
	if id == "" {
		return apperr.ErrInvalidRequest.WithMessage("account id is required")
	}
	if err := uc.tx.WithTx(ctx, func(txCtx context.Context) error {
		return uc.repo.Delete(txCtx, id)
	}); err != nil {
		return err
	}
	uc.log.InfoContext(ctx, "account deleted", slog.String("account_id", id))
	return nil
}

// ensureStrongPassword runs the password policy, mapping a weak result to
// ErrWeakPassword and a policy backend failure to ErrInternal.
func (uc *AccountUseCase) ensureStrongPassword(ctx context.Context, password string) error {
	ok, err := uc.passwords.IsValidPassword(ctx, password)
	if err != nil {
		return apperr.ErrInternal.Cause(err)
	}
	if !ok {
		return apperr.ErrWeakPassword
	}
	return nil
}

// ensureEmailFree returns ErrAccountAlreadyExists when an account with email
// already exists; a not-found result means the email is free.
func (uc *AccountUseCase) ensureEmailFree(ctx context.Context, email string) error {
	_, err := uc.repo.GetByEmail(ctx, email)
	switch {
	case err == nil:
		return apperr.ErrAccountAlreadyExists.WithMetadata("email", email)
	case errors.Is(err, apperr.ErrAccountNotFound):
		return nil
	default:
		return err
	}
}

// validateEmail performs the minimal structural check the use case owns.
func validateEmail(email string) error {
	if email == "" || !strings.Contains(email, "@") {
		return apperr.ErrInvalidEmail.WithMetadata("email", email)
	}
	return nil
}
