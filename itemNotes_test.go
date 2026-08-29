package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetItemNotes(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/items/i-1/item_notes"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"ItemNote","id":"1","attributes":{"category_name":"Band","content":"bring extra cables"}}]}`)
	})

	response, err := GetItemNotes(context.Background(), "st-1", "p-1", "i-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.Content != "bring extra cables" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestCreateItemNote(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		body := decodeBody(t, r)
		attrs := attributes(t, body)
		if attrs["content"] != "go slow on the bridge" {
			t.Errorf("unexpected attributes: %+v", attrs)
		}
		rel := relationships(t, body)
		category := rel["item_note_category"].(map[string]any)["data"].(map[string]any)
		if category["type"] != "ItemNoteCategory" || category["id"] != "cat-1" {
			t.Errorf("unexpected item_note_category relationship: %+v", category)
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"ItemNote","id":"note-1","attributes":{"category_name":"Band","content":"go slow on the bridge"}}}`)
	})

	response, err := CreateItemNote(context.Background(), "st-1", "p-1", "i-1", &CreateItemNoteParams{
		Content:            "go slow on the bridge",
		ItemNoteCategoryID: "cat-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "note-1" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestCreateItemNoteNilParams(t *testing.T) {
	if _, err := CreateItemNote(context.Background(), "st-1", "p-1", "i-1", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestUpdateItemNote(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/items/i-1/item_notes/note-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		attrs := attributes(t, decodeBody(t, r))
		if len(attrs) != 1 || attrs["content"] != "updated" {
			t.Errorf("expected only content=updated, got %+v", attrs)
		}

		writeJSON(t, w, http.StatusOK, `{"data":{"type":"ItemNote","id":"note-1","attributes":{"content":"updated"}}}`)
	})

	response, err := UpdateItemNote(context.Background(), "st-1", "p-1", "i-1", "note-1", &UpdateItemNoteParams{Content: "updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.Content != "updated" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestUpdateItemNoteNilParams(t *testing.T) {
	if _, err := UpdateItemNote(context.Background(), "st-1", "p-1", "i-1", "note-1", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestDeleteItemNote(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/items/i-1/item_notes/note-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	if err := DeleteItemNote(context.Background(), "st-1", "p-1", "i-1", "note-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
