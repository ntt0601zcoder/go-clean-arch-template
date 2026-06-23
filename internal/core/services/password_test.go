package services

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
)

func newPasswordService() *PasswordService {
	return NewPasswordService()
}

func TestPasswordService_IsValidPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{name: "too short", password: strings.Repeat("a", domain.MinPasswordLength-1), want: false},
		{name: "exactly min length", password: strings.Repeat("a", domain.MinPasswordLength), want: true},
		{name: "longer than min", password: strings.Repeat("a", domain.MinPasswordLength+5), want: true},
		{name: "empty", password: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newPasswordService()
			got, err := svc.IsValidPassword(context.Background(), tt.password)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPasswordService_Hash(t *testing.T) {
	svc := newPasswordService()
	hashed, err := svc.Hash(context.Background(), "correct-horse")
	require.NoError(t, err)
	assert.NotEmpty(t, hashed)
	assert.NotEqual(t, "correct-horse", string(hashed))
}
