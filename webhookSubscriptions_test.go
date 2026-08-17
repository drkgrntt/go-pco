package pco

import (
	"net/http"
	"testing"
)

func TestGetWebhookSubscriptions(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + webhookSubscriptionsPath; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		if got := r.URL.Query().Get("where[application_id]"); got != "app-1" {
			t.Errorf("expected where[application_id]=app-1, got %q", got)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"WebhookSubscription","id":"1","attributes":{"name":"people.v2.events.person.created","active":true}}]}`)
	})

	response, err := GetWebhookSubscriptions(&WebhookSubscriptionsParams{ApplicationID: "app-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || !response.Data[0].Attributes.Active {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestGetWebhookSubscription(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + webhookSubscriptionsPath + "/1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"WebhookSubscription","id":"1","attributes":{"name":"people.v2.events.person.created"}}}`)
	})

	response, err := GetWebhookSubscription("1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "1" {
		t.Errorf("expected id 1, got %q", response.Data.ID)
	}
}

func TestCreateWebhookSubscription(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		attrs := attributes(t, decodeBody(t, r))
		if attrs["name"] != "people.v2.events.person.created" {
			t.Errorf("unexpected name: %v", attrs["name"])
		}
		if attrs["url"] != "https://example.com/webhooks" {
			t.Errorf("unexpected url: %v", attrs["url"])
		}
		if attrs["active"] != true {
			t.Errorf("expected active true, got %v", attrs["active"])
		}
		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"WebhookSubscription","id":"1","attributes":{"name":"people.v2.events.person.created","url":"https://example.com/webhooks","active":true,"authenticity_secret":"shh"}}}`)
	})

	response, err := CreateWebhookSubscription(&CreateWebhookSubscriptionParams{
		Name:   "people.v2.events.person.created",
		URL:    "https://example.com/webhooks",
		Active: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.AuthenticitySecret != "shh" {
		t.Errorf("expected authenticity secret to round-trip, got %q", response.Data.Attributes.AuthenticitySecret)
	}
}

func TestCreateWebhookSubscriptionNilParams(t *testing.T) {
	if _, err := CreateWebhookSubscription(nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestUpdateWebhookSubscriptionActive(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		attrs := attributes(t, decodeBody(t, r))
		if len(attrs) != 1 || attrs["active"] != false {
			t.Errorf("expected only active=false in body, got %+v", attrs)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"WebhookSubscription","id":"1","attributes":{"active":false}}}`)
	})

	response, err := UpdateWebhookSubscriptionActive("1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.Active {
		t.Errorf("expected active false, got true")
	}
}

func TestDeleteWebhookSubscription(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if want := "/" + webhookSubscriptionsPath + "/1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := DeleteWebhookSubscription("1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRotateWebhookSubscriptionSecret(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if want := "/" + webhookSubscriptionsPath + "/1/rotate_secret"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"WebhookSubscription","id":"1","attributes":{"authenticity_secret":"new-secret"}}}`)
	})

	response, err := RotateWebhookSubscriptionSecret("1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.AuthenticitySecret != "new-secret" {
		t.Errorf("expected new-secret, got %q", response.Data.Attributes.AuthenticitySecret)
	}
}
