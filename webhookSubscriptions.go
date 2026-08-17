package pco

import (
	"fmt"
	"time"
)

const webhookSubscriptionsPath = "webhooks/v2/webhook_subscriptions"

type WebhookSubscriptionAttributes struct {
	Active             bool      `json:"active"`
	ApplicationID      string    `json:"application_id"`
	AuthenticitySecret string    `json:"authenticity_secret"`
	CreatedAt          time.Time `json:"created_at"`
	Name               string    `json:"name"`
	UpdatedAt          time.Time `json:"updated_at"`
	URL                string    `json:"url"`
}

type WebhookSubscriptionData struct {
	Type       string                        `json:"type"`
	ID         string                        `json:"id"`
	Attributes WebhookSubscriptionAttributes `json:"attributes"`
}

type WebhookSubscriptionResponse struct {
	Data     WebhookSubscriptionData `json:"data"`
	Included []any                   `json:"included"`
	Links    Links                   `json:"links"`
	Meta     Meta                    `json:"meta"`
}

type WebhookSubscriptionListResponse struct {
	Data     []WebhookSubscriptionData `json:"data"`
	Included []any                     `json:"included"`
	Links    Links                     `json:"links"`
	Meta     Meta                      `json:"meta"`
}

type WebhookSubscriptionsParams struct {
	ApplicationID string
	PerPage       int
	Offset        int
}

// GetWebhookSubscriptions lists the webhook subscriptions on the account.
func GetWebhookSubscriptions(params *WebhookSubscriptionsParams) (response WebhookSubscriptionListResponse, err error) {
	if params == nil {
		params = &WebhookSubscriptionsParams{}
	}

	q := NewQueryParams().
		Where("application_id", params.ApplicationID).
		PerPage(params.PerPage).
		Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, webhookSubscriptionsPath, q.Encode())

	response, err = NewRequest[WebhookSubscriptionListResponse]("GET", url, nil)

	return
}

func GetWebhookSubscription(id string) (response WebhookSubscriptionResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, webhookSubscriptionsPath, id)

	response, err = NewRequest[WebhookSubscriptionResponse]("GET", url, nil)

	return
}

type CreateWebhookSubscriptionParams struct {
	Name   string
	URL    string
	Active bool
}

// CreateWebhookSubscription registers a new webhook subscription. Name
// must match one of the values returned by GetAvailableEvents (e.g.
// "people.v2.events.person.created"). The returned
// Attributes.AuthenticitySecret is only ever returned in full at creation
// time (and after RotateWebhookSubscriptionSecret) -- store it somewhere
// you can read it back from for signature verification.
func CreateWebhookSubscription(params *CreateWebhookSubscriptionParams) (response WebhookSubscriptionResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s", baseURL, webhookSubscriptionsPath)

	body := NewRequestBody(map[string]any{
		"name":   params.Name,
		"url":    params.URL,
		"active": params.Active,
	})

	response, err = NewRequest[WebhookSubscriptionResponse]("POST", url, body)

	return
}

// UpdateWebhookSubscriptionActive updates "active", the only attribute PCO
// allows changing on an existing subscription.
func UpdateWebhookSubscriptionActive(id string, active bool) (response WebhookSubscriptionResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, webhookSubscriptionsPath, id)

	body := NewRequestBody(map[string]any{
		"active": active,
	})

	response, err = NewRequest[WebhookSubscriptionResponse]("PATCH", url, body)

	return
}

func DeleteWebhookSubscription(id string) (err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, webhookSubscriptionsPath, id)

	_, err = NewRequest[struct{}]("DELETE", url, nil)

	return
}

// RotateWebhookSubscriptionSecret invalidates the subscription's current
// authenticity_secret and returns the subscription with the new one.
func RotateWebhookSubscriptionSecret(id string) (response WebhookSubscriptionResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s/rotate_secret", baseURL, webhookSubscriptionsPath, id)

	response, err = NewRequest[WebhookSubscriptionResponse]("POST", url, nil)

	return
}
