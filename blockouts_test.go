package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetBlockouts(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + blockoutsPath("5"); r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("filter") != "future" {
			t.Errorf("expected filter=future, got %q", q.Get("filter"))
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{
			"type": "Blockout",
			"id": "1",
			"attributes": {
				"reason": "Vacation",
				"starts_at": "2026-09-19T00:00:00Z",
				"ends_at": "2026-09-21T23:59:59Z"
			}
		}]}`)
	})

	response, err := GetBlockouts(context.Background(), "5", &BlockoutsParams{Filter: "future"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.Reason != "Vacation" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestGetBlockoutsNilParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params for nil BlockoutsParams, got %q", r.URL.RawQuery)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetBlockouts(context.Background(), "5", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
