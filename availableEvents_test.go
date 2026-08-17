package pco

import (
	"net/http"
	"testing"
)

func TestGetAvailableEvents(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + availableEventsPath; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Errorf("expected per_page=100, got %q", got)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"AvailableEvent","id":"1","attributes":{"name":"people.v2.events.person.created","app":"people"}}]}`)
	})

	response, err := GetAvailableEvents(&AvailableEventsParams{PerPage: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.Name != "people.v2.events.person.created" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestGetAvailableEventsNilParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetAvailableEvents(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
