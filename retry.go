package pco

import (
	"os"
	"strconv"
	"sync"
	"time"
)

// This file implements go-pco's opt-in 429 retry: on a rate-limit
// response, wait and try again, up to a bounded number of attempts. It's
// a safety net for exactly what the throttle (throttle.go) can't fully
// close on its own - a token bucket bounds *sustained* rate, but a
// same-token burst that lands right as the bucket refills (drained at the
// start of one window, refilled by the start of the next) can still
// legitimately clear the throttle's own admission check and still get a
//429 from PCO. Off by default, same "zero added behavior until opted in"
// convention as the throttle - a consumer that never calls ConfigureRetry
// and never sets a PCO_RETRY_ON_429_* env var gets exactly today's
// behavior: one attempt, whatever error PCO returned goes straight back
// to the caller.
//
// Deliberately narrow: only a 429 is ever retried. A 4xx other than 429 is
// a request the caller built wrong (bad params, a resource that doesn't
// exist) - retrying it wastes a call and gets the same error again. A 5xx
// might be transient, but retrying every 5xx blind is a different,
// broader feature (idempotency of writes matters there in a way it
// doesn't for "wait for a rate limit to clear") that nothing has asked
// for yet - don't add it speculatively.

// RetryConfig configures the 429 retry. Pass it to ConfigureRetry to opt
// in explicitly; a zero-value field is treated as "unset" and replaced
// with the package default for that field (see the const block below).
type RetryConfig struct {
	// Enabled opts into the retry. False (the zero value) means "off,"
	// same as ThrottleConfig.Enabled - no separate "unset" state.
	Enabled bool

	// MaxAttempts is the total number of tries for one logical call,
	// including the first - 2 (the default) means "try once, and on a
	// 429, retry exactly once more." Non-positive falls back to the
	// package default.
	MaxAttempts int

	// MaxWait caps how long a single retry will sleep, even if PCO's
	// Retry-After header asks for longer - protects a caller with its own
	// deadline (a user-facing HTTP handler mid-render) from an unbounded
	// wait. Non-positive falls back to the package default (30s).
	MaxWait time.Duration

	// FallbackWait is used when the 429 response has no Retry-After
	// header at all - undocumented by PCO, but RequestError.RetryAfter is
	// already "empty when absent" rather than assumed-present, so this
	// needs its own default. Non-positive falls back to the package
	// default (5s).
	FallbackWait time.Duration
}

const (
	defaultRetryMaxAttempts   = 2
	defaultRetryMaxWait       = 30 * time.Second
	defaultRetryFallbackWait  = 5 * time.Second
	retryEnvMaxAttemptsVar    = "PCO_RETRY_ON_429_MAX_ATTEMPTS"
	retryEnvMaxWaitSecondsVar = "PCO_RETRY_ON_429_MAX_WAIT_SECONDS"
)

// applyRetryDefaults returns cfg with every non-positive numeric field
// replaced by its package default. Enabled is passed through unchanged,
// mirroring applyThrottleDefaults.
func applyRetryDefaults(cfg RetryConfig) RetryConfig {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = defaultRetryMaxAttempts
	}
	if cfg.MaxWait <= 0 {
		cfg.MaxWait = defaultRetryMaxWait
	}
	if cfg.FallbackWait <= 0 {
		cfg.FallbackWait = defaultRetryFallbackWait
	}
	return cfg
}

var (
	retryMu       sync.RWMutex
	retryCfg      RetryConfig
	retryExplicit bool

	retryEnvOnce sync.Once
)

// ConfigureRetry opts into (or explicitly configures) the 429 retry - see
// resolveThrottleConfig's doc comment for the identical precedence rule
// this mirrors: an explicit call always wins over the PCO_RETRY_ON_429_*
// environment variables below, even if a request already resolved the env
// vars before this was called.
func ConfigureRetry(cfg RetryConfig) {
	cfg = applyRetryDefaults(cfg)

	retryMu.Lock()
	retryCfg = cfg
	retryExplicit = true
	retryMu.Unlock()
}

// resolveRetryConfig returns the retry's current configuration: whatever
// ConfigureRetry was last called with, or - if it was never called - the
// environment-derived configuration (resolved once, lazily, the first
// time this is reached with no explicit config in place). Identical shape
// to resolveThrottleConfig; kept as its own copy rather than shared
// generic code since the two configs' field sets differ and this stays
// simpler to read than a generic version would.
func resolveRetryConfig() RetryConfig {
	retryMu.RLock()
	explicit := retryExplicit
	cfg := retryCfg
	retryMu.RUnlock()

	if explicit {
		return cfg
	}

	retryEnvOnce.Do(func() {
		envCfg := RetryConfig{Enabled: envBoolEnabled(os.Getenv("PCO_RETRY_ON_429_ENABLED"))}
		if v := os.Getenv(retryEnvMaxAttemptsVar); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				envCfg.MaxAttempts = n
			}
		}
		if v := os.Getenv(retryEnvMaxWaitSecondsVar); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				envCfg.MaxWait = time.Duration(n) * time.Second
			}
		}
		envCfg = applyRetryDefaults(envCfg)

		retryMu.Lock()
		// Only adopt the env-derived config if ConfigureRetry hasn't won
		// the race in the meantime - an explicit call always wins.
		if !retryExplicit {
			retryCfg = envCfg
		}
		retryMu.Unlock()
	})

	retryMu.RLock()
	defer retryMu.RUnlock()
	return retryCfg
}

// retryWaitFor picks how long NewRequest should sleep before retrying a
// 429, given that response's Retry-After header (verbatim, possibly
// empty - see RequestError.RetryAfter) and the resolved RetryConfig's
// bounds. PCO always sends a delay-in-seconds value, never an HTTP-date
// (same assumption RequestError.RetryAfter's own doc comment makes) - an
// empty or unparsable header falls back to cfg.FallbackWait. Either way,
// the result is never allowed above cfg.MaxWait, regardless of what PCO
// asked for.
func retryWaitFor(retryAfter string, cfg RetryConfig) time.Duration {
	wait := cfg.FallbackWait
	if secs, err := strconv.Atoi(retryAfter); err == nil && secs >= 0 {
		wait = time.Duration(secs) * time.Second
	}
	if wait > cfg.MaxWait {
		wait = cfg.MaxWait
	}
	return wait
}
