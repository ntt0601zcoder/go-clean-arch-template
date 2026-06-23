// Package validatorx wraps go-playground/validator with the app's error type so
// validation failures surface as apperr.ErrInvalidRequest with field metadata.
package validatorx

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/apperr"
)

// Validator validates structs tagged with `validate:"..."`.
type Validator struct {
	v *validator.Validate
}

// New builds a Validator using JSON field names in error messages.
func New() *Validator {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return &Validator{v: v}
}

// ValidateStruct returns nil or an apperr.ErrInvalidRequest describing the first
// failing field.
func (val *Validator) ValidateStruct(s any) error {
	if err := val.v.Struct(s); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) && len(verrs) > 0 {
			fe := verrs[0]
			return apperr.ErrInvalidRequest.
				WithMessage("validation failed on field '"+fe.Field()+"' ("+fe.Tag()+")").
				WithMetadata("field", fe.Field()).
				WithMetadata("rule", fe.Tag()).
				Cause(err)
		}
		return apperr.ErrInvalidRequest.Cause(err)
	}
	return nil
}
