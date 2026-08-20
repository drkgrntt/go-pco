package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetTeams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + teamsPath("1"); r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"Team","id":"1","attributes":{"name":"Band","schedule_to":"plan"}}]}`)
	})

	response, err := GetTeams(context.Background(), "1", &TeamsParams{OrderBy: "name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.Name != "Band" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
	if response.Data[0].Attributes.ScheduleTo != "plan" {
		t.Errorf("expected schedule_to plan, got %q", response.Data[0].Attributes.ScheduleTo)
	}
}

func TestGetTeamsNilParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params for nil TeamsParams, got %q", r.URL.RawQuery)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetTeams(context.Background(), "1", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
