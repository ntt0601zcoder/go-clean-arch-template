package httphandler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/apperr"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/ginx"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/inbound"
)

// fakeAccountUseCase is a hand-rolled stub of inbound.AccountUseCase whose
// behaviour each test overrides via function fields. It avoids third-party
// mocking/assertion libs to keep this package's go vet/test self-contained.
type fakeAccountUseCase struct {
	create func(ctx context.Context, req *domain.CreateAccountRequest) (*domain.Account, error)
	get    func(ctx context.Context, id string) (*domain.Account, error)
	list   func(ctx context.Context, filter domain.ListAccountFilter) ([]domain.Account, int, error)
	update func(ctx context.Context, req *domain.UpdateAccountRequest) (*domain.Account, error)
	delete func(ctx context.Context, id string) error
}

var _ inbound.AccountUseCase = (*fakeAccountUseCase)(nil)

func (f *fakeAccountUseCase) Create(ctx context.Context, req *domain.CreateAccountRequest) (*domain.Account, error) {
	return f.create(ctx, req)
}

func (f *fakeAccountUseCase) Get(ctx context.Context, id string) (*domain.Account, error) {
	return f.get(ctx, id)
}

func (f *fakeAccountUseCase) List(ctx context.Context, filter domain.ListAccountFilter) ([]domain.Account, int, error) {
	return f.list(ctx, filter)
}

func (f *fakeAccountUseCase) Update(ctx context.Context, req *domain.UpdateAccountRequest) (*domain.Account, error) {
	return f.update(ctx, req)
}

func (f *fakeAccountUseCase) Delete(ctx context.Context, id string) error {
	return f.delete(ctx, id)
}

// newTestRouter builds a gin engine with the error middleware so handler errors
// render the same way they would in production.
func newTestRouter(uc inbound.AccountUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ginx.ErrorMiddleware())
	NewAccountHandler(uc).RegisterRoutes(&r.RouterGroup)
	return r
}

func doJSON(r *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	rdr := bytes.NewBufferString(body)
	req := httptest.NewRequest(method, target, rdr)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreate_Success(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	uc := &fakeAccountUseCase{
		create: func(_ context.Context, req *domain.CreateAccountRequest) (*domain.Account, error) {
			if req.Email != "jane@example.com" {
				t.Fatalf("unexpected email %q", req.Email)
			}
			return &domain.Account{
				ID:        "acc-1",
				Email:     req.Email,
				FirstName: "Jane",
				LastName:  "Doe",
				Status:    domain.AccountStatusActive,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}
	r := newTestRouter(uc)

	body := `{"email":"jane@example.com","password":"s3cretpw","first_name":"Jane","last_name":"Doe","status":1}`
	w := doJSON(r, http.MethodPost, "/accounts", body)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", w.Code, http.StatusCreated, w.Body.String())
	}
	var resp AccountResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "acc-1" || resp.FullName != "Jane Doe" || resp.Status != "active" {
		t.Fatalf("unexpected account: %+v", resp)
	}
}

func TestCreate_ValidationError(t *testing.T) {
	uc := &fakeAccountUseCase{
		create: func(_ context.Context, _ *domain.CreateAccountRequest) (*domain.Account, error) {
			t.Fatal("use case must not be called on a bind error")
			return nil, nil
		},
	}
	r := newTestRouter(uc)

	// Invalid email + missing required fields -> binding fails.
	w := doJSON(r, http.MethodPost, "/accounts", `{"email":"not-an-email"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), apperr.ErrInvalidRequest.Code()) {
		t.Fatalf("body %q missing code %q", w.Body.String(), apperr.ErrInvalidRequest.Code())
	}
}

func TestGet_Success(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	uc := &fakeAccountUseCase{
		get: func(_ context.Context, id string) (*domain.Account, error) {
			if id != "acc-1" {
				t.Fatalf("id = %q, want acc-1", id)
			}
			return &domain.Account{
				ID:        "acc-1",
				Email:     "jane@example.com",
				FirstName: "Jane",
				LastName:  "Doe",
				Status:    domain.AccountStatusActive,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}
	r := newTestRouter(uc)

	w := doJSON(r, http.MethodGet, "/accounts/acc-1", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp AccountResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "acc-1" || resp.FullName != "Jane Doe" || resp.Status != "active" {
		t.Fatalf("unexpected account: %+v", resp)
	}
}

func TestGet_NotFound(t *testing.T) {
	uc := &fakeAccountUseCase{
		get: func(_ context.Context, _ string) (*domain.Account, error) {
			return nil, apperr.ErrAccountNotFound
		},
	}
	r := newTestRouter(uc)

	w := doJSON(r, http.MethodGet, "/accounts/missing", "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	if !strings.Contains(w.Body.String(), apperr.ErrAccountNotFound.Code()) {
		t.Fatalf("body %q missing code %q", w.Body.String(), apperr.ErrAccountNotFound.Code())
	}
}

func TestDelete_Success(t *testing.T) {
	called := false
	uc := &fakeAccountUseCase{
		delete: func(_ context.Context, id string) error {
			called = true
			if id != "acc-1" {
				t.Fatalf("id = %q, want acc-1", id)
			}
			return nil
		},
	}
	r := newTestRouter(uc)

	w := doJSON(r, http.MethodDelete, "/accounts/acc-1", "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if !called {
		t.Fatal("Delete was not called")
	}
}
