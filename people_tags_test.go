package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetPersonTags(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/services/v2/people/12345/tags"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[{
			"type": "Tag",
			"id": "233148",
			"attributes": {"name": "Electric Guitar"},
			"relationships": {"tag_group": {"data": {"type": "TagGroup", "id": "2918723"}}}
		}]}`)
	})

	response, err := GetPersonTags(context.Background(), "12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].ID != "233148" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
	if response.Data[0].Relationships.TagGroup.Data == nil || response.Data[0].Relationships.TagGroup.Data.ID != "2918723" {
		t.Errorf("unexpected tag_group relationship: %+v", response.Data[0].Relationships.TagGroup)
	}
}
