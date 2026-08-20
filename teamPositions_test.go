package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetTeamPositions(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + teamPositionsPath("1"); r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"TeamPosition","id":"1","attributes":{"name":"Drums","sequence":1}}]}`)
	})

	response, err := GetTeamPositions(context.Background(), "1", &TeamPositionsParams{OrderBy: "name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.Name != "Drums" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestGetTeamPositionsNilParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params for nil TeamPositionsParams, got %q", r.URL.RawQuery)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetTeamPositions(context.Background(), "1", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
