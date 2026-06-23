// Package hash hashes passwords with bcrypt — the single algorithm used for all
// password storage in the template.
package hash

import "golang.org/x/crypto/bcrypt"

// Cost is the bcrypt cost factor used for hashing.
const Cost = bcrypt.DefaultCost

// Hash returns the bcrypt hash of plain.
func Hash(plain string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(plain), Cost)
}

// Compare reports whether plain matches the previously hashed value. It returns
// nil on a match and a non-nil error otherwise (bcrypt.ErrMismatchedHashAndPassword
// for a wrong password).
func Compare(hashed []byte, plain string) error {
	return bcrypt.CompareHashAndPassword(hashed, []byte(plain))
}
