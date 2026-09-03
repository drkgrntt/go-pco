package pco

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// resetResponseHookForTesting clears any registered response hook before
// and after a test - the hook is a package-level var (see observe.go), so
// a test that registers one and forgets to clear it would leak into every
// later test's NewRequest calls, potentially calling back into a
// *testing.T whose test has already finished.
func resetResponseHookForTesting(t *testing.T) {
	t.Helper()
	SetResponseHook(nil)
	t.Cleanup(func() { SetResponseHook(nil) })
}

func TestResponseHookNotCalledWhenUnset(t *testing.T) {
	resetResponseHookForTesting(t)

	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{}`)
	})

	// No hook registered - this must not panic (callResponseHook's whole
	// point is being a safe no-op here).
	if _, err := NewRequest[struct{}](context.Background(), http.MethodGet, baseURL+"/thing", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResponseHookCalledOnSuccess(t *testing.T) {
	resetResponseHookForTesting(t)

	var got []ResponseInfo
	SetResponseHook(func(info ResponseInfo) { got = append(got, info) })

	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{}`)
	})

	if _, err := NewRequest[struct{}](context.Background(), http.MethodGet, baseURL+"/widgets/1", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 hook call, got %d", len(got))
	}
	info := got[0]
	if info.Method != http.MethodGet || info.URL != baseURL+"/widgets/1" {
		t.Errorf("expected method/URL to match the call, got %q %q", info.Method, info.URL)
	}
	if info.Err != nil {
		t.Errorf("expected nil Err on success, got %v", info.Err)
	}
	if info.StatusCode != http.StatusOK {
		t.Errorf("expected StatusCode 200, got %d", info.StatusCode)
	}
	if info.Attempt != 1 {
		t.Errorf("expected Attempt 1, got %d", info.Attempt)
	}
	if info.Duration <= 0 {
		t.Error("expected a positive Duration")
	}
}

func TestResponseHookCalledOnErrorWithStatusAndRetryAfter(t *testing.T) {
	resetResponseHookForTesting(t)

	var got ResponseInfo
	var calls int
	SetResponseHook(func(info ResponseInfo) {
		calls++
		got = info
	})

	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "12")
		writeJSON(t, w, http.StatusTooManyRequests, `{"errors":[{"status":"429","title":"Rate limit exceeded"}]}`)
	})

	_, err := NewRequest[struct{}](context.Background(), http.MethodGet, baseURL+"/thing", nil)
	if err == nil {
		t.Fatal("expected an error for a 429 response")
	}

	if calls != 1 {
		t.Fatalf("expected exactly 1 hook call, got %d", calls)
	}
	if got.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected StatusCode 429, got %d", got.StatusCode)
	}
	if got.RetryAfter != "12" {
		t.Errorf("expected RetryAfter %q, got %q", "12", got.RetryAfter)
	}
	if got.Err == nil {
		t.Error("expected a non-nil Err")
	}
}

// TestResponseHookFiresOncePerAttempt confirms a retried call (see
// retry_test.go) reports one ResponseInfo per HTTP attempt, with
// increasing Attempt numbers - not just one summary for the whole
// logical call - so a consumer logging every attempt can see the 429
// that got retried, not only the eventual outcome.
func TestResponseHookFiresOncePerAttempt(t *testing.T) {
	resetResponseHookForTesting(t)
	resetRetryForTesting(t)
	ConfigureRetry(RetryConfig{Enabled: true, MaxAttempts: 2, FallbackWait: time.Millisecond})

	var got []ResponseInfo
	SetResponseHook(func(info ResponseInfo) { got = append(got, info) })

	call := 0
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			w.Header().Set("Retry-After", "0")
			writeJSON(t, w, http.StatusTooManyRequests, `{"errors":[{"status":"429","title":"Rate limit exceeded"}]}`)
			return
		}
		writeJSON(t, w, http.StatusOK, `{}`)
	})

	if _, err := NewRequest[struct{}](context.Background(), http.MethodGet, baseURL+"/thing", nil); err != nil {
		t.Fatalf("expected the retried call to succeed, got: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 hook calls (one per attempt), got %d", len(got))
	}
	if got[0].Attempt != 1 || got[0].StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected first attempt to report Attempt 1, status 429, got %+v", got[0])
	}
	if got[1].Attempt != 2 || got[1].StatusCode != http.StatusOK {
		t.Errorf("expected second attempt to report Attempt 2, status 200, got %+v", got[1])
	}
}

// TestResponseHookReportsThrottleWait confirms ThrottleWait reflects real
// admission delay when the throttle is enabled, not just left at zero -
// forces a wait by exhausting a fresh token's bucket before the call.
func TestResponseHookReportsThrottleWait(t *testing.T) {
	resetResponseHookForTesting(t)
	resetThrottleForTesting(t)
	throttleMu.Lock()
	throttleCfg = ThrottleConfig{Enabled: true, RatePerWindow: 1, Window: 50 * time.Millisecond, DefaultPriority: 10, IdleEvictAfter: time.Hour}
	throttleExplicit = true
	throttleMu.Unlock()

	// Drain the bucket for this context's token before the real call, so
	// NewRequest's own admission has to wait for a refill.
	ctx := WithAccessToken(context.Background(), "observe-test-token")
	tl := getTokenLimiter("observe-test-token", resolveThrottleConfig())
	tl.mu.Lock()
	tl.tokens = 0
	tl.mu.Unlock()

	var got ResponseInfo
	SetResponseHook(func(info ResponseInfo) { got = info })

	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{}`)
	})

	if _, err := NewRequest[struct{}](ctx, http.MethodGet, baseURL+"/thing", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ThrottleWait <= 0 {
		t.Errorf("expected a positive ThrottleWait after draining the bucket, got %v", got.ThrottleWait)
	}
}
