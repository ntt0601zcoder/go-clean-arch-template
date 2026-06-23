// Package httpx maps application errors onto safe HTTP responses. Internal
// causes are never serialised — only the AppError code/message/metadata.
package httpx

import (
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/apperr"
)

// ErrorResponse is the JSON body returned for errors.
type ErrorResponse struct {
	Code     string            `json:"code"`
	Message  string            `json:"message"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// HTTPError pairs a status with a renderable body.
type HTTPError struct {
	Status int
	Body   ErrorResponse
}

// NewError converts any error to an HTTPError via apperr.FromError. A nil error
// is treated as an internal error (defensive: callers reach here only on a
// failure path).
func NewError(err error) HTTPError {
	if err == nil {
		err = apperr.ErrInternal
	}
	ae, _ := apperr.FromError(err)
	body := ErrorResponse{Code: ae.Code(), Message: ae.Message()}
	if meta := ae.Metadata(); len(meta) > 0 {
		body.Metadata = meta
	}
	return HTTPError{Status: ae.HTTPStatus(), Body: body}
}
