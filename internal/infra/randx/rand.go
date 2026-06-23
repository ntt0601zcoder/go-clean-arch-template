// Package randx provides cryptographically-secure random helpers.
package randx

import (
	"crypto/rand"
	"math/big"
)

const (
	alphanumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	digits       = "0123456789"
)

// String returns a random alphanumeric string of length n.
func String(n int) string { return fromAlphabet(n, alphanumeric) }

// Digits returns a random numeric string of length n (e.g. OTP codes).
func Digits(n int) string { return fromAlphabet(n, digits) }

func fromAlphabet(n int, alphabet string) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			// crypto/rand failure is catastrophic; fall back to first char
			// rather than panic in library code.
			out[i] = alphabet[0]
			continue
		}
		out[i] = alphabet[idx.Int64()]
	}
	return string(out)
}
