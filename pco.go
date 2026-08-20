package pco

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// baseURL is a var (not a const) so tests can point it at a local
// httptest.Server.
var baseURL = "https://api.planningcenteronline.com"

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

func NewRequest[Response interface{}](
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

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return response, err
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.SetBasicAuth(os.Getenv("PCO_CLIENT_ID"), os.Getenv("PCO_SECRET"))
	client := &http.Client{}
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
		return response, &RequestError{StatusCode: resp.StatusCode, Errors: errResp.Errors}
	}

	if len(responseBody) > 0 {
		err = json.Unmarshal(responseBody, &response)
	}

	return
}
