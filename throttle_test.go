package pco

import (
	"container/heap"
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

// resetThrottleForTesting clears every package-level throttle var (the
// explicit/env-resolved config, the env-var sync.Once, and the per-token
// registry) before and after a test, so throttle tests don't leak state
// into each other or into unrelated tests in this package. These tests
// must not run under t.Parallel() relative to one another for that reason.
func resetThrottleForTesting(t *testing.T) {
	t.Helper()

	reset := func() {
		throttleMu.Lock()
		throttleCfg = ThrottleConfig{}
		throttleExplicit = false
		throttleMu.Unlock()

		throttleEnvOnce = sync.Once{}

		throttleRegistryMu.Lock()
		throttleLimiters = map[string]*tokenLimiter{}
		throttleRegistryMu.Unlock()
	}

	reset()
	t.Cleanup(reset)
}

// --- Pure scheduler-logic tests: priority ordering, FIFO tie-break ---

func TestWaiterHeapPriorityOrder(t *testing.T) {
	h := &waiterHeap{}
	heap.Init(h)

	// Push out of order; lower priority number must come out first.
	order := []int{20, 1, 10, 5}
	waiters := make([]*throttleWaiter, len(order))
	for i, p := range order {
		w := &throttleWaiter{priority: p, seq: int64(i), ch: make(chan struct{})}
		waiters[i] = w
		heap.Push(h, w)
	}

	var got []int
	for h.Len() > 0 {
		w := heap.Pop(h).(*throttleWaiter)
		got = append(got, w.priority)
	}

	want := []int{1, 5, 10, 20}
	if len(got) != len(want) {
		t.Fatalf("expected %d waiters, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("expected pop order %v, got %v", want, got)
			break
		}
	}
}

func TestWaiterHeapSamePrioritySameOrderFIFO(t *testing.T) {
	h := &waiterHeap{}
	heap.Init(h)

	// All same priority; seq must decide order and it must be FIFO (the
	// order they were pushed), never reordered.
	const n = 8
	for i := 0; i < n; i++ {
		heap.Push(h, &throttleWaiter{priority: 10, seq: int64(i), ch: make(chan struct{})})
	}

	for i := 0; i < n; i++ {
		w := heap.Pop(h).(*throttleWaiter)
		if w.seq != int64(i) {
			t.Fatalf("expected FIFO pop order, expected seq %d, got %d", i, w.seq)
		}
	}
}

func TestWaiterHeapMixedPriorityAndFIFO(t *testing.T) {
	h := &waiterHeap{}
	heap.Init(h)

	// Two waiters at priority 1 (pushed first and third), one at priority 5
	// pushed in between - the two priority-1 waiters must come out in the
	// order they were pushed (seq order), both before the priority-5 one.
	a := &throttleWaiter{priority: 1, seq: 0, ch: make(chan struct{})}
	b := &throttleWaiter{priority: 5, seq: 1, ch: make(chan struct{})}
	c := &throttleWaiter{priority: 1, seq: 2, ch: make(chan struct{})}
	heap.Push(h, a)
	heap.Push(h, b)
	heap.Push(h, c)

	first := heap.Pop(h).(*throttleWaiter)
	second := heap.Pop(h).(*throttleWaiter)
	third := heap.Pop(h).(*throttleWaiter)

	if first != a || second != c || third != b {
		t.Fatalf("expected pop order a, c, b (priority then FIFO), got seqs %d, %d, %d", first.seq, second.seq, third.seq)
	}
}

// --- Pure rate-limiting arithmetic tests: no goroutines needed ---

func TestTokenLimiterRefillArithmetic(t *testing.T) {
	tl := newTokenLimiter(ThrottleConfig{RatePerWindow: 10, Window: time.Second})
	// Drain the bucket to zero.
	tl.tokens = 0
	tl.lastRefill = time.Unix(0, 0)

	// Half the window elapsed -> half the rate refilled (5 tokens).
	tl.refillLocked(time.Unix(0, 0).Add(500 * time.Millisecond))
	if got := tl.tokens; got < 4.9 || got > 5.1 {
		t.Fatalf("expected ~5 tokens refilled after half the window, got %v", got)
	}
}

func TestTokenLimiterRefillCapsAtRate(t *testing.T) {
	tl := newTokenLimiter(ThrottleConfig{RatePerWindow: 10, Window: time.Second})
	tl.tokens = 9
	tl.lastRefill = time.Unix(0, 0)

	// A full minute elapsed - far more than enough to refill, but the
	// bucket must cap at its capacity (rate), never exceed it.
	tl.refillLocked(time.Unix(0, 0).Add(time.Minute))
	if tl.tokens != 10 {
		t.Fatalf("expected tokens capped at rate (10), got %v", tl.tokens)
	}
}

func TestTokenLimiterTryAdmitAdmitsUpToBudget(t *testing.T) {
	tl := newTokenLimiter(ThrottleConfig{RatePerWindow: 10, Window: time.Second})
	tl.tokens = 2 // only 2 tokens available

	now := time.Now()
	tl.lastRefill = now // no additional refill during this check

	for i := 0; i < 5; i++ {
		w := &throttleWaiter{priority: 10, seq: int64(i), ch: make(chan struct{})}
		heap.Push(&tl.heap, w)
	}
	tl.tryAdmitLocked(now)

	// Exactly 2 should have been admitted (channel closed), 3 left queued.
	if tl.heap.Len() != 3 {
		t.Fatalf("expected 3 waiters left queued after admitting 2, got %d", tl.heap.Len())
	}
	if tl.tokens >= 1 {
		t.Fatalf("expected fewer than 1 token left after admitting 2 of budget 2, got %v", tl.tokens)
	}
}

func TestTokenLimiterTimeUntilNextToken(t *testing.T) {
	tl := newTokenLimiter(ThrottleConfig{RatePerWindow: 10, Window: 10 * time.Second})

	tl.tokens = 1.5
	if got := tl.timeUntilNextTokenLocked(); got != 0 {
		t.Errorf("expected 0 wait when a token is already available, got %v", got)
	}

	tl.tokens = 0
	// need 1 token; refill rate is 10 tokens per 10s = 1 token/s, so wait ~1s.
	got := tl.timeUntilNextTokenLocked()
	if got < 900*time.Millisecond || got > 1100*time.Millisecond {
		t.Errorf("expected ~1s wait for the next token, got %v", got)
	}
}

// --- Concurrency-level tests ---

func TestAdmitOrdersMixedPriorities(t *testing.T) {
	resetThrottleForTesting(t)

	priorities := []int{20, 1, 10, 1, 5}

	// Window long enough that no auto-refill happens during the test - the
	// only tokens available are the ones the test hands out explicitly,
	// one at a time below.
	tl := newTokenLimiter(ThrottleConfig{RatePerWindow: 1, Window: time.Hour})
	tl.tokens = 0 // force everything to queue

	// admittedCh reports each admission as it happens. Recording order via
	// a shared slice appended to *after* each goroutine wakes from admit()
	// would be racy - closing several waiters' channels doesn't guarantee
	// the order the runtime resumes those goroutines in. Handing out
	// exactly one token at a time (below) and waiting for exactly one
	// admission to land before releasing the next sidesteps that: only one
	// specific waiter (the true head of the queue) can possibly be ready to
	// send at each step.
	admittedCh := make(chan int, len(priorities))
	var wg sync.WaitGroup
	for _, p := range priorities {
		wg.Add(1)
		go func(priority int) {
			defer wg.Done()
			if err := tl.admit(context.Background(), priority); err != nil {
				t.Errorf("unexpected admit error: %v", err)
				return
			}
			admittedCh <- priority
		}(p)
	}

	// Give every goroutine time to enqueue before releasing any tokens, so
	// admission order reflects the full queue, not enqueue-time races.
	time.Sleep(50 * time.Millisecond)

	var got []int
	for range priorities {
		tl.mu.Lock()
		tl.tokens = 1
		tl.tryAdmitLocked(time.Now())
		tl.mu.Unlock()

		select {
		case p := <-admittedCh:
			got = append(got, p)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for the next admission")
		}
	}

	wg.Wait()

	want := []int{1, 1, 5, 10, 20}
	if len(got) != len(want) {
		t.Fatalf("expected %d admissions, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected admission order %v, got %v", want, got)
		}
	}
}

func TestAdmitCancelledContextReturnsPromptly(t *testing.T) {
	resetThrottleForTesting(t)

	// Rate exhausted, window an hour long - without cancellation this
	// would block for a very long time.
	tl := newTokenLimiter(ThrottleConfig{RatePerWindow: 1, Window: time.Hour})
	tl.tokens = 0

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := tl.admit(ctx, 10)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected prompt return on context cancellation, took %v", elapsed)
	}

	// The waiter must actually have been removed from the queue, not just
	// abandoned - otherwise it'd wrongly consume a future admission slot.
	tl.mu.Lock()
	n := tl.heap.Len()
	tl.mu.Unlock()
	if n != 0 {
		t.Fatalf("expected cancelled waiter removed from queue, %d still queued", n)
	}
}

func TestAdmitAlreadyCancelledContextReturnsImmediately(t *testing.T) {
	resetThrottleForTesting(t)

	tl := newTokenLimiter(ThrottleConfig{RatePerWindow: 1, Window: time.Hour})
	tl.tokens = 0

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := tl.admit(ctx, 10); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled for an already-cancelled context, got %v", err)
	}
}

func TestTokenLimitersAreIndependentPerToken(t *testing.T) {
	resetThrottleForTesting(t)
	throttleMu.Lock()
	throttleCfg = ThrottleConfig{RatePerWindow: 1, Window: time.Hour, DefaultPriority: 10, IdleEvictAfter: time.Hour}
	throttleExplicit = true
	throttleMu.Unlock()

	cfg := resolveThrottleConfig()

	tokenA := getTokenLimiter("token-a", cfg)
	tokenB := getTokenLimiter("token-b", cfg)

	if tokenA == tokenB {
		t.Fatal("expected distinct tokenLimiters for distinct keys")
	}

	// Exhaust token A's budget and queue several waiters on it.
	tokenA.mu.Lock()
	tokenA.tokens = 0
	tokenA.mu.Unlock()

	for i := 0; i < 5; i++ {
		go func() { _ = tokenA.admit(context.Background(), 10) }()
	}
	time.Sleep(20 * time.Millisecond) // let them pile up in token A's queue

	// Token B, untouched, should admit essentially immediately even while
	// token A's queue is backed up.
	start := time.Now()
	if err := tokenB.admit(context.Background(), 10); err != nil {
		t.Fatalf("unexpected error admitting on independent token: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		t.Fatalf("expected token B to admit promptly despite token A's busy queue, took %v", elapsed)
	}
}

func TestGetTokenLimiterEvictsIdleEntries(t *testing.T) {
	resetThrottleForTesting(t)

	cfg := ThrottleConfig{RatePerWindow: 10, Window: time.Second, DefaultPriority: 10, IdleEvictAfter: 10 * time.Millisecond}

	first := getTokenLimiter("stale-token", cfg)
	time.Sleep(30 * time.Millisecond)

	// Fetching a different token should sweep the now-idle "stale-token"
	// entry out of the registry.
	getTokenLimiter("fresh-token", cfg)

	second := getTokenLimiter("stale-token", cfg)
	if first == second {
		t.Fatal("expected the idle-evicted token's limiter to be recreated, got the same instance back")
	}
}

// --- ConfigureThrottle / env var precedence ---

func TestConfigureThrottleAppliesDefaults(t *testing.T) {
	resetThrottleForTesting(t)

	ConfigureThrottle(ThrottleConfig{Enabled: true})
	cfg := resolveThrottleConfig()

	if !cfg.Enabled {
		t.Error("expected Enabled to stay true")
	}
	if cfg.RatePerWindow != defaultThrottleRatePerWindow {
		t.Errorf("expected default RatePerWindow %d, got %d", defaultThrottleRatePerWindow, cfg.RatePerWindow)
	}
	if cfg.Window != defaultThrottleWindow {
		t.Errorf("expected default Window %v, got %v", defaultThrottleWindow, cfg.Window)
	}
	if cfg.DefaultPriority != defaultThrottlePriority {
		t.Errorf("expected default DefaultPriority %d, got %d", defaultThrottlePriority, cfg.DefaultPriority)
	}
	if cfg.IdleEvictAfter != defaultThrottleIdleEvictAfter {
		t.Errorf("expected default IdleEvictAfter %v, got %v", defaultThrottleIdleEvictAfter, cfg.IdleEvictAfter)
	}
}

func TestConfigureThrottleOverridesEnvVars(t *testing.T) {
	resetThrottleForTesting(t)

	t.Setenv("PCO_THROTTLE_ENABLED", "true")
	t.Setenv("PCO_THROTTLE_RATE_PER_20S", "5")
	t.Setenv("PCO_THROTTLE_DEFAULT_PRIORITY", "1")

	// An explicit call must win even though env vars are also set.
	ConfigureThrottle(ThrottleConfig{Enabled: true, RatePerWindow: 99, DefaultPriority: 42})

	cfg := resolveThrottleConfig()
	if cfg.RatePerWindow != 99 {
		t.Errorf("expected explicit config (99) to win over env var (5), got %d", cfg.RatePerWindow)
	}
	if cfg.DefaultPriority != 42 {
		t.Errorf("expected explicit config (42) to win over env var (1), got %d", cfg.DefaultPriority)
	}
}

func TestEnvVarsUsedWhenConfigureThrottleNeverCalled(t *testing.T) {
	resetThrottleForTesting(t)

	t.Setenv("PCO_THROTTLE_ENABLED", "1")
	t.Setenv("PCO_THROTTLE_RATE_PER_20S", "7")
	t.Setenv("PCO_THROTTLE_DEFAULT_PRIORITY", "3")

	cfg := resolveThrottleConfig()
	if !cfg.Enabled {
		t.Error("expected PCO_THROTTLE_ENABLED=1 to enable the throttle")
	}
	if cfg.RatePerWindow != 7 {
		t.Errorf("expected RatePerWindow 7 from env var, got %d", cfg.RatePerWindow)
	}
	if cfg.DefaultPriority != 3 {
		t.Errorf("expected DefaultPriority 3 from env var, got %d", cfg.DefaultPriority)
	}
	if cfg.Window != defaultThrottleWindow {
		t.Errorf("expected default Window (20s) since env vars don't set it, got %v", cfg.Window)
	}
}

func TestEnvVarsDisabledWhenUnset(t *testing.T) {
	resetThrottleForTesting(t)

	cfg := resolveThrottleConfig()
	if cfg.Enabled {
		t.Error("expected throttle disabled by default with no ConfigureThrottle call and no env vars set")
	}
}

// --- WithPriority / priorityFromContext ---

func TestPriorityFromContextDefaultsWhenUnset(t *testing.T) {
	if got := priorityFromContext(context.Background(), 10); got != 10 {
		t.Errorf("expected default priority 10, got %d", got)
	}
}

func TestWithPriorityOverridesDefault(t *testing.T) {
	ctx := WithPriority(context.Background(), 1)
	if got := priorityFromContext(ctx, 10); got != 1 {
		t.Errorf("expected priority 1 from WithPriority, got %d", got)
	}
}

// --- throttleTokenKey ---

func TestThrottleTokenKeyUsesAccessTokenWhenPresent(t *testing.T) {
	ctx := WithAccessToken(context.Background(), "user-token")
	if got := throttleTokenKey(ctx); got != "user-token" {
		t.Errorf("expected token key %q, got %q", "user-token", got)
	}
}

func TestThrottleTokenKeyFallsBackToPATKey(t *testing.T) {
	if got := throttleTokenKey(context.Background()); got != patLimiterKey {
		t.Errorf("expected PAT fallback key, got %q", got)
	}
}

// --- Integration-style tests via SetBaseURLForTesting ---

func TestNewRequestThrottleDisabledIsNoop(t *testing.T) {
	resetThrottleForTesting(t)
	// Explicitly disabled (also the default with nothing configured).
	ConfigureThrottle(ThrottleConfig{Enabled: false})

	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{}`)
	})

	const n = 20
	start := time.Now()
	for i := 0; i < n; i++ {
		if _, err := NewRequest[struct{}](context.Background(), http.MethodGet, baseURL+"/thing", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	elapsed := time.Since(start)

	// n plain HTTP round trips to a local httptest.Server should be fast;
	// with the throttle enabled at any realistic rate this would take much
	// longer. This is a loose bound (not a tight latency assertion) since
	// CI/dev machines vary, but it should easily catch an accidental queue
	// on the disabled path.
	if elapsed > 2*time.Second {
		t.Fatalf("expected disabled throttle to add no meaningful delay across %d requests, took %v", n, elapsed)
	}
}

func TestNewRequestThrottleEnabledEnforcesRate(t *testing.T) {
	resetThrottleForTesting(t)
	ConfigureThrottle(ThrottleConfig{
		Enabled:         true,
		RatePerWindow:   2,
		Window:          200 * time.Millisecond,
		DefaultPriority: 10,
		IdleEvictAfter:  time.Minute,
	})

	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{}`)
	})

	const n = 5 // budget is 2, so 3 of these must wait on refill
	start := time.Now()
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := NewRequest[struct{}](context.Background(), http.MethodGet, baseURL+"/thing", nil)
			errs[i] = err
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	for i, err := range errs {
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i, err)
		}
	}

	// 3 requests beyond the initial burst of 2, at 2 tokens per 200ms,
	// need at least ~150ms more of refill (1.5 windows). Give it a wide
	// margin below that floor and a generous ceiling to avoid flakiness.
	if elapsed < 80*time.Millisecond {
		t.Fatalf("expected rate limiting to add meaningful delay for %d requests over a budget of 2, took %v", n, elapsed)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("rate-limited requests took implausibly long: %v", elapsed)
	}
}

func TestNewRequestThrottleEnabledRespectsContextCancellation(t *testing.T) {
	resetThrottleForTesting(t)
	ConfigureThrottle(ThrottleConfig{
		Enabled:         true,
		RatePerWindow:   1,
		Window:          time.Hour, // effectively no refill during this test
		DefaultPriority: 10,
		IdleEvictAfter:  time.Minute,
	})

	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{}`)
	})

	// Consume the only token.
	if _, err := NewRequest[struct{}](context.Background(), http.MethodGet, baseURL+"/thing", nil); err != nil {
		t.Fatalf("unexpected error on first request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := NewRequest[struct{}](ctx, http.MethodGet, baseURL+"/thing", nil)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded while queued behind an exhausted budget, got %v", err)
	}
	if elapsed > 1*time.Second {
		t.Fatalf("expected prompt return on cancellation instead of waiting out the hour-long window, took %v", elapsed)
	}
}
