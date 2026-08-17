package pco

import (
	"net/http"
	"testing"
)

func TestGetItems(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/items"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"Item","id":"1","attributes":{"title":"Welcome"}}]}`)
	})

	response, err := GetItems("st-1", "p-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.Title != "Welcome" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestGetItem(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/items/i-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Item","id":"i-1","attributes":{"title":"Welcome"}}}`)
	})

	response, err := GetItem("st-1", "p-1", "i-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "i-1" {
		t.Errorf("expected id i-1, got %q", response.Data.ID)
	}
}

func TestCreateItem(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attrs := attributes(t, decodeBody(t, r))
		if attrs["title"] != "Opening Song" {
			t.Errorf("expected title Opening Song, got %v", attrs["title"])
		}
		if attrs["item_type"] != ItemTypeSong {
			t.Errorf("expected item_type %q, got %v", ItemTypeSong, attrs["item_type"])
		}
		if attrs["service_position"] != ServicePositionPre {
			t.Errorf("expected service_position %q, got %v", ServicePositionPre, attrs["service_position"])
		}
		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Item","id":"i-1","attributes":{"title":"Opening Song","item_type":"song"}}}`)
	})

	response, err := CreateItem("st-1", "p-1", &CreateItemParams{
		Title:           "Opening Song",
		ItemType:        ItemTypeSong,
		ServicePosition: ServicePositionPre,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.ItemType != "song" {
		t.Errorf("expected item_type song, got %q", response.Data.Attributes.ItemType)
	}
}

func TestCreateItemNilParams(t *testing.T) {
	if _, err := CreateItem("st-1", "p-1", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestUpdateItem(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/items/i-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		attrs := attributes(t, decodeBody(t, r))
		if attrs["sequence"] != float64(2) {
			t.Errorf("expected sequence 2, got %v", attrs["sequence"])
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Item","id":"i-1","attributes":{"sequence":2}}}`)
	})

	response, err := UpdateItem("st-1", "p-1", "i-1", &UpdateItemParams{Sequence: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.Sequence != 2 {
		t.Errorf("expected sequence 2, got %d", response.Data.Attributes.Sequence)
	}
}

func TestDeleteItem(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/items/i-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := DeleteItem("st-1", "p-1", "i-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
