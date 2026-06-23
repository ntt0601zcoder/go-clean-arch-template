// Package inbound declares the driving ports: use-case interfaces that
// transports (HTTP, gRPC, worker) call. The core implements them.
package inbound

import (
	"context"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
)

// AccountUseCase is the account application API: plain CRUD consumed by the HTTP
// and gRPC handlers.
type AccountUseCase interface {
	Create(ctx context.Context, req *domain.CreateAccountRequest) (*domain.Account, error)
	Get(ctx context.Context, id string) (*domain.Account, error)
	List(ctx context.Context, filter domain.ListAccountFilter) ([]domain.Account, int, error)
	Update(ctx context.Context, req *domain.UpdateAccountRequest) (*domain.Account, error)
	Delete(ctx context.Context, id string) error
}
