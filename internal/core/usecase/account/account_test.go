package account

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/apperr"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/hash"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/inbound"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// wantAppErr asserts err is an AppError whose code matches want.Code().
func wantAppErr(t *testing.T, err error, want *apperr.AppError) {
	t.Helper()
	ae, ok := apperr.FromError(err)
	if !ok {
		t.Fatalf("want AppError %q, got %v", want.Code(), err)
	}
	if ae.Code() != want.Code() {
		t.Fatalf("want code %q, got %q (%v)", want.Code(), ae.Code(), err)
	}
}

// --- hand-written fakes (no mockgen) ---

// fakeAccountRepo is an in-memory AccountRepository keyed by id and email.
type fakeAccountRepo struct {
	byID    map[string]*domain.Account
	byEmail map[string]*domain.Account
}

func newFakeAccountRepo() *fakeAccountRepo {
	return &fakeAccountRepo{
		byID:    map[string]*domain.Account{},
		byEmail: map[string]*domain.Account{},
	}
}

func (r *fakeAccountRepo) Create(_ context.Context, e *domain.Account) (string, error) {
	if _, ok := r.byEmail[e.Email]; ok {
		return "", apperr.ErrAccountAlreadyExists
	}
	cp := *e
	r.byID[cp.ID] = &cp
	r.byEmail[cp.Email] = &cp
	return cp.ID, nil
}

func (r *fakeAccountRepo) GetByID(_ context.Context, id string) (*domain.Account, error) {
	a, ok := r.byID[id]
	if !ok {
		return nil, apperr.ErrAccountNotFound
	}
	cp := *a
	return &cp, nil
}

func (r *fakeAccountRepo) GetByEmail(_ context.Context, email string) (*domain.Account, error) {
	a, ok := r.byEmail[email]
	if !ok {
		return nil, apperr.ErrAccountNotFound
	}
	cp := *a
	return &cp, nil
}

func (r *fakeAccountRepo) Update(_ context.Context, p *domain.UpdateAccountParams) error {
	a, ok := r.byID[p.ID]
	if !ok {
		return apperr.ErrAccountNotFound
	}
	if p.FirstName != nil {
		a.FirstName = *p.FirstName
	}
	if p.LastName != nil {
		a.LastName = *p.LastName
	}
	if p.Status != nil {
		a.Status = *p.Status
	}
	a.UpdatedAt = p.UpdatedAt
	return nil
}

func (r *fakeAccountRepo) Delete(_ context.Context, id string) error {
	a, ok := r.byID[id]
	if !ok {
		return apperr.ErrAccountNotFound
	}
	delete(r.byID, id)
	delete(r.byEmail, a.Email)
	return nil
}

func (r *fakeAccountRepo) List(_ context.Context, _ *domain.ListAccountFilter) ([]domain.Account, int, error) {
	out := make([]domain.Account, 0, len(r.byID))
	for _, a := range r.byID {
		out = append(out, *a)
	}
	return out, len(out), nil
}

// passthroughTxManager runs fn with the same context (no real transaction).
type passthroughTxManager struct{}

func (passthroughTxManager) WithTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}

// fakeLimiter returns a fixed verdict.
type fakeLimiter struct{ allowed bool }

func (l fakeLimiter) Limit(_ context.Context, _ *domain.LimitRequest) (*domain.LimitResponse, error) {
	return &domain.LimitResponse{Allowed: l.allowed}, nil
}

// realPasswordService is a minimal real PasswordService backed by infra/hash.
type realPasswordService struct{}

func (realPasswordService) IsValidPassword(_ context.Context, password string) (bool, error) {
	return len(password) >= 8, nil
}
func (realPasswordService) Hash(_ context.Context, password string) ([]byte, error) {
	return hash.Hash(password)
}

// compile-time conformance of the test doubles.
var (
	_ outbound.AccountRepository = (*fakeAccountRepo)(nil)
	_ outbound.TxManager         = passthroughTxManager{}
	_ outbound.RateLimiter       = fakeLimiter{}
	_ inbound.PasswordService    = realPasswordService{}
)

func newTestUseCase(repo outbound.AccountRepository, allowed bool) *AccountUseCase {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewAccountUseCase(repo, passthroughTxManager{}, fakeLimiter{allowed: allowed}, realPasswordService{}, log)
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name      string
		seedEmail string
		allowed   bool
		req       *domain.CreateAccountRequest
		wantErr   *apperr.AppError
	}{
		{
			name:    "success",
			allowed: true,
			req:     &domain.CreateAccountRequest{Email: "alice@example.com", Password: "supersecret"},
		},
		{
			name:      "duplicate email",
			allowed:   true,
			seedEmail: "bob@example.com",
			req:       &domain.CreateAccountRequest{Email: "bob@example.com", Password: "supersecret"},
			wantErr:   apperr.ErrAccountAlreadyExists,
		},
		{
			name:    "weak password",
			allowed: true,
			req:     &domain.CreateAccountRequest{Email: "carol@example.com", Password: "short"},
			wantErr: apperr.ErrWeakPassword,
		},
		{
			name:    "invalid email",
			allowed: true,
			req:     &domain.CreateAccountRequest{Email: "no-at-sign", Password: "supersecret"},
			wantErr: apperr.ErrInvalidEmail,
		},
		{
			name:    "rate limited",
			allowed: false,
			req:     &domain.CreateAccountRequest{Email: "dave@example.com", Password: "supersecret"},
			wantErr: apperr.ErrTooManyRequests,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeAccountRepo()
			if tc.seedEmail != "" {
				if _, err := repo.Create(context.Background(), &domain.Account{ID: "seed", Email: tc.seedEmail}); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			uc := newTestUseCase(repo, tc.allowed)

			account, err := uc.Create(context.Background(), tc.req)

			if tc.wantErr != nil {
				wantAppErr(t, err, tc.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if account.ID == "" {
				t.Error("expected generated id")
			}
			if account.Status != domain.AccountStatusActive {
				t.Errorf("expected active status, got %v", account.Status)
			}
			if account.PasswordHash == tc.req.Password {
				t.Error("password must be hashed")
			}
		})
	}
}

func TestGet(t *testing.T) {
	repo := newFakeAccountRepo()
	if _, err := repo.Create(context.Background(), &domain.Account{ID: "id-1", Email: "e@e.com"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	uc := newTestUseCase(repo, true)

	t.Run("found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), "id-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "id-1" {
			t.Errorf("got id %q", got.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := uc.Get(context.Background(), "missing")
		if !errors.Is(err, apperr.ErrAccountNotFound) {
			t.Fatalf("want ErrAccountNotFound, got %v", err)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		_, err := uc.Get(context.Background(), "")
		wantAppErr(t, err, apperr.ErrInvalidRequest)
	})
}

func TestUpdate(t *testing.T) {
	repo := newFakeAccountRepo()
	if _, err := repo.Create(context.Background(), &domain.Account{ID: "id-1", Email: "e@e.com", FirstName: "Old"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	uc := newTestUseCase(repo, true)

	newName := "New"
	got, err := uc.Update(context.Background(), &domain.UpdateAccountRequest{ID: "id-1", FirstName: &newName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.FirstName != "New" {
		t.Errorf("first name = %q, want New", got.FirstName)
	}
}

func TestDelete(t *testing.T) {
	repo := newFakeAccountRepo()
	if _, err := repo.Create(context.Background(), &domain.Account{ID: "id-1", Email: "e@e.com"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	uc := newTestUseCase(repo, true)

	if err := uc.Delete(context.Background(), "id-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := uc.Get(context.Background(), "id-1"); !errors.Is(err, apperr.ErrAccountNotFound) {
		t.Fatalf("want ErrAccountNotFound after delete, got %v", err)
	}
}
