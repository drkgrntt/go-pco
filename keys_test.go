package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetKeys(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath + "/song-1/arrangements/arr-1/keys"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"Key","id":"1","attributes":{"starting_key":"G"}}]}`)
	})

	response, err := GetKeys(context.Background(), "song-1", "arr-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.StartingKey != "G" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestGetKey(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath + "/song-1/arrangements/arr-1/keys/key-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Key","id":"key-1","attributes":{"starting_key":"G","ending_key":"A"}}}`)
	})

	response, err := GetKey(context.Background(), "song-1", "arr-1", "key-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.EndingKey != "A" {
		t.Errorf("expected ending_key A, got %q", response.Data.Attributes.EndingKey)
	}
}

func TestCreateKey(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath + "/song-1/arrangements/arr-1/keys"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		attrs := attributes(t, decodeBody(t, r))
		if attrs["starting_key"] != "G" {
			t.Errorf("expected starting_key G, got %v", attrs["starting_key"])
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Key","id":"key-1","attributes":{"starting_key":"G"}}}`)
	})

	response, err := CreateKey(context.Background(), "song-1", "arr-1", &CreateKeyParams{StartingKey: "G"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "key-1" {
		t.Errorf("expected id key-1, got %q", response.Data.ID)
	}
}

func TestCreateKeyOmitsUnsetFields(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attrs := attributes(t, decodeBody(t, r))
		if len(attrs) != 1 {
			t.Errorf("expected only starting_key to be sent for otherwise-zero params, got %+v", attrs)
		}
		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Key","id":"key-1"}}`)
	})

	if _, err := CreateKey(context.Background(), "song-1", "arr-1", &CreateKeyParams{StartingKey: "G"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateKeyNilParams(t *testing.T) {
	if _, err := CreateKey(context.Background(), "song-1", "arr-1", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}
