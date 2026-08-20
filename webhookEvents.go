package pco

import (
	"context"
	"fmt"
	"time"
)

type WebhookEventRelationships struct {
	Subscription struct {
		Data General `json:"data"`
	} `json:"subscription"`
	Person struct {
		Data *General `json:"data"`
	} `json:"person"`
}

type WebhookEventAttributes struct {
	CreatedAt time.Time `json:"created_at"`
	Payload   string    `json:"payload"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
	UUID      string    `json:"uuid"`
}

type WebhookEventData struct {
	Type          string                    `json:"type"`
	ID            string                    `json:"id"`
	Attributes    WebhookEventAttributes    `json:"attributes"`
	Relationships WebhookEventRelationships `json:"relationships"`
}

type WebhookEventResponse struct {
	Data  WebhookEventData `json:"data"`
	Links Links            `json:"links"`
	Meta  Meta             `json:"meta"`
}

type WebhookEventListResponse struct {
	Data  []WebhookEventData `json:"data"`
	Links Links              `json:"links"`
	Meta  Meta               `json:"meta"`
}

type WebhookEventsParams struct {
	PerPage int
	Offset  int
}

func webhookEventsPath(subscriptionID string) string {
	return fmt.Sprintf("%s/%s/events", webhookSubscriptionsPath, subscriptionID)
}

// GetWebhookEvents lists the delivery attempts PCO has recorded for a
// subscription. Each Attributes.Payload is a JSON-encoded string of the
// same envelope PCO POSTs to your webhook URL -- decode it with
// ParseEventPayload.
func GetWebhookEvents(ctx context.Context, subscriptionID string, params *WebhookEventsParams) (response WebhookEventListResponse, err error) {
	if params == nil {
		params = &WebhookEventsParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, webhookEventsPath(subscriptionID), q.Encode())

	response, err = NewRequest[WebhookEventListResponse](ctx, "GET", url, nil)

	return
}

func GetWebhookEvent(ctx context.Context, subscriptionID, eventID string) (response WebhookEventResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, webhookEventsPath(subscriptionID), eventID)

	response, err = NewRequest[WebhookEventResponse](ctx, "GET", url, nil)

	return
}

// IgnoreWebhookEvent marks a pending event as ignored so PCO stops
// retrying delivery for it.
func IgnoreWebhookEvent(ctx context.Context, subscriptionID, eventID string) (response WebhookEventResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s/ignore", baseURL, webhookEventsPath(subscriptionID), eventID)

	response, err = NewRequest[WebhookEventResponse](ctx, "POST", url, nil)

	return
}

// RedeliverWebhookEvent asks PCO to resend a previously failed/skipped
// event.
func RedeliverWebhookEvent(ctx context.Context, subscriptionID, eventID string) (response WebhookEventResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s/redeliver", baseURL, webhookEventsPath(subscriptionID), eventID)

	response, err = NewRequest[WebhookEventResponse](ctx, "POST", url, nil)

	return
}
