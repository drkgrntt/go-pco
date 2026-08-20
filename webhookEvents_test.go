package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetWebhookEvents(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + webhookSubscriptionsPath + "/sub-1/events"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"Event","id":"1","attributes":{"status":"delivered","uuid":"abc"}}]}`)
	})

	response, err := GetWebhookEvents(context.Background(), "sub-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.Status != "delivered" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestGetWebhookEvent(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + webhookSubscriptionsPath + "/sub-1/events/evt-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Event","id":"evt-1","attributes":{"status":"pending"}}}`)
	})

	response, err := GetWebhookEvent(context.Background(), "sub-1", "evt-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "evt-1" {
		t.Errorf("expected id evt-1, got %q", response.Data.ID)
	}
}

func TestIgnoreWebhookEvent(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if want := "/" + webhookSubscriptionsPath + "/sub-1/events/evt-1/ignore"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Event","id":"evt-1","attributes":{"status":"skipped"}}}`)
	})

	response, err := IgnoreWebhookEvent(context.Background(), "sub-1", "evt-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.Status != "skipped" {
		t.Errorf("expected status skipped, got %q", response.Data.Attributes.Status)
	}
}

func TestRedeliverWebhookEvent(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if want := "/" + webhookSubscriptionsPath + "/sub-1/events/evt-1/redeliver"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Event","id":"evt-1","attributes":{"status":"pending"}}}`)
	})

	if _, err := RedeliverWebhookEvent(context.Background(), "sub-1", "evt-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
