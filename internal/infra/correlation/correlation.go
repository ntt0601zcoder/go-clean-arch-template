// Package correlation propagates a request-scoped correlation id through
// context.Context (replacing the reference's interface-common correlation_id).
package correlation

import (
	"context"

	"github.com/google/uuid"
)

// HeaderKey is the canonical HTTP/gRPC metadata key for the correlation id.
const HeaderKey = "X-Correlation-ID"

type ctxKey struct{}

// Ensure returns ctx guaranteed to carry a correlation id, plus the id. If one
// is already present it is reused; otherwise a new UUID is generated.
func Ensure(ctx context.Context) (context.Context, string) {
	if id, ok := ctx.Value(ctxKey{}).(string); ok && id != "" {
		return ctx, id
	}
	id := uuid.NewString()
	return context.WithValue(ctx, ctxKey{}, id), id
}

// With stores an explicit id (e.g. taken from an inbound header).
func With(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, id)
}

// Get returns the correlation id, or "" if absent.
func Get(ctx context.Context) string {
	if id, ok := ctx.Value(ctxKey{}).(string); ok {
		return id
	}
	return ""
}
