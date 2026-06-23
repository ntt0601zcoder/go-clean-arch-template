package domain

import "time"

// RateLimitAction names a business action subject to rate limiting. Limits are
// expressed per action so policy lives in the domain, not in transport.
type RateLimitAction string

const (
	RateLimitActionCreateAccount RateLimitAction = "create_account"
)

// RateLimitConfig is a token-bucket-ish policy: Rate events per Period (with an
// optional Burst on top).
type RateLimitConfig struct {
	Rate   int
	Period time.Duration
	Burst  int
}

// RateLimitDefaults are the built-in limits, overridable from config.
var RateLimitDefaults = map[RateLimitAction]RateLimitConfig{
	RateLimitActionCreateAccount: {Rate: 5, Period: time.Minute, Burst: 5},
}

// LimitRequest asks the limiter whether one unit of Action for Key may proceed.
type LimitRequest struct {
	Action RateLimitAction
	Key    string // discriminator: account id, email, ip, ...
}

// LimitResponse is the limiter verdict.
type LimitResponse struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}
