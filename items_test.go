package pco

import (
	"context"
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

	response, err := GetItems(context.Background(), "st-1", "p-1", nil)
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

	response, err := GetItem(context.Background(), "st-1", "p-1", "i-1")
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

	response, err := CreateItem(context.Background(), "st-1", "p-1", &CreateItemParams{
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

func TestCreateItemWithSong(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(t, r)

		attrs := attributes(t, body)
		if attrs["title"] != "Opening Song" {
			t.Errorf("expected title Opening Song, got %v", attrs["title"])
		}

		rels := relationships(t, body)
		song, ok := rels["song"].(map[string]any)
		if !ok {
			t.Fatalf("expected relationships.song object, got %v", rels)
		}
		data, ok := song["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected relationships.song.data object, got %v", song)
		}
		if data["type"] != "Song" || data["id"] != "song-1" {
			t.Errorf("expected relationships.song.data {type: Song, id: song-1}, got %v", data)
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Item","id":"i-1","attributes":{"title":"Opening Song"}}}`)
	})

	_, err := CreateItem(context.Background(), "st-1", "p-1", &CreateItemParams{
		Title:    "Opening Song",
		ItemType: ItemTypeSong,
		SongID:   "song-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCreateItemWithSongAndArrangement confirms both relationships can be
// set together on one create - required for an item to actually show its
// chord chart/lyrics/structure in Planning Center, not just be linked to
// the bare song (confirmed live against the real API: a song-only item
// gets no arrangement relationship at all).
func TestCreateItemWithSongAndArrangement(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(t, r)
		rels := relationships(t, body)

		song, ok := rels["song"].(map[string]any)
		if !ok {
			t.Fatalf("expected relationships.song object, got %v", rels)
		}
		songData, ok := song["data"].(map[string]any)
		if !ok || songData["type"] != "Song" || songData["id"] != "song-1" {
			t.Errorf("expected relationships.song.data {type: Song, id: song-1}, got %v", song)
		}

		arrangement, ok := rels["arrangement"].(map[string]any)
		if !ok {
			t.Fatalf("expected relationships.arrangement object, got %v", rels)
		}
		arrangementData, ok := arrangement["data"].(map[string]any)
		if !ok || arrangementData["type"] != "Arrangement" || arrangementData["id"] != "arr-1" {
			t.Errorf("expected relationships.arrangement.data {type: Arrangement, id: arr-1}, got %v", arrangement)
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Item","id":"i-1","attributes":{"title":"Opening Song"}}}`)
	})

	_, err := CreateItem(context.Background(), "st-1", "p-1", &CreateItemParams{
		Title:         "Opening Song",
		ItemType:      ItemTypeSong,
		SongID:        "song-1",
		ArrangementID: "arr-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCreateItemWithArrangementOnly confirms ArrangementID doesn't require
// SongID to be set too - not a realistic use, but the relationships map is
// built independently for each, so nothing should assume they're paired.
func TestCreateItemWithArrangementOnly(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(t, r)
		rels := relationships(t, body)

		if _, ok := rels["song"]; ok {
			t.Errorf("expected no relationships.song, got %v", rels)
		}
		arrangement, ok := rels["arrangement"].(map[string]any)
		if !ok {
			t.Fatalf("expected relationships.arrangement object, got %v", rels)
		}
		if data, ok := arrangement["data"].(map[string]any); !ok || data["id"] != "arr-1" {
			t.Errorf("expected relationships.arrangement.data.id arr-1, got %v", arrangement)
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Item","id":"i-1","attributes":{"title":"Opening Song"}}}`)
	})

	_, err := CreateItem(context.Background(), "st-1", "p-1", &CreateItemParams{
		Title:         "Opening Song",
		ItemType:      ItemTypeSong,
		ArrangementID: "arr-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestCreateItemWithArrangementAndKey confirms KeyID is a genuinely
// separate relationship from ArrangementID, not implied by it - confirmed
// live: an item with only ArrangementID set still comes back with an
// empty KeyName and no key relationship at all.
func TestCreateItemWithArrangementAndKey(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(t, r)
		rels := relationships(t, body)

		arrangement, ok := rels["arrangement"].(map[string]any)
		if !ok {
			t.Fatalf("expected relationships.arrangement object, got %v", rels)
		}
		if data, ok := arrangement["data"].(map[string]any); !ok || data["id"] != "arr-1" {
			t.Errorf("expected relationships.arrangement.data.id arr-1, got %v", arrangement)
		}

		key, ok := rels["key"].(map[string]any)
		if !ok {
			t.Fatalf("expected relationships.key object, got %v", rels)
		}
		data, ok := key["data"].(map[string]any)
		if !ok || data["type"] != "Key" || data["id"] != "key-1" {
			t.Errorf("expected relationships.key.data {type: Key, id: key-1}, got %v", key)
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Item","id":"i-1"}}`)
	})

	_, err := CreateItem(context.Background(), "st-1", "p-1", &CreateItemParams{
		Title:         "Opening Song",
		ItemType:      ItemTypeSong,
		ArrangementID: "arr-1",
		KeyID:         "key-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateItemNilParams(t *testing.T) {
	if _, err := CreateItem(context.Background(), "st-1", "p-1", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestUpdateItem(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/items/i-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		attrs := attributes(t, decodeBody(t, r))
		if attrs["title"] != "Opening Song" {
			t.Errorf("expected title Opening Song, got %v", attrs["title"])
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Item","id":"i-1","attributes":{"title":"Opening Song"}}}`)
	})

	response, err := UpdateItem(context.Background(), "st-1", "p-1", "i-1", &UpdateItemParams{Title: "Opening Song"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.Title != "Opening Song" {
		t.Errorf("expected title Opening Song, got %q", response.Data.Attributes.Title)
	}
}

// TestUpdateItemPartial confirms unset fields are omitted from the request
// entirely rather than sent as zero values - the whole point of the
// pointer/empty-string convention on UpdateItemParams, since otherwise an
// update meant to touch only one field would silently clobber the rest
// back to blank in the real plan.
func TestUpdateItemPartial(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attrs := attributes(t, decodeBody(t, r))
		if len(attrs) != 1 {
			t.Errorf("expected exactly one attribute set, got %v", attrs)
		}
		if attrs["length"] != float64(0) {
			t.Errorf("expected length 0, got %v", attrs["length"])
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Item","id":"i-1","attributes":{"length":0}}}`)
	})

	length := 0
	if _, err := UpdateItem(context.Background(), "st-1", "p-1", "i-1", &UpdateItemParams{Length: &length}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestReorderItems asserts the exact request shape ReorderItems sends,
// confirmed against PCO's own documentation API for the "item_reorder"
// plan action - see ReorderItems' doc comment.
func TestReorderItems(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/item_reorder"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		body := decodeBody(t, r)
		data, ok := body["data"].(map[string]any)
		if !ok {
			t.Fatalf("request body has no data object: %v", body)
		}
		if data["type"] != "PlanItemReorder" {
			t.Errorf("expected type PlanItemReorder, got %v", data["type"])
		}
		attrs, ok := data["attributes"].(map[string]any)
		if !ok {
			t.Fatalf("request body has no data.attributes object: %v", body)
		}
		seq, ok := attrs["sequence"].([]any)
		if !ok || len(seq) != 3 || seq[0] != "i-1" || seq[1] != "i-2" || seq[2] != "i-3" {
			t.Errorf("expected sequence [i-1 i-2 i-3], got %v", attrs["sequence"])
		}

		w.WriteHeader(http.StatusNoContent)
	})

	if err := ReorderItems(context.Background(), "st-1", "p-1", []string{"i-1", "i-2", "i-3"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteItem(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/items/i-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := DeleteItem(context.Background(), "st-1", "p-1", "i-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
