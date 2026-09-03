package pco

import (
	"sync"
	"time"
)

// This file implements go-pco's opt-in response hook: a way for a consumer
// to observe every NewRequest call (method, URL, outcome, timing) without
// touching every individual call site - useful for logging a 429 or
// tracking latency across the ~40 resource functions this package exposes,
// none of which funnel through anything else a consumer could wrap. Off by
// default (nil hook), same "zero added behavior until opted in" convention
// as the throttle in throttle.go.

// ResponseInfo describes one completed NewRequest call, success or
// failure, handed to whatever func SetResponseHook registered.
type ResponseInfo struct {
	Method string
	URL    string

	// Err is the error NewRequest is about to return - nil on success. A
	// non-2xx response is a *RequestError (use errors.As to inspect
	// StatusCode/RetryAfter/Errors); anything else (a network failure, a
	// context cancellation, a JSON decode failure) is whatever error
	// produced it.
	Err error

	// StatusCode is the response's HTTP status when one was received (any
	// 2xx or the non-2xx status carried by Err), 0 when the request never
	// got a response at all (network error, timeout, context cancelled
	// before or during the call).
	StatusCode int

	// RetryAfter mirrors RequestError.RetryAfter for convenience - empty
	// unless StatusCode is 429 and PCO sent the header. Duplicated here
	// (rather than making the caller type-assert Err) since a 429 is
	// exactly the case this hook exists to make easy to log.
	RetryAfter string

	// Attempt is this call's attempt number, starting at 1. Always 1
	// unless the opt-in 429 retry (see retry.go) is enabled and actually
	// retried - a hook that doesn't care about retries can ignore this
	// field entirely.
	Attempt int

	// ThrottleWait is how long this attempt sat in the throttle's
	// admission queue before its HTTP call started - zero when the
	// throttle is disabled, or when it admitted the call immediately.
	ThrottleWait time.Duration

	// Duration is this attempt's own wall-clock cost: ThrottleWait plus
	// the HTTP round trip. It does not include time spent on an earlier,
	// retried attempt - sum Duration across every ResponseInfo sharing the
	// same logical call (same Method/URL, increasing Attempt) for that.
	Duration time.Duration
}

var (
	responseHookMu sync.RWMutex
	responseHook   func(ResponseInfo)
)

// SetResponseHook registers fn to be called synchronously, after every
// NewRequest attempt completes (success or failure, including a retried
// attempt - see retry.go), with details about that attempt. Pass nil to
// clear a previously-registered hook.
//
// fn runs on the same goroutine as the request that triggered it, in the
// brief window between the HTTP response finishing and NewRequest
// returning - keep it fast (a log call, a metrics increment), never a
// blocking call of its own, since it directly adds to every request's
// latency.
//
// Call this once at process startup, the same process-lifetime convention
// ConfigureThrottle already uses - it's a plain package-level var, not
// safe to swap concurrently with the intent of catching every in-flight
// request's completion consistently (a request already past the read
// below still fires under whichever hook was set when it started).
//
// This is purely observational: fn's return value (there isn't one) can't
// affect whether a call succeeds, retries, or what NewRequest returns to
// its caller - see ConfigureRetry (retry.go) if you want to change
// behavior on a 429, not just observe it.
func SetResponseHook(fn func(ResponseInfo)) {
	responseHookMu.Lock()
	responseHook = fn
	responseHookMu.Unlock()
}

// callResponseHook invokes the currently-registered hook, if any. A no-op
// (one RLock/RUnlock, no allocation) when SetResponseHook was never
// called - cheap enough to call unconditionally from NewRequest rather
// than making every call site check first.
func callResponseHook(info ResponseInfo) {
	responseHookMu.RLock()
	fn := responseHook
	responseHookMu.RUnlock()
	if fn != nil {
		fn(info)
	}
}

// buildResponseInfo assembles a ResponseInfo from one NewRequest attempt's
// raw outcome - factored out of NewRequest so it's unit-testable without
// driving a full request.
func buildResponseInfo(method, url string, err error, attempt int, throttleWait, duration time.Duration) ResponseInfo {
	info := ResponseInfo{
		Method:       method,
		URL:          url,
		Err:          err,
		Attempt:      attempt,
		ThrottleWait: throttleWait,
		Duration:     duration,
	}

	if reqErr, ok := err.(*RequestError); ok {
		info.StatusCode = reqErr.StatusCode
		info.RetryAfter = reqErr.RetryAfter
	} else if err == nil {
		info.StatusCode = 200 // NewRequest's callers never see the exact 2xx code (see doRequest); "success" is all a hook needs.
	}

	return info
}
