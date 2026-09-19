package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetTags(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + tagGroupsPath + "/51232/tags"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[{
			"type": "Tag",
			"id": "233148",
			"attributes": {"name": "Chorus"},
			"relationships": {"tag_group": {"data": {"type": "TagGroup", "id": "51232"}}}
		}]}`)
	})

	response, err := GetTags(context.Background(), "51232", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(response.Data))
	}

	tag := response.Data[0]
	if tag.Attributes.Name != "Chorus" {
		t.Errorf("expected name Chorus, got %q", tag.Attributes.Name)
	}
	if tag.Relationships.TagGroup.Data == nil || tag.Relationships.TagGroup.Data.ID != "51232" {
		t.Errorf("unexpected tag_group relationship: %+v", tag.Relationships.TagGroup)
	}
}

func TestGetTagsBuildsQueryParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("per_page") != "50" {
			t.Errorf("expected per_page=50, got %q", q.Get("per_page"))
		}
		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetTags(context.Background(), "51232", &TagsParams{PerPage: 50}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTagsNilParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetTags(context.Background(), "51232", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetSongTags(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath + "/30823519/tags"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[{
			"type": "Tag",
			"id": "233148",
			"attributes": {"name": "Chorus"},
			"relationships": {"tag_group": {"data": {"type": "TagGroup", "id": "51232"}}}
		}]}`)
	})

	response, err := GetSongTags(context.Background(), "30823519")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].ID != "233148" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

// TestAssignSongTags asserts the exact request shape confirmed live -
// {"data":{"type":"TagGroup","relationships":{"tags":{"data":[...]}}}} -
// since this is the one place this endpoint doesn't follow the "obvious"
// JSON:API attribute shape (see AssignSongTags' own doc comment in tags.go
// for the two shapes that were tried and failed).
func TestAssignSongTags(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath + "/30823519/assign_tags"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		body := decodeBody(t, r)
		data, ok := body["data"].(map[string]any)
		if !ok {
			t.Fatalf("request body has no data object: %v", body)
		}
		if data["type"] != "TagGroup" {
			t.Errorf("expected data.type TagGroup, got %v", data["type"])
		}
		if _, hasAttrs := data["attributes"]; hasAttrs {
			t.Errorf("expected no data.attributes, got %v", data["attributes"])
		}

		rels := relationships(t, body)
		tagsRel, ok := rels["tags"].(map[string]any)
		if !ok {
			t.Fatalf("request body has no relationships.tags object: %v", rels)
		}
		tagsData, ok := tagsRel["data"].([]any)
		if !ok {
			t.Fatalf("request body has no relationships.tags.data array: %v", tagsRel)
		}
		if len(tagsData) != 2 {
			t.Fatalf("expected 2 tag refs, got %d", len(tagsData))
		}

		first := tagsData[0].(map[string]any)
		if first["type"] != "Tag" || first["id"] != "233148" {
			t.Errorf("unexpected first tag ref: %+v", first)
		}
		second := tagsData[1].(map[string]any)
		if second["type"] != "Tag" || second["id"] != "233151" {
			t.Errorf("unexpected second tag ref: %+v", second)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	if err := AssignSongTags(context.Background(), "30823519", []string{"233148", "233151"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestAssignSongTagsEmptyClearsGroup confirms an empty tagIDs slice still
// sends a (empty) tags.data array rather than omitting the relationship
// entirely - confirmed live this clears the group's assignment on the song
// rather than being rejected.
func TestAssignSongTagsEmptyClearsGroup(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(t, r)
		rels := relationships(t, body)
		tagsRel := rels["tags"].(map[string]any)
		tagsData, ok := tagsRel["data"].([]any)
		if !ok {
			t.Fatalf("expected relationships.tags.data to be present (even if empty), got %v", tagsRel)
		}
		if len(tagsData) != 0 {
			t.Errorf("expected empty tags.data, got %+v", tagsData)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	if err := AssignSongTags(context.Background(), "30823519", []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssignSongTagsNoContentNoError(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	if err := AssignSongTags(context.Background(), "30823519", []string{"233148"}); err != nil {
		t.Fatalf("expected no error on 204 No Content, got %v", err)
	}
}
