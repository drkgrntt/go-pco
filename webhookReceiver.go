package pco

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// SignatureHeader is the header PCO signs each webhook delivery with.
const SignatureHeader = "X-PCO-Webhooks-Authenticity"

// ErrInvalidSignature is returned by ReadWebhookRequest when the
// X-PCO-Webhooks-Authenticity header doesn't match the request body.
var ErrInvalidSignature = errors.New("pco: webhook signature does not match")

// VerifyWebhookSignature reports whether signature (the value of the
// X-PCO-Webhooks-Authenticity header) is a valid HMAC-SHA256 digest of
// body, keyed with the subscription's authenticity_secret.
func VerifyWebhookSignature(secret string, body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ReadWebhookRequest reads and verifies an incoming webhook HTTP request
// against a subscription's authenticity_secret, returning the raw body on
// success. Call this first in your webhook receiver handler, then pass the
// result to ParseWebhookDelivery.
func ReadWebhookRequest(r *http.Request, secret string) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if !VerifyWebhookSignature(secret, body, r.Header.Get(SignatureHeader)) {
		return nil, ErrInvalidSignature
	}

	return body, nil
}

// EventDeliveryAttributes describes one event within a webhook delivery
// POST body.
type EventDeliveryAttributes struct {
	// Name is the subscribed event, e.g. "people.v2.events.person.updated".
	Name string `json:"name"`
	// Attempt is the 1-indexed delivery attempt number.
	Attempt int `json:"attempt"`
	// Payload is a JSON-encoded string of the affected resource. Decode it
	// with ParseEventPayload once you know the resource's response type.
	Payload string `json:"payload"`
}

type EventDeliveryData struct {
	Type          string                     `json:"type"`
	ID            string                     `json:"id"`
	Attributes    EventDeliveryAttributes    `json:"attributes"`
	Relationships EventDeliveryRelationships `json:"relationships"`
}

type EventDeliveryRelationships struct {
	Organization struct {
		Data General `json:"data"`
	} `json:"organization"`
}

// WebhookDelivery is the JSON body PCO POSTs to a subscription's URL. A
// single delivery can carry more than one event.
type WebhookDelivery struct {
	Data []EventDeliveryData `json:"data"`
}

// ParseWebhookDelivery decodes the raw, signature-verified body from
// ReadWebhookRequest into its EventDelivery envelope.
func ParseWebhookDelivery(body []byte) (delivery WebhookDelivery, err error) {
	err = json.Unmarshal(body, &delivery)
	return
}

// ParseEventPayload decodes an EventDeliveryAttributes.Payload string (or
// a WebhookEventAttributes.Payload from GetWebhookEvents) into the given
// response type, e.g. ParseEventPayload[pco.PersonCreateResponse](payload).
func ParseEventPayload[T any](payload string) (result T, err error) {
	err = json.Unmarshal([]byte(payload), &result)
	return
}
