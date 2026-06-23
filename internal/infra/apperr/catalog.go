package apperr

import (
	"net/http"

	"google.golang.org/grpc/codes"
)

// Generic sentinels. Treat these as read-only templates: derive request-specific
// errors with .Cause(err) / .WithMetadata(k, v), which return copies.
var (
	ErrInvalidRequest  = New("invalid_request", "the request is invalid", http.StatusBadRequest, codes.InvalidArgument)
	ErrUnauthorized    = New("unauthorized", "authentication required", http.StatusUnauthorized, codes.Unauthenticated)
	ErrForbidden       = New("forbidden", "operation not permitted", http.StatusForbidden, codes.PermissionDenied)
	ErrNotFound        = New("not_found", "resource not found", http.StatusNotFound, codes.NotFound)
	ErrAlreadyExists   = New("already_exists", "resource already exists", http.StatusConflict, codes.AlreadyExists)
	ErrConflict        = New("conflict", "resource conflict", http.StatusConflict, codes.Aborted)
	ErrTooManyRequests = New("too_many_requests", "rate limit exceeded", http.StatusTooManyRequests, codes.ResourceExhausted)
	ErrInternal        = New("internal", "internal server error", http.StatusInternalServerError, codes.Internal)
)

// Account business sentinels.
var (
	ErrAccountNotFound      = New("account_not_found", "account not found", http.StatusNotFound, codes.NotFound)
	ErrAccountAlreadyExists = New("account_already_exists", "account already exists", http.StatusConflict, codes.AlreadyExists)
	ErrInvalidEmail         = New("invalid_email", "invalid email", http.StatusBadRequest, codes.InvalidArgument)
	ErrInvalidStatus        = New("invalid_status", "invalid account status", http.StatusBadRequest, codes.InvalidArgument)
	ErrWeakPassword         = New("weak_password", "password does not meet strength requirements", http.StatusBadRequest, codes.InvalidArgument)
	ErrInvalidCredentials   = New("invalid_credentials", "invalid credentials", http.StatusUnauthorized, codes.Unauthenticated)
	ErrAccountBlocked       = New("account_blocked", "account is blocked", http.StatusForbidden, codes.PermissionDenied)
)
