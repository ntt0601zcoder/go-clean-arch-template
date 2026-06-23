// Package grpchandler is the gRPC adapter for the account use case. It owns the
// translation between the generated protobuf wire types and the transport-agnostic
// domain types, so the core never sees gRPC types. Error mapping (AppError ->
// status code) is handled by the grpcx interceptor, so handlers return errors
// from the use case unchanged.
package grpchandler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	accountv1 "github.com/ntt0601zcoder/go-clean-arch-template/internal/gen/account/v1"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/inbound"
)

// AccountHandler implements accountv1.AccountServiceServer by delegating to the
// inbound AccountUseCase. The Unimplemented embed keeps it forward-compatible
// when new RPCs are added to the proto contract.
type AccountHandler struct {
	accountv1.UnimplementedAccountServiceServer
	uc inbound.AccountUseCase
}

// NewAccountHandler wires the handler to its use case.
func NewAccountHandler(uc inbound.AccountUseCase) *AccountHandler {
	return &AccountHandler{uc: uc}
}

// RegisterServer binds this handler onto a gRPC server.
func (h *AccountHandler) RegisterServer(s *grpc.Server) {
	accountv1.RegisterAccountServiceServer(s, h)
}

// CreateAccount creates an account.
func (h *AccountHandler) CreateAccount(ctx context.Context, req *accountv1.CreateAccountRequest) (*accountv1.CreateAccountResponse, error) {
	account, err := h.uc.Create(ctx, &domain.CreateAccountRequest{
		Email:     req.GetEmail(),
		Password:  req.GetPassword(),
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
	})
	if err != nil {
		return nil, err
	}
	return &accountv1.CreateAccountResponse{Id: account.ID}, nil
}

// GetAccount fetches a single account by id.
func (h *AccountHandler) GetAccount(ctx context.Context, req *accountv1.GetAccountRequest) (*accountv1.GetAccountResponse, error) {
	account, err := h.uc.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &accountv1.GetAccountResponse{Account: toProtoAccount(account)}, nil
}

// ListAccounts returns a page of accounts plus the total matching the filter.
func (h *AccountHandler) ListAccounts(ctx context.Context, req *accountv1.ListAccountsRequest) (*accountv1.ListAccountsResponse, error) {
	filter := domain.ListAccountFilter{
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	}
	if req.Status != nil {
		status := toDomainStatus(*req.Status)
		filter.Status = &status
	}
	accounts, total, err := h.uc.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]*accountv1.Account, 0, len(accounts))
	for i := range accounts {
		out = append(out, toProtoAccount(&accounts[i]))
	}
	return &accountv1.ListAccountsResponse{Accounts: out, Total: int32(total)}, nil
}

// UpdateAccount edits mutable profile fields. Absent optional fields are left
// unchanged, mirroring the nil-pointer semantics of the domain request.
func (h *AccountHandler) UpdateAccount(ctx context.Context, req *accountv1.UpdateAccountRequest) (*accountv1.UpdateAccountResponse, error) {
	domReq := &domain.UpdateAccountRequest{
		ID:        req.GetId(),
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}
	if req.Status != nil {
		status := toDomainStatus(*req.Status)
		domReq.Status = &status
	}
	account, err := h.uc.Update(ctx, domReq)
	if err != nil {
		return nil, err
	}
	return &accountv1.UpdateAccountResponse{Account: toProtoAccount(account)}, nil
}

// DeleteAccount removes an account by id.
func (h *AccountHandler) DeleteAccount(ctx context.Context, req *accountv1.DeleteAccountRequest) (*accountv1.DeleteAccountResponse, error) {
	if err := h.uc.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &accountv1.DeleteAccountResponse{}, nil
}

// toProtoAccount maps a domain account onto its wire representation. The password
// hash is intentionally never exposed on the wire.
func toProtoAccount(a *domain.Account) *accountv1.Account {
	if a == nil {
		return nil
	}
	return &accountv1.Account{
		Id:        a.ID,
		Email:     a.Email,
		FirstName: a.FirstName,
		LastName:  a.LastName,
		Status:    toProtoStatus(a.Status),
		CreatedAt: timestamppb.New(a.CreatedAt),
		UpdatedAt: timestamppb.New(a.UpdatedAt),
	}
}

// toProtoStatus maps the domain status enum to the proto enum (values align 1:1).
func toProtoStatus(s domain.AccountStatus) accountv1.AccountStatus {
	return accountv1.AccountStatus(s)
}

// toDomainStatus maps the proto status enum back to the domain enum.
func toDomainStatus(s accountv1.AccountStatus) domain.AccountStatus {
	return domain.AccountStatus(s)
}

var _ accountv1.AccountServiceServer = (*AccountHandler)(nil)
