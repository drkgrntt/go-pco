package pco

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// resetRetryForTesting clears every package-level retry var (the
// explicit/env-resolved config and the env-var sync.Once) before and after
// a test - same reasoning as resetThrottleForTesting in throttle_test.go:
// resolveRetryConfig's env check only ever runs once per process via
// sync.Once, so a test that wants a specific env-derived config needs a
// fresh Once, and must leave one behind for whichever test runs next.
func resetRetryForTesting(t *testing.T) {
	t.Helper()

	reset := func() {
		retryMu.Lock()
		retryCfg = RetryConfig{}
		retryExplicit = false
		retryMu.Unlock()

		retryEnvOnce = sync.Once{}
	}

	reset()
	t.Cleanup(reset)
}

// --- Pure config/backoff logic ---

func TestApplyRetryDefaults(t *testing.T) {
	cfg := applyRetryDefaults(RetryConfig{Enabled: true})
	if cfg.MaxAttempts != defaultRetryMaxAttempts {
		t.Errorf("expected default MaxAttempts %d, got %d", defaultRetryMaxAttempts, cfg.MaxAttempts)
	}
	if cfg.MaxWait != defaultRetryMaxWait {
		t.Errorf("expected default MaxWait %v, got %v", defaultRetryMaxWait, cfg.MaxWait)
	}
	if cfg.FallbackWait != defaultRetryFallbackWait {
		t.Errorf("expected default FallbackWait %v, got %v", defaultRetryFallbackWait, cfg.FallbackWait)
	}
	if !cfg.Enabled {
		t.Error("expected Enabled to pass through unchanged")
	}

	explicit := applyRetryDefaults(RetryConfig{MaxAttempts: 5, MaxWait: time.Minute, FallbackWait: time.Second})
	if explicit.MaxAttempts != 5 || explicit.MaxWait != time.Minute || explicit.FallbackWait != time.Second {
		t.Errorf("expected explicit non-positive-guarded fields to survive unchanged, got %+v", explicit)
	}
}

func TestConfigureRetryAppliesDefaults(t *testing.T) {
	resetRetryForTesting(t)

	ConfigureRetry(RetryConfig{Enabled: true})

	cfg := resolveRetryConfig()
	if !cfg.Enabled {
		t.Error("expected Enabled true")
	}
	if cfg.MaxAttempts != defaultRetryMaxAttempts {
		t.Errorf("expected default MaxAttempts, got %d", cfg.MaxAttempts)
	}
}

func TestConfigureRetryOverridesEnvVars(t *testing.T) {
	resetRetryForTesting(t)

	t.Setenv("PCO_RETRY_ON_429_ENABLED", "true")
	t.Setenv(retryEnvMaxAttemptsVar, "9")

	// An explicit call wins even though the env vars above would resolve
	// to something different.
	ConfigureRetry(RetryConfig{Enabled: true, MaxAttempts: 3})

	cfg := resolveRetryConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("expected explicit MaxAttempts 3 to win over env's 9, got %d", cfg.MaxAttempts)
	}
}

func TestEnvVarsUsedWhenConfigureRetryNeverCalled(t *testing.T) {
	resetRetryForTesting(t)

	t.Setenv("PCO_RETRY_ON_429_ENABLED", "1")
	t.Setenv(retryEnvMaxAttemptsVar, "4")
	t.Setenv(retryEnvMaxWaitSecondsVar, "7")

	cfg := resolveRetryConfig()
	if !cfg.Enabled {
		t.Error("expected PCO_RETRY_ON_429_ENABLED=1 to enable the retry")
	}
	if cfg.MaxAttempts != 4 {
		t.Errorf("expected MaxAttempts 4 from env, got %d", cfg.MaxAttempts)
	}
	if cfg.MaxWait != 7*time.Second {
		t.Errorf("expected MaxWait 7s from env, got %v", cfg.MaxWait)
	}
}

func TestRetryDisabledByDefaultWithNoConfigureCallAndNoEnvVars(t *testing.T) {
	resetRetryForTesting(t)

	if cfg := resolveRetryConfig(); cfg.Enabled {
		t.Error("expected retry disabled by default with no ConfigureRetry call and no env vars set")
	}
}

func TestRetryWaitForUsesRetryAfterHeader(t *testing.T) {
	cfg := applyRetryDefaults(RetryConfig{})
	got := retryWaitFor("3", cfg)
	if got != 3*time.Second {
		t.Errorf("expected 3s from Retry-After header, got %v", got)
	}
}

func TestRetryWaitForFallsBackWhenHeaderAbsentOrUnparsable(t *testing.T) {
	cfg := applyRetryDefaults(RetryConfig{FallbackWait: 250 * time.Millisecond})

	for _, retryAfter := range []string{"", "not-a-number", "-1"} {
		if got := retryWaitFor(retryAfter, cfg); got != cfg.FallbackWait {
			t.Errorf("retryWaitFor(%q): expected fallback %v, got %v", retryAfter, cfg.FallbackWait, got)
		}
	}
}

func TestRetryWaitForCapsAtMaxWait(t *testing.T) {
	cfg := applyRetryDefaults(RetryConfig{MaxWait: 5 * time.Second})
	if got := retryWaitFor("3600", cfg); got != 5*time.Second {
		t.Errorf("expected PCO's requested 3600s capped at MaxWait 5s, got %v", got)
	}
}

// --- NewRequest integration: does the retry actually fire (or not) ---

// TestNewRequestRetriesOn429ThenSucceeds confirms a 429 followed by a 200
// is retried once and returns the eventual success, not the first
// failure.
func TestNewRequestRetriesOn429ThenSucceeds(t *testing.T) {
	resetRetryForTesting(t)
	ConfigureRetry(RetryConfig{Enabled: true, MaxAttempts: 2, FallbackWait: time.Millisecond})

	var hits int32
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			writeJSON(t, w, http.StatusTooManyRequests, `{"errors":[{"status":"429","title":"Rate limit exceeded"}]}`)
			return
		}
		writeJSON(t, w, http.StatusOK, `{"greeting":"hello"}`)
	})

	type response struct {
		Greeting string `json:"greeting"`
	}
	got, err := NewRequest[response](context.Background(), http.MethodGet, baseURL+"/thing", nil)
	if err != nil {
		t.Fatalf("expected the retried call to succeed, got error: %v", err)
	}
	if got.Greeting != "hello" {
		t.Errorf("expected the second (successful) response body, got %+v", got)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("expected exactly 2 attempts, got %d", got)
	}
}

// TestNewRequestRetryStopsAtMaxAttempts confirms a caller who keeps
// getting 429s doesn't retry forever - it gives up after MaxAttempts and
// returns the last 429 as a real error.
func TestNewRequestRetryStopsAtMaxAttempts(t *testing.T) {
	resetRetryForTesting(t)
	ConfigureRetry(RetryConfig{Enabled: true, MaxAttempts: 3, FallbackWait: time.Millisecond})

	var hits int32
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Retry-After", "0")
		writeJSON(t, w, http.StatusTooManyRequests, `{"errors":[{"status":"429","title":"Rate limit exceeded"}]}`)
	})

	_, err := NewRequest[struct{}](context.Background(), http.MethodGet, baseURL+"/thing", nil)
	reqErr, ok := err.(*RequestError)
	if !ok || reqErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected a final *RequestError with status 429, got %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Errorf("expected exactly MaxAttempts (3) attempts, got %d", got)
	}
}

// TestNewRequestDoesNotRetryNon429Errors confirms the retry is scoped to
// 429 specifically - a 422 (a request built wrong) must not be retried,
// even with the retry enabled, since retrying it just gets the same error
// again for no benefit.
func TestNewRequestDoesNotRetryNon429Errors(t *testing.T) {
	resetRetryForTesting(t)
	ConfigureRetry(RetryConfig{Enabled: true, MaxAttempts: 5, FallbackWait: time.Millisecond})

	var hits int32
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		writeJSON(t, w, http.StatusUnprocessableEntity, `{"errors":[{"status":"422","title":"Unprocessable Entity"}]}`)
	})

	_, err := NewRequest[struct{}](context.Background(), http.MethodPost, baseURL+"/thing", map[string]string{})
	reqErr, ok := err.(*RequestError)
	if !ok || reqErr.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected a *RequestError with status 422, got %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("expected exactly 1 attempt for a non-429 error, got %d", got)
	}
}

// TestNewRequestRetryOffByDefaultMakesOneAttempt confirms a 429 with the
// retry left at its default (off) behaves exactly as it did before the
// retry existed: one attempt, the 429 returned straight to the caller.
func TestNewRequestRetryOffByDefaultMakesOneAttempt(t *testing.T) {
	resetRetryForTesting(t)

	var hits int32
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		writeJSON(t, w, http.StatusTooManyRequests, `{"errors":[{"status":"429","title":"Rate limit exceeded"}]}`)
	})

	_, err := NewRequest[struct{}](context.Background(), http.MethodGet, baseURL+"/thing", nil)
	if err == nil {
		t.Fatal("expected an error for a 429 response")
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("expected exactly 1 attempt with retry disabled (the default), got %d", got)
	}
}

// TestNewRequestRetryRespectsContextCancellation confirms a caller whose
// context is done doesn't sit through a long Retry-After wait - it returns
// promptly with the context's own error instead.
func TestNewRequestRetryRespectsContextCancellation(t *testing.T) {
	resetRetryForTesting(t)
	ConfigureRetry(RetryConfig{Enabled: true, MaxAttempts: 2, MaxWait: time.Hour, FallbackWait: time.Hour})

	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// No Retry-After header - falls back to cfg.FallbackWait (an hour
		// above), which the context timeout below must preempt.
		writeJSON(t, w, http.StatusTooManyRequests, `{"errors":[{"status":"429","title":"Rate limit exceeded"}]}`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := NewRequest[struct{}](ctx, http.MethodGet, baseURL+"/thing", nil)
	elapsed := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected prompt return on context cancellation, took %v", elapsed)
	}
}
