package pco

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func sign(t *testing.T, secret, body string) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "top-secret"
	body := []byte(`{"data":[{"id":"1"}]}`)

	if !VerifyWebhookSignature(secret, body, sign(t, secret, string(body))) {
		t.Error("expected a signature computed with the right secret/body to verify")
	}
	if VerifyWebhookSignature(secret, body, sign(t, "wrong-secret", string(body))) {
		t.Error("expected a signature computed with the wrong secret to fail")
	}
	if VerifyWebhookSignature(secret, []byte(`{"tampered":true}`), sign(t, secret, string(body))) {
		t.Error("expected a signature for a different body to fail")
	}
	if VerifyWebhookSignature(secret, body, "not-hex-at-all") {
		t.Error("expected a garbage signature to fail")
	}
}

func TestReadWebhookRequestValidSignature(t *testing.T) {
	secret := "top-secret"
	payload := `{"data":[{"id":"1","type":"EventDelivery","attributes":{"name":"people.v2.events.person.created","attempt":1,"payload":"{}"}}]}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set(SignatureHeader, sign(t, secret, payload))

	body, err := ReadWebhookRequest(req, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != payload {
		t.Errorf("expected body to round-trip, got %q", body)
	}
}

func TestReadWebhookRequestInvalidSignature(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{}`))
	req.Header.Set(SignatureHeader, "deadbeef")

	if _, err := ReadWebhookRequest(req, "top-secret"); err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestParseWebhookDelivery(t *testing.T) {
	payload := `{"data":[{"id":"cac783dc-75e7-4c5b-88e8-502d9d8682ae","type":"EventDelivery","attributes":{"name":"people.v2.events.person.updated","attempt":1,"payload":"{\"data\":{\"type\":\"Person\",\"id\":\"1\"}}"},"relationships":{"organization":{"data":{"type":"Organization","id":"127"}}}}]}`

	delivery, err := ParseWebhookDelivery([]byte(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(delivery.Data) != 1 {
		t.Fatalf("expected 1 event, got %d", len(delivery.Data))
	}

	event := delivery.Data[0]
	if event.Attributes.Name != "people.v2.events.person.updated" {
		t.Errorf("unexpected event name: %q", event.Attributes.Name)
	}
	if event.Attributes.Attempt != 1 {
		t.Errorf("expected attempt 1, got %d", event.Attributes.Attempt)
	}
	if event.Relationships.Organization.Data.ID != "127" {
		t.Errorf("expected organization id 127, got %q", event.Relationships.Organization.Data.ID)
	}
}

func TestParseEventPayload(t *testing.T) {
	payload := `{"data":{"type":"Person","id":"1","attributes":{"first_name":"Ada"}}}`

	result, err := ParseEventPayload[PersonCreateResponse](payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Attributes.FirstName != "Ada" {
		t.Errorf("expected first_name Ada, got %q", result.Data.Attributes.FirstName)
	}
}
