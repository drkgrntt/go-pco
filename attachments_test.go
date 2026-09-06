package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetAttachments(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + songsPath + "/song-1/attachments"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"Attachment","id":"att-1","attributes":{
			"filename": "Lead Sheet.pdf",
			"content_type": "application/pdf",
			"file_size": 12345,
			"url": "https://files.planningcenteronline.com/lead-sheet.pdf",
			"created_at": "2026-01-01T00:00:00Z",
			"updated_at": "2026-01-02T00:00:00Z"
		}}]}`)
	})

	response, err := GetAttachments(context.Background(), "song-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(response.Data))
	}

	attrs := response.Data[0].Attributes
	if attrs.Filename != "Lead Sheet.pdf" {
		t.Errorf("expected filename %q, got %q", "Lead Sheet.pdf", attrs.Filename)
	}
	if attrs.ContentType != "application/pdf" {
		t.Errorf("expected content_type %q, got %q", "application/pdf", attrs.ContentType)
	}
	if attrs.FileSize != 12345 {
		t.Errorf("expected file_size 12345, got %d", attrs.FileSize)
	}
	if attrs.URL != "https://files.planningcenteronline.com/lead-sheet.pdf" {
		t.Errorf("expected url to round-trip, got %q", attrs.URL)
	}
}

func TestGetAttachmentsEmpty(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	response, err := GetAttachments(context.Background(), "song-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 0 {
		t.Errorf("expected no attachments, got %d", len(response.Data))
	}
}

func TestGetAttachmentsBuildsQueryParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("order"); got != "-created_at" {
			t.Errorf("expected order=-created_at, got %q", got)
		}
		if got := q.Get("per_page"); got != "10" {
			t.Errorf("expected per_page=10, got %q", got)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetAttachments(context.Background(), "song-1", &AttachmentsParams{OrderBy: "-created_at", PerPage: 10}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAttachmentsNilParamsDoesNotPanic(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetAttachments(context.Background(), "song-1", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
