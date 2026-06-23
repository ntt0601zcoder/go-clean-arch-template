// Package httphandler is the Gin REST adapter for the account use case. It owns
// the wire DTOs (with json + binding tags and swaggo doc comments) and maps them
// to/from the transport-agnostic domain types, so the core never sees HTTP types.
package httphandler

import (
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
)

// AccountResponse is the public JSON shape of an account. The password hash is
// intentionally never serialised.
type AccountResponse struct {
	ID        string `json:"id" example:"3f2504e0-4f89-11d3-9a0c-0305e82c3301"`
	Email     string `json:"email" example:"jane@example.com"`
	FirstName string `json:"first_name" example:"Jane"`
	LastName  string `json:"last_name" example:"Doe"`
	FullName  string `json:"full_name" example:"Jane Doe"`
	Status    string `json:"status" example:"active"`
	CreatedAt string `json:"created_at" example:"2026-06-22T10:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2026-06-22T10:00:00Z"`
}

// CreateAccountRequest is the create command (POST /accounts).
type CreateAccountRequest struct {
	Email     string `json:"email" binding:"required,email" example:"jane@example.com"`
	Password  string `json:"password" binding:"required,min=8" example:"s3cretpw"`
	FirstName string `json:"first_name" example:"Jane"`
	LastName  string `json:"last_name" example:"Doe"`
	// Status is the initial lifecycle state (1=active, 2=inactive, 3=blocked).
	Status int `json:"status" example:"1"`
}

// UpdateAccountRequest edits mutable profile fields. Omitted (nil) fields are
// left unchanged; pointers distinguish "absent" from "set to zero value".
type UpdateAccountRequest struct {
	FirstName *string `json:"first_name,omitempty" example:"Jane"`
	LastName  *string `json:"last_name,omitempty" example:"Doe"`
	// Status optionally moves the account to a new lifecycle state.
	Status *int `json:"status,omitempty" example:"2"`
}

// ListAccountsResponse wraps a page of accounts plus the total match count.
type ListAccountsResponse struct {
	Accounts []AccountResponse `json:"accounts"`
	Total    int               `json:"total"`
}

// --- mappers: wire -> domain ---

func (r CreateAccountRequest) toDomain() *domain.CreateAccountRequest {
	return &domain.CreateAccountRequest{
		Email:     r.Email,
		Password:  r.Password,
		FirstName: r.FirstName,
		LastName:  r.LastName,
		Status:    domain.AccountStatus(r.Status),
	}
}

func (r UpdateAccountRequest) toDomain(id string) *domain.UpdateAccountRequest {
	out := &domain.UpdateAccountRequest{
		ID:        id,
		FirstName: r.FirstName,
		LastName:  r.LastName,
	}
	if r.Status != nil {
		st := domain.AccountStatus(*r.Status)
		out.Status = &st
	}
	return out
}

// --- mappers: domain -> wire ---

// fromDomain projects a domain.Account onto the public response shape.
func fromDomain(a *domain.Account) AccountResponse {
	if a == nil {
		return AccountResponse{}
	}
	return AccountResponse{
		ID:        a.ID,
		Email:     a.Email,
		FirstName: a.FirstName,
		LastName:  a.LastName,
		FullName:  a.FullName(),
		Status:    a.Status.String(),
		CreatedAt: a.CreatedAt.Format(timeFormat),
		UpdatedAt: a.UpdatedAt.Format(timeFormat),
	}
}

func fromDomainList(accounts []domain.Account) []AccountResponse {
	out := make([]AccountResponse, 0, len(accounts))
	for i := range accounts {
		out = append(out, fromDomain(&accounts[i]))
	}
	return out
}
