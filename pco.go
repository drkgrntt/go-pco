package pco

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// baseURL is a var (not a const) so tests can point it at a local
// httptest.Server.
var baseURL = "https://api.planningcenteronline.com"

// requestTimeout bounds every call NewRequest makes. Without it, a hung PCO
// response (rare, but the previous *http.Client{} had no Timeout at all)
// hangs the caller's request indefinitely, since http.NewRequestWithContext
// only enforces whatever deadline the caller's own context carries - which
// most callers (an HTTP handler, a page render) don't set. Not currently
// configurable: no caller has needed anything other than "generous enough
// for a slow PCO response, short enough to fail a render rather than hang
// it" - revisit only if a real use case needs a different value.
const requestTimeout = 30 * time.Second

// SetBaseURLForTesting points every subsequent call in this package at url
// instead of the real PCO API - e.g. a local httptest.Server standing in
// for PCO, so an importing package's tests can exercise fetch/pagination/
// concurrency/pacing behavior with no real network call or credentials
// involved. Returns a restore func that puts the original base URL back;
// call it via defer so a failure to restore doesn't leak into later tests.
//
// This package's own tests reach the unexported baseURL var directly (see
// startTestServer in pco_test.go); this exported wrapper exists purely so
// an importer - which can't see the unexported var - has the same seam.
// Test-only: never call this from production code.
func SetBaseURLForTesting(url string) (restore func()) {
	original := baseURL
	baseURL = url
	return func() {
		baseURL = original
	}
}

// accessTokenKey is the context.Context key WithAccessToken/NewRequest use
// to pass a per-request OAuth Bearer token - unexported so nothing outside
// this package can collide with it or read it directly.
type contextKey int

const accessTokenKey contextKey = iota

// WithAccessToken returns a context carrying a per-request OAuth access
// token - when present, NewRequest authenticates with it (Authorization:
// Bearer <token>) instead of the PCO_CLIENT_ID/PCO_SECRET Personal Access
// Token. This is how a multi-user caller (each signed-in user making
// requests as themselves, against their own PCO organization and
// permissions) differs from a single-tenant script using the PAT - thread
// this context through instead of the PAT for anything user-scoped.
func WithAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenKey, token)
}

type Meta struct {
	CanFilter  []string `json:"can_filter"`
	CanInclude []string `json:"can_include"`
	CanOrderBy []string `json:"can_order_by"`
	CanQueryBy []string `json:"can_query_by"`
	Count      int      `json:"count"`
	Parent     General  `json:"parent"`
	TotalCount int      `json:"total_count"`
}

type General struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// HasOneRelationship is the JSON:API shape of a to-one relationship, e.g.
// {"data": {"type": "Campus", "id": "1"}} or {"data": null}.
type HasOneRelationship struct {
	Data *General `json:"data"`
}

// HasManyRelationship is the JSON:API shape of a to-many relationship, e.g.
// {"data": [{"type": "Email", "id": "1"}, ...]}. This is the shape PCO uses
// for relationships whose resources you can pull in with ?include=, such as
// a person's emails/addresses/phone_numbers.
type HasManyRelationship struct {
	Data []General `json:"data"`
}

// Links appears on both single-resource responses (Self/HTML) and
// collection responses (Self/Next/Prev), so all four fields are optional
// depending on which kind of response it was unmarshaled from.
type Links struct {
	Self string `json:"self"`
	HTML string `json:"html,omitempty"`
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

// RequestData is the JSON:API "data" object sent in create/update request
// bodies. Attributes/Relationships use `any` values so non-string fields
// (bool, int, arrays) are encoded correctly instead of being forced into
// strings.
type RequestData struct {
	Type          string         `json:"type,omitempty"`
	ID            string         `json:"id,omitempty"`
	Attributes    map[string]any `json:"attributes,omitempty"`
	Relationships map[string]any `json:"relationships,omitempty"`
}

type RequestBody struct {
	Data RequestData `json:"data"`
}

// NewRequestBody wraps a set of attributes in the JSON:API body shape
// expected by PCO's create/update endpoints.
func NewRequestBody(attributes map[string]any) RequestBody {
	return RequestBody{Data: RequestData{Attributes: attributes}}
}

// NewRequestBodyWithRelationships is like NewRequestBody but also sets the
// JSON:API `relationships` object, for creates/updates that need to link
// another resource (e.g. an Item linked to a library Song) rather than just
// set plain attributes. Each value in relationships should be shaped like
// `map[string]any{"data": map[string]any{"type": "...", "id": "..."}}`
// (or a nil/omitted "data" to clear a to-one relationship).
func NewRequestBodyWithRelationships(attributes map[string]any, relationships map[string]any) RequestBody {
	return RequestBody{Data: RequestData{Attributes: attributes, Relationships: relationships}}
}

// APIError is a single JSON:API error object, as returned in the top-level
// "errors" array on 4xx/5xx responses.
type APIError struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// RequestError is returned by NewRequest whenever PCO responds with a
// non-2xx status. It carries the parsed JSON:API errors when the response
// body included any.
type RequestError struct {
	StatusCode int
	Errors     []APIError
	// RetryAfter is the response's Retry-After header, verbatim, when PCO
	// sent one - normally only present on a 429 (rate limit exceeded), per
	// https://api.planningcenteronline.com/docs/overview/rate-limiting.
	// Empty when the header was absent. Left as the raw header string
	// (PCO sends a delay-in-seconds value, never an HTTP-date) rather than
	// parsed into a time.Duration, since a caller that doesn't need it
	// shouldn't pay for parsing it, and one that does can call
	// strconv.Atoi itself.
	RetryAfter string
}

func (e *RequestError) Error() string {
	if len(e.Errors) == 0 {
		return fmt.Sprintf("pco: request failed with status %d", e.StatusCode)
	}

	details := make([]string, len(e.Errors))
	for i, apiErr := range e.Errors {
		if apiErr.Detail != "" {
			details[i] = apiErr.Detail
		} else {
			details[i] = apiErr.Title
		}
	}

	return fmt.Sprintf("pco: request failed with status %d: %s", e.StatusCode, strings.Join(details, "; "))
}

// QueryParams builds PCO's `where[]`, `include`, `order`, `per_page`, and
// `offset` query string parameters.
type QueryParams struct {
	values url.Values
}

func NewQueryParams() *QueryParams {
	return &QueryParams{values: url.Values{}}
}

// Where adds a `where[field]=value` filter. Empty values are ignored so
// callers can chain calls with optional params unconditionally.
func (q *QueryParams) Where(field, value string) *QueryParams {
	if value != "" {
		q.values.Set(fmt.Sprintf("where[%s]", field), value)
	}
	return q
}

func (q *QueryParams) Include(fields ...string) *QueryParams {
	if len(fields) > 0 {
		q.values.Set("include", strings.Join(fields, ","))
	}
	return q
}

func (q *QueryParams) OrderBy(field string) *QueryParams {
	if field != "" {
		q.values.Set("order", field)
	}
	return q
}

// Filter adds PCO's top-level `filter=` param - distinct from Where's
// `where[field]=`. Some endpoints document named filters here (e.g. Plans'
// "past"/"future") rather than as where[] filters.
func (q *QueryParams) Filter(value string) *QueryParams {
	if value != "" {
		q.values.Set("filter", value)
	}
	return q
}

func (q *QueryParams) PerPage(n int) *QueryParams {
	if n > 0 {
		q.values.Set("per_page", strconv.Itoa(n))
	}
	return q
}

func (q *QueryParams) Offset(n int) *QueryParams {
	if n > 0 {
		q.values.Set("offset", strconv.Itoa(n))
	}
	return q
}

// Encode returns the query string, including a leading "?" when non-empty
// (or "" when there are no params, so it's always safe to append to a URL).
func (q *QueryParams) Encode() string {
	if q == nil || len(q.values) == 0 {
		return ""
	}
	return "?" + q.values.Encode()
}

// NewRequest is the one place every resource function in this package
// eventually calls to actually talk to PCO. Three independent, opt-in
// pieces wrap the actual HTTP call in doRequest below, each a no-op until
// a consumer turns it on:
//   - the priority-aware request throttle (throttle.go /
//     ConfigureThrottle / PCO_THROTTLE_* env vars) - an admission wait in
//     front of doRequest, scoped per access token;
//   - the 429 retry (retry.go / ConfigureRetry / PCO_RETRY_ON_429_* env
//     vars) - on a 429, wait and call doRequest again, up to a bounded
//     number of attempts;
//   - the response hook (observe.go / SetResponseHook) - every attempt,
//     successful or not, is reported for logging/metrics before this
//     returns.
//
// With all three left off (the default for a consumer that never opts
// into any of them), this is one doRequest call and a couple of cheap
// config reads/RLocks - no behavior change from before any of the three
// existed.
func NewRequest[Response interface{}](
	ctx context.Context,
	method string,
	url string,
	body interface{},
) (response Response, err error) {
	tCfg := resolveThrottleConfig()
	rCfg := resolveRetryConfig()

	maxAttempts := 1
	if rCfg.Enabled {
		maxAttempts = rCfg.MaxAttempts
	}

	for attempt := 1; ; attempt++ {
		start := time.Now()

		var throttleWait time.Duration
		if tCfg.Enabled {
			admitStart := time.Now()
			if admitErr := throttleAdmit(ctx, tCfg); admitErr != nil {
				callResponseHook(buildResponseInfo(ctx, method, url, admitErr, attempt, time.Since(admitStart), time.Since(start)))
				return response, admitErr
			}
			throttleWait = time.Since(admitStart)
		}

		response, err = doRequest[Response](ctx, method, url, body)
		callResponseHook(buildResponseInfo(ctx, method, url, err, attempt, throttleWait, time.Since(start)))

		if err == nil || attempt >= maxAttempts {
			return response, err
		}

		reqErr, ok := err.(*RequestError)
		if !ok || reqErr.StatusCode != 429 {
			return response, err // only a 429 is ever retried - see retry.go's doc comment for why
		}

		select {
		case <-ctx.Done():
			return response, ctx.Err()
		case <-time.After(retryWaitFor(reqErr.RetryAfter, rCfg)):
		}
	}
}

// doRequest is NewRequest's actual HTTP call, unchanged from what
// NewRequest itself did before the throttle/retry/hook existed - splitting
// it out keeps this the one place a future addition to NewRequest's own
// wrapping logic can't accidentally touch the request-building code
// itself.
func doRequest[Response interface{}](
	ctx context.Context,
	method string,
	url string,
	body interface{},
) (response Response, err error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return response, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return response, err
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// A context-carried per-user token (see WithAccessToken) takes
	// precedence over the PAT - a signed-in user's own requests should
	// always act as them, not as whoever's PCO_CLIENT_ID/PCO_SECRET happen
	// to be configured. Falling back to the PAT keeps standalone scripts
	// and this package's own tests working without needing a token.
	if token, ok := ctx.Value(accessTokenKey).(string); ok && token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		req.SetBasicAuth(os.Getenv("PCO_CLIENT_ID"), os.Getenv("PCO_SECRET"))
	}

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Errors []APIError `json:"errors"`
		}
		_ = json.Unmarshal(responseBody, &errResp)
		return response, &RequestError{
			StatusCode: resp.StatusCode,
			Errors:     errResp.Errors,
			RetryAfter: resp.Header.Get("Retry-After"),
		}
	}

	if len(responseBody) > 0 {
		err = json.Unmarshal(responseBody, &response)
	}

	return
}
