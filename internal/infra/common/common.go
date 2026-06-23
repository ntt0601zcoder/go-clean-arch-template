// Package common holds tiny, dependency-free helpers shared across layers.
package common

// SetIfNotNil assigns *src to *dst when src is non-nil. Handy for applying
// optional (pointer) update fields onto an entity.
func SetIfNotNil[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}

// Ptr returns a pointer to v. Useful for building optional fields in tests.
func Ptr[T any](v T) *T { return &v }

// Deref returns *p, or the zero value of T when p is nil.
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
