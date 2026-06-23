package domain

// AccountStatus is the lifecycle state of an account.
type AccountStatus int

const (
	AccountStatusUnknown  AccountStatus = iota // 0
	AccountStatusActive                        // 1
	AccountStatusInactive                      // 2
	AccountStatusBlocked                       // 3
	AccountStatusDeleted                       // 4
)

// Valid reports whether s is a known, assignable status.
func (s AccountStatus) Valid() bool {
	switch s {
	case AccountStatusActive, AccountStatusInactive, AccountStatusBlocked, AccountStatusDeleted:
		return true
	default:
		return false
	}
}

// String renders the status for logs/JSON.
func (s AccountStatus) String() string {
	switch s {
	case AccountStatusActive:
		return "active"
	case AccountStatusInactive:
		return "inactive"
	case AccountStatusBlocked:
		return "blocked"
	case AccountStatusDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

// HashAlgorithm identifies a password hashing scheme. bcrypt is the only one
// allowed for new writes; others (if added) are read-only/legacy.
type HashAlgorithm string

const (
	HashAlgorithmBcrypt HashAlgorithm = "bcrypt"
)

// MinPasswordLength is the domain rule for password strength.
const MinPasswordLength = 8
