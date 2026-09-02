package pco

import (
	"container/heap"
	"context"
	"os"
	"strconv"
	"sync"
	"time"
)

// This file implements go-pco's opt-in, priority-aware request throttle.
// It's off by default - a consumer that never calls ConfigureThrottle and
// never sets a PCO_THROTTLE_* env var gets exactly today's behavior, no
// queueing, no added latency (see doRequest/NewRequest in pco.go). Once
// enabled, every call to NewRequest is admitted through a per-access-token
// scheduler before the real HTTP call happens.
//
// Why per-token, not one global limiter: PCO's rate limit
// (https://api.planningcenteronline.com/docs/overview/rate-limiting) is
// per authenticated user token, not per-process. A single process-wide
// queue would let one busy token's traffic (e.g. a background sync for one
// organization) delay an unrelated token's interactive traffic, even
// though each token has its own independent budget with PCO. So the
// throttle keys its state by whatever WithAccessToken put on the ctx (see
// pco.go), falling back to one fixed key for PAT-authenticated calls,
// since every PAT call shares that single credential's budget too.
//
// Deliberate non-goal: no fairness/anti-starvation between priority
// levels. See tokenLimiter's doc comment below - do not "fix" this
// without reading it first.

// ThrottleConfig configures the throttle. Pass it to ConfigureThrottle to
// opt in explicitly; a zero-value RatePerWindow/Window/DefaultPriority/
// IdleEvictAfter field is treated as "unset" and replaced with the
// package's default for that field (see the const block below) - so a
// caller who only cares about turning the throttle on can write
// ConfigureThrottle(ThrottleConfig{Enabled: true}) and get sane defaults
// for everything else.
type ThrottleConfig struct {
	// Enabled opts into the throttle. False (the zero value) means "off,"
	// unlike the other fields here - there's no separate "unset" state for
	// this one, since "off" already is the safe default.
	Enabled bool

	// RatePerWindow and Window together bound sustained throughput per
	// access token: at most RatePerWindow requests admitted per Window,
	// refilled continuously (a token bucket, not a hard reset every
	// Window). Non-positive values fall back to the package defaults
	// below.
	RatePerWindow int
	Window        time.Duration

	// DefaultPriority applies to any call whose context never set one via
	// WithPriority. Non-positive values fall back to defaultThrottlePriority
	// (10) - if you specifically want the process default to admit before
	// WordPress-style priority 10, set it via WithPriority on individual
	// calls instead of trying to drive it to 0 or below here.
	DefaultPriority int

	// IdleEvictAfter controls how long a per-token limiter can sit with no
	// activity before it's dropped from the internal registry, so a
	// long-running process doesn't accumulate an ever-growing map across
	// many distinct users over its lifetime. Non-positive falls back to
	// defaultThrottleIdleEvictAfter (30m).
	IdleEvictAfter time.Duration
}

const (
	// defaultThrottleRatePerWindow is the shipped default when a consumer
	// enables the throttle without setting RatePerWindow. It's deliberately
	// more conservative than it needs to be against PCO's raw 100-req/20s
	// ceiling: 60 (60% headroom) rather than a first-draft proposal of 80
	// (80% headroom), matching the same 60%-of-ceiling figure this SDK's
	// primary consumer already uses by hand for its own Song Info
	// background sync. The reasoning for picking the more conservative
	// number: raising a rate limit later, once there's real production
	// signal that more headroom is safe, is a low-risk change; lowering
	// one that already looked "safe" after a consumer has come to rely on
	// it is not. Revisit this once the throttle has actually run in
	// production for a while.
	defaultThrottleRatePerWindow = 60
	defaultThrottleWindow        = 20 * time.Second

	// defaultThrottlePriority is deliberately the same value (10) WordPress
	// uses as add_action/add_filter's default priority argument - see
	// WithPriority's doc comment for why matching that convention matters
	// here.
	defaultThrottlePriority = 10

	defaultThrottleIdleEvictAfter = 30 * time.Minute
)

// applyThrottleDefaults returns cfg with every non-positive numeric field
// replaced by its package default. Enabled is passed through unchanged -
// see its own doc comment on ThrottleConfig for why it has no "unset"
// state.
func applyThrottleDefaults(cfg ThrottleConfig) ThrottleConfig {
	if cfg.RatePerWindow <= 0 {
		cfg.RatePerWindow = defaultThrottleRatePerWindow
	}
	if cfg.Window <= 0 {
		cfg.Window = defaultThrottleWindow
	}
	if cfg.DefaultPriority <= 0 {
		cfg.DefaultPriority = defaultThrottlePriority
	}
	if cfg.IdleEvictAfter <= 0 {
		cfg.IdleEvictAfter = defaultThrottleIdleEvictAfter
	}
	return cfg
}

var (
	throttleMu       sync.RWMutex
	throttleCfg      ThrottleConfig
	throttleExplicit bool // true once ConfigureThrottle has been called

	throttleEnvOnce sync.Once
)

// ConfigureThrottle opts into (or explicitly configures) the request
// throttle. Call it once, before making requests - e.g. alongside other
// package-level construction at process startup. An explicit call always
// takes precedence over the PCO_THROTTLE_* environment variables below,
// even if a request was already made (and so already resolved the env
// vars) before this was called.
//
// If ConfigureThrottle is never called, NewRequest instead lazily checks
// the environment on its first call (via sync.Once, so the check only
// happens once per process) - this lets a pure ops/deployment toggle
// (setting PCO_THROTTLE_ENABLED and friends) enable the throttle with no
// code change at all.
func ConfigureThrottle(cfg ThrottleConfig) {
	cfg = applyThrottleDefaults(cfg)

	throttleMu.Lock()
	throttleCfg = cfg
	throttleExplicit = true
	throttleMu.Unlock()
}

// resolveThrottleConfig returns the throttle's current configuration:
// whatever ConfigureThrottle was last called with, or - if it was never
// called - the environment-derived configuration (resolved once, lazily,
// the first time this is reached with no explicit config in place).
func resolveThrottleConfig() ThrottleConfig {
	throttleMu.RLock()
	explicit := throttleExplicit
	cfg := throttleCfg
	throttleMu.RUnlock()

	if explicit {
		return cfg
	}

	throttleEnvOnce.Do(func() {
		envCfg := ThrottleConfig{
			Enabled: envBoolEnabled(os.Getenv("PCO_THROTTLE_ENABLED")),
			Window:  defaultThrottleWindow, // PCO_THROTTLE_RATE_PER_20S names its window explicitly
		}
		if v := os.Getenv("PCO_THROTTLE_RATE_PER_20S"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				envCfg.RatePerWindow = n
			}
		}
		if v := os.Getenv("PCO_THROTTLE_DEFAULT_PRIORITY"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				envCfg.DefaultPriority = n
			}
		}
		envCfg = applyThrottleDefaults(envCfg)

		throttleMu.Lock()
		// Only adopt the env-derived config if ConfigureThrottle hasn't won
		// the race in the meantime - an explicit call always wins.
		if !throttleExplicit {
			throttleCfg = envCfg
		}
		throttleMu.Unlock()
	})

	throttleMu.RLock()
	defer throttleMu.RUnlock()
	return throttleCfg
}

// envBoolEnabled matches this repo's existing boolean-env-var convention:
// "true" or "1" means enabled, anything else (including unset) means
// disabled.
func envBoolEnabled(v string) bool {
	return v == "true" || v == "1"
}

// throttlePriorityKey is the context.Context key WithPriority uses -
// unexported so nothing outside this package can collide with it.
type throttlePriorityKey struct{}

// WithPriority returns a context tagging every PCO call made with it (and
// any context derived from it) with a numeric priority, used only once the
// throttle is enabled (see ConfigureThrottle) - it's inert otherwise.
//
// Priority is numeric and lower numbers are admitted first: a call tagged
// priority 1 is admitted before a call tagged priority 20, and both are
// admitted before a call left at the untagged default (10, see
// ThrottleConfig.DefaultPriority). This is the same direction and the same
// default value (10) as WordPress's add_action/add_filter priority
// argument, deliberately - a widely known precedent beats an arbitrary
// house convention, and "priority 10, same as WordPress's default" is
// immediately legible to anyone who already knows that system.
//
// Never describe this convention in code or comments as "higher priority"
// or "lower priority" runs first - both phrases are genuinely ambiguous in
// English (does "higher priority" mean numerically higher, or more
// urgent?). Always state the number-to-order relationship concretely, the
// way this comment does, with an example.
//
// Example: an interactive, user-waiting call might use
// WithPriority(ctx, 1) so it's never stuck behind a large background sync
// sharing the same access token; an unattended background sync might use
// WithPriority(ctx, 20) so it never delays interactive traffic.
func WithPriority(ctx context.Context, priority int) context.Context {
	return context.WithValue(ctx, throttlePriorityKey{}, priority)
}

func priorityFromContext(ctx context.Context, defaultPriority int) int {
	if p, ok := ctx.Value(throttlePriorityKey{}).(int); ok {
		return p
	}
	return defaultPriority
}

// patLimiterKey is the fixed registry key used for every call
// authenticated by the PCO_CLIENT_ID/PCO_SECRET Personal Access Token
// (i.e. no per-user token present via WithAccessToken) - every such call
// shares that one credential's rate-limit budget with PCO, so they all
// share one tokenLimiter too. Chosen so it can never collide with a real
// OAuth access token string.
const patLimiterKey = "\x00pco-pat\x00"

var (
	throttleRegistryMu sync.Mutex
	throttleLimiters   = map[string]*tokenLimiter{}
)

// getTokenLimiter returns the tokenLimiter for the given registry key
// (an access token, or patLimiterKey), creating one on first use.
// Opportunistically sweeps out any other limiter that's been idle longer
// than cfg.IdleEvictAfter - simple last-used-timestamp eviction, no LRU
// library needed, since the number of distinct tokens a single process
// juggles at once is small.
func getTokenLimiter(key string, cfg ThrottleConfig) *tokenLimiter {
	throttleRegistryMu.Lock()
	defer throttleRegistryMu.Unlock()

	now := time.Now()
	for k, tl := range throttleLimiters {
		if k == key {
			continue
		}
		tl.mu.Lock()
		idle := now.Sub(tl.lastUsed)
		tl.mu.Unlock()
		if idle > cfg.IdleEvictAfter {
			delete(throttleLimiters, k)
		}
	}

	tl, ok := throttleLimiters[key]
	if !ok {
		tl = newTokenLimiter(cfg)
		throttleLimiters[key] = tl
	}
	return tl
}

// throttleAdmit resolves the calling context's access token (or the PAT
// fallback key) and priority, then blocks until that token's limiter
// admits the call or ctx is done - whichever happens first.
func throttleAdmit(ctx context.Context, cfg ThrottleConfig) error {
	key := throttleTokenKey(ctx)
	tl := getTokenLimiter(key, cfg)
	priority := priorityFromContext(ctx, cfg.DefaultPriority)
	return tl.admit(ctx, priority)
}

// throttleTokenKey mirrors the same precedence NewRequest itself uses for
// auth (see doRequest in pco.go): a context-carried access token if
// present, otherwise the fixed PAT key.
func throttleTokenKey(ctx context.Context) string {
	if token, ok := ctx.Value(accessTokenKey).(string); ok && token != "" {
		return token
	}
	return patLimiterKey
}

// throttleWaiter is one call waiting for admission in a tokenLimiter's
// queue. ch is closed exactly once, by whichever goroutine admits this
// waiter - the waiter itself is just watching for that close (or its own
// ctx.Done()).
type throttleWaiter struct {
	priority int
	seq      int64 // enqueue order, breaks priority ties FIFO
	ch       chan struct{}
	index    int // maintained by container/heap for O(log n) Remove
}

// waiterHeap is a container/heap.Interface ordering waiters by
// (priority, seq) - numerically smaller priority first, and within equal
// priority, earlier seq (i.e. FIFO) first. Equal-priority calls must never
// reorder relative to each other; seq is what guarantees that.
type waiterHeap []*throttleWaiter

func (h waiterHeap) Len() int { return len(h) }

func (h waiterHeap) Less(i, j int) bool {
	if h[i].priority != h[j].priority {
		return h[i].priority < h[j].priority
	}
	return h[i].seq < h[j].seq
}

func (h waiterHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *waiterHeap) Push(x any) {
	w := x.(*throttleWaiter)
	w.index = len(*h)
	*h = append(*h, w)
}

func (h *waiterHeap) Pop() any {
	old := *h
	n := len(old)
	w := old[n-1]
	old[n-1] = nil
	w.index = -1
	*h = old[:n-1]
	return w
}

// tokenLimiter admits calls for one access token (or the shared PAT key)
// in strict priority order, gated by a token-bucket capped at RatePerWindow
// tokens and refilling continuously toward that cap over Window.
//
// Deliberate accepted tradeoff: no anti-starvation. Admission is strict
// priority order with no aging/promotion of long-waiting low-priority
// calls - a low-priority call queued behind a steady stream of
// higher-priority ones can wait indefinitely. This is intentional, not an
// oversight: this SDK's actual use case is an unattended background sync
// competing with interactive, user-waiting traffic on the same token, and
// starving the background work under sustained interactive load is the
// *intended* outcome ("it'll just take longer"), not a bug. Do not add
// fairness/aging here without first re-reading the plan this was built
// from - revisit only if a real background job is observed never
// completing under sustained load in practice, not speculatively.
type tokenLimiter struct {
	mu   sync.Mutex
	heap waiterHeap

	nextSeq  int64
	lastUsed time.Time

	rate   int
	window time.Duration

	tokens     float64
	lastRefill time.Time
}

func newTokenLimiter(cfg ThrottleConfig) *tokenLimiter {
	now := time.Now()
	return &tokenLimiter{
		rate:       cfg.RatePerWindow,
		window:     cfg.Window,
		tokens:     float64(cfg.RatePerWindow), // start full, like a freshly-refilled bucket
		lastRefill: now,
		lastUsed:   now,
	}
}

// refillLocked adds tokens accrued since the last refill, capped at the
// bucket's capacity (rate). Must be called with mu held.
func (tl *tokenLimiter) refillLocked(now time.Time) {
	elapsed := now.Sub(tl.lastRefill)
	if elapsed <= 0 {
		return
	}
	tl.lastRefill = now

	tl.tokens += elapsed.Seconds() / tl.window.Seconds() * float64(tl.rate)
	if tl.tokens > float64(tl.rate) {
		tl.tokens = float64(tl.rate)
	}
}

// tryAdmitLocked refills, then admits as many head-of-queue waiters as the
// available token budget allows (in priority/FIFO order), closing each
// admitted waiter's channel. Must be called with mu held.
func (tl *tokenLimiter) tryAdmitLocked(now time.Time) {
	tl.refillLocked(now)
	for len(tl.heap) > 0 && tl.tokens >= 1 {
		w := heap.Pop(&tl.heap).(*throttleWaiter)
		tl.tokens--
		close(w.ch)
	}
}

// timeUntilNextTokenLocked returns how long until at least one more token
// is available, given the current (already-refilled) token count. Must be
// called with mu held, immediately after refillLocked.
func (tl *tokenLimiter) timeUntilNextTokenLocked() time.Duration {
	if tl.tokens >= 1 {
		return 0
	}
	need := 1 - tl.tokens
	wait := time.Duration(need / float64(tl.rate) * float64(tl.window))
	if wait <= 0 {
		wait = time.Millisecond
	}
	return wait
}

// admit blocks until this call is admitted (by this tokenLimiter's
// priority/rate rules) or ctx is done, whichever happens first. A
// cancelled/expired ctx returns promptly with ctx.Err(), never waiting for
// this call's actual turn.
func (tl *tokenLimiter) admit(ctx context.Context, priority int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	tl.mu.Lock()
	now := time.Now()
	tl.lastUsed = now
	w := &throttleWaiter{priority: priority, seq: tl.nextSeq, ch: make(chan struct{})}
	tl.nextSeq++
	heap.Push(&tl.heap, w)
	tl.tryAdmitLocked(now)
	wait := tl.timeUntilNextTokenLocked()
	tl.mu.Unlock()

	for {
		select {
		case <-w.ch:
			return nil
		case <-ctx.Done():
			tl.removeWaiter(w)
			return ctx.Err()
		case <-time.After(wait):
			tl.mu.Lock()
			now := time.Now()
			tl.lastUsed = now
			tl.tryAdmitLocked(now)
			wait = tl.timeUntilNextTokenLocked()
			tl.mu.Unlock()
		}
	}
}

// removeWaiter drops w from the heap if it's still queued (i.e. wasn't
// already admitted, in which case its heap index was already set to -1 by
// Pop) - used when a waiting caller's context is done before its turn
// arrives.
func (tl *tokenLimiter) removeWaiter(w *throttleWaiter) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	if w.index >= 0 && w.index < len(tl.heap) && tl.heap[w.index] == w {
		heap.Remove(&tl.heap, w.index)
	}
}
