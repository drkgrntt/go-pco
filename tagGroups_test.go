package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetTagGroups(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + tagGroupsPath; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[{
			"type": "TagGroup",
			"id": "51232",
			"attributes": {
				"allow_multiple_selections": true,
				"name": "Type",
				"required": false,
				"service_type_folder_name": null,
				"tags_for": "song"
			},
			"relationships": {
				"tags": {"data": [{"type": "Tag", "id": "233148"}, {"type": "Tag", "id": "233151"}]}
			}
		}]}`)
	})

	response, err := GetTagGroups(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 {
		t.Fatalf("expected 1 tag group, got %d", len(response.Data))
	}

	group := response.Data[0]
	attrs := group.Attributes
	if !attrs.AllowMultipleSelections || attrs.Required {
		t.Errorf("unexpected attributes: %+v", attrs)
	}
	if attrs.Name != "Type" {
		t.Errorf("expected name Type, got %q", attrs.Name)
	}
	if attrs.ServiceTypeFolderName != nil {
		t.Errorf("expected nil ServiceTypeFolderName, got %v", *attrs.ServiceTypeFolderName)
	}
	if attrs.TagsFor != "song" {
		t.Errorf("expected tags_for song, got %q", attrs.TagsFor)
	}

	if len(group.Relationships.Tags.Data) != 2 {
		t.Fatalf("expected 2 related tags, got %d", len(group.Relationships.Tags.Data))
	}
	if group.Relationships.Tags.Data[0].ID != "233148" || group.Relationships.Tags.Data[1].ID != "233151" {
		t.Errorf("unexpected related tag ids: %+v", group.Relationships.Tags.Data)
	}
}

func TestGetTagGroupsBuildsQueryParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("include") != "tags" {
			t.Errorf("expected include=tags, got %q", q.Get("include"))
		}
		if q.Get("per_page") != "100" {
			t.Errorf("expected per_page=100, got %q", q.Get("per_page"))
		}

		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetTagGroups(context.Background(), &TagGroupsParams{Include: []string{"tags"}, PerPage: 100}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTagGroupsNilParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params for nil TagGroupsParams, got %q", r.URL.RawQuery)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetTagGroups(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTagGroup(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + tagGroupsPath + "/51232"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"TagGroup","id":"51232","attributes":{"name":"Type","tags_for":"song"}}}`)
	})

	response, err := GetTagGroup(context.Background(), "51232")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "51232" || response.Data.Attributes.Name != "Type" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}
