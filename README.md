# go-pco

An unofficial Go SDK for the [Planning Center](https://www.planningcenter.com/) API. Not affiliated with or endorsed by Planning Center.

This is a work in progress, built out resource-by-resource as needed rather than generated from a spec. See [Status & roadmap](#status--roadmap) for what's covered today.

## Installation

```sh
go get github.com/drkgrntt/go-pco
```

Requires Go 1.26+ (generics are used throughout).

## Authentication

Every request is signed with HTTP Basic Auth using two environment variables:

```sh
export PCO_CLIENT_ID=your_application_id
export PCO_SECRET=your_secret
```

These can be a [Personal Access Token](https://developer.planning.center/docs/#/overview/authentication) (simplest, good for scripts/single-org use — the "app ID" and "secret" PCO gives you when you create a PAT) or an OAuth application's client credentials. This is the fallback auth every call uses when nothing more specific is provided.

For a real multi-user app - each signed-in person's requests acting as them, against their own PCO organization and permissions, rather than everyone sharing one PAT - carry their OAuth access token through `context.Context` instead:

```go
ctx := pco.WithAccessToken(context.Background(), userAccessToken)
people, err := pco.GetPeople(ctx, &pco.PeopleParams{LastName: "Lovelace"})
```

When a context carries a token this way, every function authenticates with `Authorization: Bearer <token>` instead of the PAT - per call, not per-process, so concurrent requests for different users on the same server never cross streams. The SDK doesn't handle obtaining that token itself (the OAuth authorization-code flow, or refreshing an expired one) - that's still on the caller.

## Quick start

```go
package main

import (
	"fmt"
	"log"

	"github.com/drkgrntt/go-pco"
)

func main() {
	people, err := pco.GetPeople(ctx, &pco.PeopleParams{LastName: "Lovelace"})
	if err != nil {
		log.Fatal(err)
	}

	for _, person := range people.Data {
		fmt.Println(person.Attributes.FirstName, person.Attributes.LastName)
	}
}
```

## Core concepts

Every resource file follows the same shape: an `Attributes` struct, a `Relationships` struct, a `*Data` struct combining them, response envelope struct(s) (`*Response` for a single resource, `*ListResponse` for a collection), a `*Params` struct for inputs, and a set of plain functions (not methods) that call `pco.NewRequest`. The pieces below (from [pco.go](pco.go)) are shared by all of them.

### Errors

Any non-2xx response is returned as a `*pco.RequestError`, which carries the parsed JSON:API `errors[]` array from the response body:

```go
_, err := pco.CreatePerson(ctx, &pco.CreatePersonParams{})
var reqErr *pco.RequestError
if errors.As(err, &reqErr) {
	fmt.Println(reqErr.StatusCode) // e.g. 422
	for _, apiErr := range reqErr.Errors {
		fmt.Println(apiErr.Title, apiErr.Detail)
	}
}
```

### Pagination

List params structs (`PeopleParams`, `PlansParams`, etc.) accept `PerPage` and `Offset`. Every list response's `Links.Next` is set to the next page's URL when there is one:

```go
params := &pco.PeopleParams{PerPage: 100}
for {
	page, err := pco.GetPeople(ctx, params)
	if err != nil {
		log.Fatal(err)
	}
	// ...use page.Data...

	if page.Links.Next == "" {
		break
	}
	page, err = pco.NewRequest[pco.PersonListResponse](ctx, "GET", page.Links.Next, nil)
	if err != nil {
		log.Fatal(err)
	}
	params = nil // page.Links.Next already has offset/per_page baked in
}
```

### Filtering & querying

Internally, params get turned into `where[]`/`include`/`order`/`per_page`/`offset` query params via `pco.QueryParams`. You won't normally build one of these yourself — it's exposed mainly so you can call `pco.NewRequest` directly against an endpoint this SDK doesn't wrap yet:

```go
q := pco.NewQueryParams().Where("first_name", "Ada").Include("emails").PerPage(25)
response, err := pco.NewRequest[pco.PersonListResponse](ctx, "GET", "https://api.planningcenteronline.com/people/v2/people"+q.Encode(), nil)
```

### Request bodies

Create/update functions build their body with `pco.NewRequestBody(map[string]any{...})`, which wraps attributes in the `{"data":{"attributes":{...}}}` shape PCO's JSON:API expects. `any` values are used (not `string`) so booleans, numbers, and arrays serialize correctly. `pco.NewRequestBodyWithRelationships(attributes, relationships)` is the same, plus a `{"data":{"relationships":{...}}}` object, for creates/updates that link another resource (e.g. an Item linked to a Song) rather than just set plain attributes.

### Request throttle (opt-in)

PCO enforces a rate limit of 100 requests per 20 seconds, **per authenticated user token** (see [PCO's rate-limiting docs](https://api.planningcenteronline.com/docs/overview/rate-limiting)). go-pco itself never throttles requests unless you turn it on - by default it's a plain "make HTTP requests to PCO" library, and stays that way for any consumer who doesn't opt in.

Opting in gets you two things at once: a global cap on sustained throughput per token, and a way to say some calls matter more than others when there's contention for that budget.

**Opt in one of two ways** - an explicit call always wins if both are present:

```go
// In code, e.g. once at process startup:
pco.ConfigureThrottle(pco.ThrottleConfig{
	Enabled: true,
	// RatePerWindow/Window/DefaultPriority/IdleEvictAfter are all optional -
	// zero values fall back to package defaults (see below).
})
```

```sh
# Or via environment variable, no code change needed:
export PCO_THROTTLE_ENABLED=true
export PCO_THROTTLE_RATE_PER_20S=60     # optional, defaults to 60
export PCO_THROTTLE_DEFAULT_PRIORITY=10 # optional, defaults to 10
```

The env vars are only consulted if `ConfigureThrottle` is never called (checked once, lazily, on first use) - a code-level `ConfigureThrottle` call always takes precedence.

**What the defaults mean in practice**: once enabled, each access token (or, for PAT-authenticated calls with no per-user token, one shared bucket for the PAT itself) is limited to `RatePerWindow` requests per `Window`, refilled continuously rather than reset in hard steps. The shipped default is **60 requests per 20 seconds** - deliberately more conservative than PCO's raw 100/20s ceiling, chosen to match the same 60%-headroom figure already proven safe elsewhere rather than starting closer to the real limit and hoping it holds up; it's meant to be raised later with real production signal, not lowered after the fact. A call that never sets a priority is admitted at `DefaultPriority` (10).

**Priority convention** - read this carefully, the direction is easy to get backwards: priority is numeric, and **lower numbers are admitted first**. A call tagged priority `1` is admitted before a call tagged priority `10`, which is admitted before a call tagged priority `20`. This is the same direction and the same default value (10) as WordPress's `add_action`/`add_filter` priority argument, deliberately - if you already know that convention, "priority 10, same as WordPress's default" tells you everything. Tag a context with `pco.WithPriority`:

```go
ctx := pco.WithPriority(context.Background(), 1) // admitted before the default (10)
people, err := pco.GetPeople(ctx, &pco.PeopleParams{})
```

Never describe this as "higher priority" or "lower priority" wins - both phrases are genuinely ambiguous (does "higher priority" mean numerically higher, or more urgent?). Always state it as "admitted before/after," the way this section does.

**Motivating example** - a background sync sharing an access token with interactive traffic:

```go
// Interactive, user-waiting request: leave it at the default, or tag it
// explicitly with a low number so it's never stuck behind background work.
people, err := pco.GetPeople(ctx, &pco.PeopleParams{})

// Unattended background sync on the same token: tag it with a high
// number so it never delays the interactive request above.
syncCtx := pco.WithPriority(ctx, 20)
songs, err := pco.GetSongs(syncCtx, &pco.SongsParams{})
```

**Deliberate tradeoff - no anti-starvation**: admission is strict priority order, with no aging or promotion of a long-waiting low-priority call. A background sync queued behind a steady stream of interactive traffic on the same token can wait a long time - that's the intended outcome for this SDK's actual use case (an unattended sync competing with user-waiting requests), not a bug to be fixed. Don't add fairness/aging logic without understanding this was a deliberate design choice.

Other notes: idle per-token limiters (no activity for `IdleEvictAfter`, default 30 minutes) are dropped from the internal registry so a long-running process doesn't accumulate state for users who've stopped making requests. A context whose `Done()` fires while a call is queued returns that context's error promptly - it never waits for its turn to come up.

**A note on what the throttle can't fully guarantee**: it bounds *sustained* rate, not every possible burst pattern. Its token bucket starts full and refills continuously, so a same-token burst that lands right as the bucket refills (drained at the start of one window, refilled again by the start of the next) can, in an unlucky worst case, still get a 429 from PCO even with the throttle on. That's exactly what the 429 retry below exists to catch.

### 429 retry (opt-in)

A caller can still get a real 429 from PCO even with the throttle enabled (see the note just above), or with the throttle off entirely. The retry is a separate, independent opt-in: on a 429, wait and try the same call again, up to a bounded number of attempts.

**Opt in the same two ways as the throttle** - an explicit call always wins if both are present:

```go
pco.ConfigureRetry(pco.RetryConfig{
	Enabled: true,
	// MaxAttempts/MaxWait/FallbackWait are all optional - zero values
	// fall back to package defaults (see below).
})
```

```sh
export PCO_RETRY_ON_429_ENABLED=true
export PCO_RETRY_ON_429_MAX_ATTEMPTS=2       # optional, defaults to 2 (one retry)
export PCO_RETRY_ON_429_MAX_WAIT_SECONDS=30  # optional, defaults to 30
```

**What happens on a 429** with the retry enabled: `NewRequest` waits, then calls PCO again, for up to `MaxAttempts` total tries. How long it waits: PCO's own `Retry-After` header when the response sent one (PCO always sends a delay in seconds, never an HTTP-date), or `FallbackWait` (default 5s) when it didn't - either way, never longer than `MaxWait` (default 30s), so a caller with its own deadline (a user-facing HTTP handler mid-render) isn't stuck waiting on whatever PCO asked for. A context whose `Done()` fires while waiting to retry returns that context's error promptly, same as the throttle.

**Deliberately narrow**: only a 429 is ever retried. A 422 or other 4xx means the request was built wrong (bad params, a resource that doesn't exist) - retrying it wastes a call and gets the same error again. A 5xx might be transient, but retrying every 5xx blind is a different, broader feature (write idempotency matters there in a way it doesn't for "wait out a rate limit") that this package doesn't attempt.

### Response hook (opt-in)

`SetResponseHook` registers a function called after every `NewRequest` attempt (success or failure, including each attempt of a retried call) - the one place to observe every request this package makes (log a 429, track latency) without wrapping each of the ~40 resource functions individually:

```go
pco.SetResponseHook(func(info pco.ResponseInfo) {
	if info.StatusCode == http.StatusTooManyRequests {
		log.Printf("PCO rate limited: %s %s (attempt %d, retry-after %s)", info.Method, info.URL, info.Attempt, info.RetryAfter)
	}
})
```

`ResponseInfo` carries `Ctx`/`Method`/`URL`/`Err`/`StatusCode`/`RetryAfter`/`Attempt`/`ThrottleWait`/`Duration` - see its doc comment in [observe.go](observe.go) for exactly what each field means and when it's zero. `Ctx` is the exact `context.Context` the call was made with, untouched - a consumer that stashes its own values on the context it passes to a `pco.Get*`/`pco.Create*`/etc. call (a request id, a trace span) can read them back out here, e.g. `log.Printf("... request_id=%s", info.Ctx.Value(myRequestIDKey{}))`, without this package needing to know what those values are. Purely observational: the hook's return value (there isn't one) can't change whether a call succeeds, retries, or what's returned to its caller - use `ConfigureRetry` above to change behavior on a 429, not this. Runs synchronously on the calling goroutine, so keep it fast (a log call, a metrics increment) - it directly adds to every request's latency.

## Modules

| Module | Docs | Covers |
|---|---|---|
| People | [docs/people.md](docs/people.md) | Organization, People, Addresses, Emails, Phone Numbers |
| Webhooks | [docs/webhooks.md](docs/webhooks.md) | Subscriptions, Available Events, Event deliveries, receiving & verifying incoming webhooks |
| Services | [docs/services.md](docs/services.md) | Service Types, Plans, Items, Item Notes, Item Note Categories, Songs, Arrangements, Keys, Teams, Team Positions, Needed Positions, Team Members (PlanPerson), Person Team Position Assignments, Blockouts |

## Testing

```sh
go test ./...
```

Tests run against a local `httptest.Server` — no live PCO account or credentials needed. Each test file mirrors the resource file it covers (e.g. `plans_test.go` tests `plans.go`) and asserts the HTTP method, path, query string, and/or request body the function produced, plus that the response unmarshals correctly. `pco_test.go` holds the shared harness (`startTestServer`, `decodeBody`, `attributes`, `relationships`, `writeJSON`) and the core `NewRequest`/`QueryParams`/error-handling tests.

## Extending

To add a new resource, copy the shape of an existing file with a similar level of nesting — [serviceTypes.go](serviceTypes.go) for a top-level resource, [plans.go](plans.go) or [items.go](items.go) for one nested under another. In outline:

1. `XAttributes` / `XRelationships` structs matching the resource's documented attributes/relationships.
2. `XData` combining them with `Type`/`ID`.
3. `XResponse` (single) and `XListResponse` (collection) envelopes — `Data`, `Included []any`, `Links`, `Meta`.
4. An `XParams` struct for list filters (`PerPage`/`Offset`, plus any `where[]` filters) and separate `CreateXParams`/`UpdateXParams` structs for only the attributes that endpoint documents as assignable.
5. `GetXs`, `GetX`, `CreateX`, `UpdateX`, `DeleteX` functions following the existing naming pattern, using `pco.NewQueryParams()` / `pco.NewRequestBody()` / `pco.NewRequest[...]`.
6. A matching `_test.go` asserting method, path, query/body construction, and response decoding — see any existing `*_test.go` for the pattern.

Look up the resource's attributes/relationships/actions at `https://api.planningcenteronline.com/docs/apps/<product>/versions/<version>/vertices/<resource>` before writing the struct — don't guess field names, PCO's docs are the source of truth.

## Status & roadmap

Implemented: People (Organization, People, Addresses, Emails, Phone Numbers), Webhooks (full), Services (Service Types, Plans, Items - full CRUD plus reorder; Item Notes - full CRUD; Item Note Categories - read-only; Songs - create/read/delete; Arrangements - create/read/update; Keys - create/read; Teams, Team Positions - read-only; Needed Positions - read-only, see below; Team Members/PlanPerson - full CRUD; Person Team Position Assignments - full CRUD; Blockouts - read-only by design).

Not yet implemented, in roughly the order they'd likely matter for a Services-focused integration:
- Services: Needed Position create/update/delete (every request shape tried 500s live on a real account regardless of body - see [docs/services.md](docs/services.md#needed-positions)), Song update, Schedules, Attachments, and the ~40 remaining resources listed under `/docs/apps/services`.
- Other PCO products entirely: Giving, Check-Ins, Calendar, Groups, Publishing, Registrations, Resources.

Pull requests / additions following the pattern in [Extending](#extending) are the intended way this grows.
