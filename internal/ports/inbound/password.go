package inbound

import "context"

// PasswordService encapsulates password policy and hashing so use cases never
// touch a hashing library directly.
type PasswordService interface {
	IsValidPassword(ctx context.Context, password string) (bool, error)
	Hash(ctx context.Context, password string) ([]byte, error)
}
