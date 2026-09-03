package pco

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestStringOrNumberUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		json string
		want StringOrNumber
	}{
		{"null", `null`, ""},
		{"string", `"2"`, "2"},
		{"integer", `2`, "2"},
		{"float", `1.5`, "1.5"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got StringOrNumber
			if err := json.Unmarshal([]byte(tc.json), &got); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

// TestGetArrangementSequenceFullNumberAsInteger reproduces a real response
// shape PCO has sent in production: sequence_full[].number as a bare JSON
// number instead of the usual string, which the plain string field this
// replaced would fail to decode entirely (breaking the whole arrangement
// fetch over one field).
func TestGetArrangementSequenceFullNumberAsInteger(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Arrangement","id":"arr-1","attributes":{
			"sequence_full": [{"label": "Verse", "number": 2, "t": "00:12:500", "sid": 1}]
		}}}`)
	})

	response, err := GetArrangement(context.Background(), "song-1", "arr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	steps := response.Data.Attributes.SequenceFull
	if len(steps) != 1 || steps[0].Number != "2" {
		t.Errorf("unexpected sequence_full: %+v", steps)
	}
}

func TestGetArrangements(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath + "/song-1/arrangements"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"Arrangement","id":"1","attributes":{"name":"Default"}}]}`)
	})

	response, err := GetArrangements(context.Background(), "song-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.Name != "Default" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestGetArrangement(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath + "/song-1/arrangements/arr-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Arrangement","id":"arr-1","attributes":{"name":"Default","chord_chart_key":"G"}}}`)
	})

	response, err := GetArrangement(context.Background(), "song-1", "arr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.ChordChartKey != "G" {
		t.Errorf("expected chord_chart_key G, got %q", response.Data.Attributes.ChordChartKey)
	}
}

func TestCreateArrangement(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath + "/song-1/arrangements"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		attrs := attributes(t, decodeBody(t, r))
		if attrs["name"] != "Default" {
			t.Errorf("expected name Default, got %v", attrs["name"])
		}
		if attrs["chord_chart_key"] != "G" {
			t.Errorf("expected chord_chart_key G, got %v", attrs["chord_chart_key"])
		}
		if attrs["bpm"] != float64(120) {
			t.Errorf("expected bpm 120, got %v", attrs["bpm"])
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Arrangement","id":"arr-1","attributes":{"name":"Default"}}}`)
	})

	response, err := CreateArrangement(context.Background(), "song-1", &CreateArrangementParams{
		Name:          "Default",
		ChordChartKey: "G",
		BPM:           120,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "arr-1" {
		t.Errorf("expected id arr-1, got %q", response.Data.ID)
	}
}

func TestCreateArrangementOmitsUnsetFields(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attrs := attributes(t, decodeBody(t, r))
		if len(attrs) != 1 {
			t.Errorf("expected only name to be sent for otherwise-zero params, got %+v", attrs)
		}
		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Arrangement","id":"arr-1"}}`)
	})

	if _, err := CreateArrangement(context.Background(), "song-1", &CreateArrangementParams{Name: "Default"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateArrangementNilParams(t *testing.T) {
	if _, err := CreateArrangement(context.Background(), "song-1", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestUpdateArrangement(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath + "/song-1/arrangements/arr-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		attrs := attributes(t, decodeBody(t, r))
		if len(attrs) != 1 || attrs["chord_chart_key"] != "G" {
			t.Errorf("expected only chord_chart_key=G, got %+v", attrs)
		}

		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Arrangement","id":"arr-1","attributes":{"chord_chart_key":"G"}}}`)
	})

	response, err := UpdateArrangement(context.Background(), "song-1", "arr-1", &UpdateArrangementParams{ChordChartKey: "G"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.ChordChartKey != "G" {
		t.Errorf("expected chord_chart_key G, got %q", response.Data.Attributes.ChordChartKey)
	}
}

func TestUpdateArrangementNilParams(t *testing.T) {
	if _, err := UpdateArrangement(context.Background(), "song-1", "arr-1", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

// TestUpdateArrangementClearsNotes - Notes is a pointer specifically so a
// caller can clear it to empty (see UpdateArrangementParams' own doc
// comment) - a plain string couldn't distinguish "leave notes alone" from
// "set notes to empty" here, same reasoning as UpdateItemParams.
func TestUpdateArrangementClearsNotes(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attrs := attributes(t, decodeBody(t, r))
		if len(attrs) != 1 || attrs["notes"] != "" {
			t.Errorf("expected only notes=\"\", got %+v", attrs)
		}

		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Arrangement","id":"arr-1","attributes":{"notes":""}}}`)
	})

	empty := ""
	response, err := UpdateArrangement(context.Background(), "song-1", "arr-1", &UpdateArrangementParams{Notes: &empty})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.Notes != "" {
		t.Errorf("expected empty notes, got %q", response.Data.Attributes.Notes)
	}
}
