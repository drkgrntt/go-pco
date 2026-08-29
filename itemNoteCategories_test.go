package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetItemNoteCategories(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/item_note_categories"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"ItemNoteCategory","id":"1","attributes":{"name":"Band","frequently_used":true}}]}`)
	})

	response, err := GetItemNoteCategories(context.Background(), "st-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.Name != "Band" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}
