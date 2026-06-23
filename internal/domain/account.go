// Package domain is the innermost layer: entities, value objects, constants and
// the request/response DTOs the use cases speak. It imports only the standard
// library — no frameworks, no transport, no persistence.
package domain

import (
	"fmt"
	"strings"
	"time"
)

// Account is the core entity. PasswordHash is always stored hashed; plaintext
// never lives on the entity.
type Account struct {
	ID           string
	Email        string
	FirstName    string
	LastName     string
	PasswordHash string
	Status       AccountStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

// FullName is a tiny piece of domain behaviour kept next to the entity.
func (a Account) FullName() string {
	return strings.TrimSpace(a.FirstName + " " + a.LastName)
}

// IsActive reports whether the account may sign in / be served.
func (a Account) IsActive() bool { return a.Status == AccountStatusActive }

// --- cache key patterns (single source of truth for derived keys) ---

const (
	accountCacheByIDKey    = "account:id:%s"
	accountCacheByEmailKey = "account:email:%s"
)

// AccountCacheKeyByID returns the cache key for an account fetched by id.
func AccountCacheKeyByID(id string) string { return fmt.Sprintf(accountCacheByIDKey, id) }

// AccountCacheKeyByEmail returns the cache key for an account fetched by email.
func AccountCacheKeyByEmail(email string) string {
	return fmt.Sprintf(accountCacheByEmailKey, strings.ToLower(email))
}

// CacheKeys returns every cache key derived from this account, for invalidation.
func (a Account) CacheKeys() []string {
	keys := []string{AccountCacheKeyByID(a.ID)}
	if a.Email != "" {
		keys = append(keys, AccountCacheKeyByEmail(a.Email))
	}
	return keys
}

// --- Request DTOs (use-case boundary objects) ---

// CreateAccountRequest creates an account. Status is optional (defaults active).
type CreateAccountRequest struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
	Status    AccountStatus
}

// UpdateAccountRequest edits mutable profile fields. nil pointer => unchanged.
type UpdateAccountRequest struct {
	ID        string
	FirstName *string
	LastName  *string
	Status    *AccountStatus
}

// UpdateAccountParams is the persistence-level partial update. nil => unchanged.
type UpdateAccountParams struct {
	ID        string
	FirstName *string
	LastName  *string
	Status    *AccountStatus
	UpdatedAt time.Time
}

// ListAccountFilter narrows a List query. Zero values mean "no constraint".
type ListAccountFilter struct {
	Status *AccountStatus
	Limit  int
	Offset int
}
