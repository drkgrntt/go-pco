package pco

import (
	"context"
	"fmt"
)

const availableEventsPath = "webhooks/v2/available_events"

type AvailableEventAttributes struct {
	Action   string `json:"action"`
	App      string `json:"app"`
	Name     string `json:"name"`
	Resource string `json:"resource"`
	Type     string `json:"type"`
	Version  string `json:"version"`
}

type AvailableEventData struct {
	Type       string                   `json:"type"`
	ID         string                   `json:"id"`
	Attributes AvailableEventAttributes `json:"attributes"`
}

type AvailableEventListResponse struct {
	Data  []AvailableEventData `json:"data"`
	Links Links                `json:"links"`
	Meta  Meta                 `json:"meta"`
}

type AvailableEventsParams struct {
	PerPage int
	Offset  int
}

// GetAvailableEvents lists every event PCO can send a webhook for (e.g.
// "people.v2.events.person.created"). Use an entry's Attributes.Name as
// CreateWebhookSubscriptionParams.Name to subscribe to it.
func GetAvailableEvents(ctx context.Context, params *AvailableEventsParams) (response AvailableEventListResponse, err error) {
	if params == nil {
		params = &AvailableEventsParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, availableEventsPath, q.Encode())

	response, err = NewRequest[AvailableEventListResponse](ctx, "GET", url, nil)

	return
}
