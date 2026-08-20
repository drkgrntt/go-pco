package pco

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// startTestServer spins up an httptest.Server, points the package-level
// baseURL at it for the duration of the test, and restores everything on
// cleanup.
func startTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(handler)
	original := baseURL
	baseURL = server.URL

	t.Cleanup(func() {
		baseURL = original
		server.Close()
	})

	return server
}

// decodeBody reads and JSON-decodes a request body inside a test handler.
func decodeBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("reading request body: %v", err)
	}

	body := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decoding request body %q: %v", raw, err)
		}
	}

	return body
}

// attributes pulls data.attributes out of a decoded JSON:API request body.
func attributes(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("request body has no data object: %v", body)
	}

	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("request body has no data.attributes object: %v", body)
	}

	return attrs
}

// relationships pulls data.relationships out of a decoded JSON:API request
// body.
func relationships(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("request body has no data object: %v", body)
	}

	rels, ok := data["relationships"].(map[string]any)
	if !ok {
		t.Fatalf("request body has no data.relationships object: %v", body)
	}

	return rels
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatalf("writing response body: %v", err)
	}
}

func TestQueryParamsEncode(t *testing.T) {
	if got := NewQueryParams().Encode(); got != "" {
		t.Fatalf("expected empty query for no params, got %q", got)
	}

	q := NewQueryParams().
		Where("first_name", "").
		Where("last_name", "Smith").
		PerPage(0).
		Offset(-1)

	got := q.Encode()
	if !strings.HasPrefix(got, "?") {
		t.Fatalf("expected query to start with '?', got %q", got)
	}
	if strings.Contains(got, "first_name") {
		t.Fatalf("expected empty Where value to be omitted, got %q", got)
	}
	if strings.Contains(got, "per_page") || strings.Contains(got, "offset") {
		t.Fatalf("expected non-positive PerPage/Offset to be omitted, got %q", got)
	}
	if !strings.Contains(got, "where%5Blast_name%5D=Smith") {
		t.Fatalf("expected where[last_name]=Smith to be encoded, got %q", got)
	}
}

func TestQueryParamsChaining(t *testing.T) {
	got := NewQueryParams().
		Include("emails", "phone_numbers").
		OrderBy("-created_at").
		PerPage(50).
		Offset(100).
		Encode()

	for _, want := range []string{"include=emails%2Cphone_numbers", "order=-created_at", "per_page=50", "offset=100"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected query %q to contain %q", got, want)
		}
	}
}

func TestNewRequestBody(t *testing.T) {
	body := NewRequestBody(map[string]any{"first_name": "Ada", "site_administrator": true})

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshaling request body: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshaling request body: %v", err)
	}

	data := decoded["data"].(map[string]any)
	attrs := data["attributes"].(map[string]any)

	if attrs["first_name"] != "Ada" {
		t.Errorf("expected first_name to be Ada, got %v", attrs["first_name"])
	}
	if attrs["site_administrator"] != true {
		t.Errorf("expected site_administrator to stay a bool, got %v (%T)", attrs["site_administrator"], attrs["site_administrator"])
	}
}

func TestRequestErrorMessage(t *testing.T) {
	noErrors := &RequestError{StatusCode: 500}
	if got := noErrors.Error(); !strings.Contains(got, "500") {
		t.Errorf("expected message to mention status code, got %q", got)
	}

	withDetail := &RequestError{
		StatusCode: 422,
		Errors:     []APIError{{Title: "Unprocessable", Detail: "first_name can't be blank"}},
	}
	got := withDetail.Error()
	if !strings.Contains(got, "422") || !strings.Contains(got, "first_name can't be blank") {
		t.Errorf("expected message to include status and detail, got %q", got)
	}

	titleOnly := &RequestError{StatusCode: 404, Errors: []APIError{{Title: "Not Found"}}}
	if got := titleOnly.Error(); !strings.Contains(got, "Not Found") {
		t.Errorf("expected message to fall back to title, got %q", got)
	}
}

func TestNewRequestSuccess(t *testing.T) {
	t.Setenv("PCO_CLIENT_ID", "test-id")
	t.Setenv("PCO_SECRET", "test-secret")

	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "test-id" || pass != "test-secret" {
			t.Errorf("expected basic auth test-id/test-secret, got %q/%q (ok=%v)", user, pass, ok)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		writeJSON(t, w, http.StatusOK, `{"greeting":"hello"}`)
	})

	type response struct {
		Greeting string `json:"greeting"`
	}

	got, err := NewRequest[response](context.Background(), http.MethodGet, baseURL+"/anything", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Greeting != "hello" {
		t.Errorf("expected greeting hello, got %q", got.Greeting)
	}
}

// TestNewRequestContextTokenTakesPrecedence confirms a context-carried
// access token (see WithAccessToken) is used as a Bearer token instead of
// the PCO_CLIENT_ID/PCO_SECRET PAT, even when both are available - a
// signed-in user's own requests should always act as them, not as whoever
// the PAT happens to belong to.
func TestNewRequestContextTokenTakesPrecedence(t *testing.T) {
	t.Setenv("PCO_CLIENT_ID", "test-id")
	t.Setenv("PCO_SECRET", "test-secret")

	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); ok {
			t.Error("expected no Basic Auth when a context token is present")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer user-token" {
			t.Errorf("expected Authorization: Bearer user-token, got %q", got)
		}
		writeJSON(t, w, http.StatusOK, `{}`)
	})

	ctx := WithAccessToken(context.Background(), "user-token")
	if _, err := NewRequest[struct{}](ctx, http.MethodGet, baseURL+"/anything", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestNewRequestFallsBackToPATWithoutContextToken confirms the PAT is still
// used when no context token is present - existing scripts/tests that never
// call WithAccessToken keep working unchanged.
func TestNewRequestFallsBackToPATWithoutContextToken(t *testing.T) {
	t.Setenv("PCO_CLIENT_ID", "test-id")
	t.Setenv("PCO_SECRET", "test-secret")

	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "test-id" || pass != "test-secret" {
			t.Errorf("expected basic auth test-id/test-secret, got %q/%q (ok=%v)", user, pass, ok)
		}
		writeJSON(t, w, http.StatusOK, `{}`)
	})

	if _, err := NewRequest[struct{}](context.Background(), http.MethodGet, baseURL+"/anything", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRequestSetsContentTypeOnlyWithBody(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.Header.Get("Content-Type") != "" {
			t.Errorf("did not expect Content-Type on a bodyless GET, got %q", r.Header.Get("Content-Type"))
		}
		if r.Method == http.MethodPost && r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json on POST with body, got %q", r.Header.Get("Content-Type"))
		}
		writeJSON(t, w, http.StatusOK, `{}`)
	})

	if _, err := NewRequest[struct{}](context.Background(), http.MethodGet, baseURL+"/no-body", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := NewRequest[struct{}](context.Background(), http.MethodPost, baseURL+"/with-body", map[string]string{"a": "b"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRequestEmptyBodyNoError(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := NewRequest[struct{}](context.Background(), http.MethodDelete, baseURL+"/thing/1", nil); err != nil {
		t.Fatalf("expected no error on empty 204 body, got %v", err)
	}
}

func TestNewRequestErrorStatus(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusUnprocessableEntity, `{"errors":[{"status":"422","title":"Unprocessable Entity","detail":"name can't be blank"}]}`)
	})

	_, err := NewRequest[struct{}](context.Background(), http.MethodPost, baseURL+"/thing", map[string]string{})
	if err == nil {
		t.Fatal("expected an error for a 422 response")
	}

	reqErr, ok := err.(*RequestError)
	if !ok {
		t.Fatalf("expected *RequestError, got %T", err)
	}
	if reqErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", reqErr.StatusCode)
	}
	if len(reqErr.Errors) != 1 || reqErr.Errors[0].Detail != "name can't be blank" {
		t.Errorf("expected parsed error detail, got %+v", reqErr.Errors)
	}
}
