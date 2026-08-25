package pco

import (
	"context"
	"net/http"
	"testing"
)

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
