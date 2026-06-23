// Package apperr is the single in-repo replacement for the reference's external
// corerror/bizerror/errors packages. An AppError carries a stable code, a
// safe public message, the transport mappings (HTTP status + gRPC code), an
// optional wrapped cause and free-form metadata. Handlers map any error through
// FromError and render Code/Message/HTTPStatus/GRPCCode — internal causes are
// never leaked to callers.
package apperr

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
)

// AppError is an immutable-by-convention application error. Cause and
// WithMetadata return COPIES so shared sentinels in catalog.go are never
// mutated by callers.
type AppError struct {
	code       string
	message    string
	httpStatus int
	grpcCode   codes.Code
	cause      error
	meta       map[string]string
}

// New constructs a sentinel/base AppError.
func New(code, message string, httpStatus int, grpcCode codes.Code) *AppError {
	return &AppError{code: code, message: message, httpStatus: httpStatus, grpcCode: grpcCode}
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.code, e.message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.code, e.message)
}

// Unwrap exposes the wrapped cause to errors.Is/As.
func (e *AppError) Unwrap() error { return e.cause }

// Is lets errors.Is match by stable code rather than pointer identity, so an
// error derived from a sentinel via Cause/WithMetadata/WithMessage (which return
// copies) still matches the original sentinel:
//
//	errors.Is(apperr.ErrAccountNotFound.Cause(driverErr), apperr.ErrAccountNotFound) // true
func (e *AppError) Is(target error) bool {
	var t *AppError
	if errors.As(target, &t) {
		return e.code == t.code
	}
	return false
}

// Code returns the stable machine code.
func (e *AppError) Code() string { return e.code }

// Message returns the safe, caller-facing message.
func (e *AppError) Message() string { return e.message }

// HTTPStatus returns the HTTP status to render.
func (e *AppError) HTTPStatus() int { return e.httpStatus }

// GRPCCode returns the gRPC status code to render.
func (e *AppError) GRPCCode() codes.Code { return e.grpcCode }

// Metadata returns a copy of the attached metadata.
func (e *AppError) Metadata() map[string]string {
	out := make(map[string]string, len(e.meta))
	for k, v := range e.meta {
		out[k] = v
	}
	return out
}

func (e *AppError) clone() *AppError {
	cp := *e
	cp.meta = e.Metadata()
	return &cp
}

// Cause returns a copy wrapping err as the underlying cause.
func (e *AppError) Cause(err error) *AppError {
	c := e.clone()
	c.cause = err
	return c
}

// WithMessage returns a copy with a different public message.
func (e *AppError) WithMessage(msg string) *AppError {
	c := e.clone()
	c.message = msg
	return c
}

// WithMetadata returns a copy with an extra metadata key/value.
func (e *AppError) WithMetadata(key, value string) *AppError {
	c := e.clone()
	if c.meta == nil {
		c.meta = map[string]string{}
	}
	c.meta[key] = value
	return c
}

// FromError extracts an *AppError from err (via errors.As). It returns
// (ErrInternal.Cause(err), false) when err is not an AppError, so callers always
// get something renderable.
func FromError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return ErrInternal.Cause(err), false
}
