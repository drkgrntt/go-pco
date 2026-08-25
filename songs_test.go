package pco

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestGetSongs(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		q := r.URL.Query()
		if q.Get("order") != "-created_at" {
			t.Errorf("expected order=-created_at, got %q", q.Get("order"))
		}
		if q.Get("per_page") != "10" {
			t.Errorf("expected per_page=10, got %q", q.Get("per_page"))
		}

		writeJSON(t, w, http.StatusOK, `{"data":[{
			"type": "Song",
			"id": "1",
			"attributes": {
				"title": "Holy Forever",
				"author": "Brian Johnson",
				"ccli_number": 7201044,
				"hidden": false,
				"last_scheduled_at": null
			}
		}]}`)
	})

	response, err := GetSongs(context.Background(), &SongsParams{OrderBy: "-created_at", PerPage: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 {
		t.Fatalf("expected 1 song, got %d", len(response.Data))
	}

	song := response.Data[0].Attributes
	if song.Title != "Holy Forever" || song.CCLINumber != 7201044 {
		t.Errorf("unexpected attributes: %+v", song)
	}
	if song.LastScheduledAt != nil {
		t.Errorf("expected nil LastScheduledAt, got %v", song.LastScheduledAt)
	}
}

func TestGetSongsNilParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params for nil SongsParams, got %q", r.URL.RawQuery)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetSongs(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStringOrStringsUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		json string
		want StringOrStrings
	}{
		{"null", `null`, nil},
		{"single string", `"Aug 16, 2026"`, StringOrStrings{"Aug 16, 2026"}},
		{"array", `["Aug 16, 2026", "Aug 23, 2026"]`, StringOrStrings{"Aug 16, 2026", "Aug 23, 2026"}},
		{"empty array", `[]`, StringOrStrings{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got StringOrStrings
			if err := json.Unmarshal([]byte(tc.json), &got); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("expected %#v, got %#v", tc.want, got)
			}
		})
	}
}

// TestGetSongsLastScheduledShortDatesAsString reproduces a real response
// shape PCO has sent in production: last_scheduled_short_dates as a bare
// string instead of the documented array, which the plain []string field
// this replaced would fail to decode entirely.
func TestGetSongsLastScheduledShortDatesAsString(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"data":[{
			"type": "Song",
			"id": "1",
			"attributes": {
				"title": "Holy Forever",
				"last_scheduled_short_dates": "Aug 16, 2026"
			}
		}]}`)
	})

	response, err := GetSongs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := response.Data[0].Attributes.LastScheduledShortDates
	want := StringOrStrings{"Aug 16, 2026"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %#v, got %#v", want, got)
	}
}

func TestGetSong(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath + "/1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Song","id":"1","attributes":{"title":"Holy Forever"}}}`)
	})

	response, err := GetSong(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "1" || response.Data.Attributes.Title != "Holy Forever" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestCreateSong(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if want := "/" + songsPath; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		attrs := attributes(t, decodeBody(t, r))
		if attrs["title"] != "Holy Forever" {
			t.Errorf("expected title Holy Forever, got %v", attrs["title"])
		}
		if attrs["author"] != "Brian Johnson" {
			t.Errorf("expected author Brian Johnson, got %v", attrs["author"])
		}
		if _, ok := attrs["notes"]; ok {
			t.Errorf("expected no notes attribute (PCO rejects it on create), got %v", attrs["notes"])
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Song","id":"1","attributes":{"title":"Holy Forever"}}}`)
	})

	response, err := CreateSong(context.Background(), &CreateSongParams{Title: "Holy Forever", Author: "Brian Johnson"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "1" {
		t.Errorf("expected id 1, got %q", response.Data.ID)
	}
}

func TestCreateSongOmitsUnsetFields(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attrs := attributes(t, decodeBody(t, r))
		if len(attrs) != 1 {
			t.Errorf("expected only title to be sent for otherwise-zero params, got %+v", attrs)
		}
		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Song","id":"1"}}`)
	})

	if _, err := CreateSong(context.Background(), &CreateSongParams{Title: "Holy Forever"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateSongNilParams(t *testing.T) {
	if _, err := CreateSong(context.Background(), nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestDeleteSong(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath + "/1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := DeleteSong(context.Background(), "1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
