# Webhooks

Wraps the [Webhooks v2 API](https://api.planningcenteronline.com/docs/apps/webhooks): managing subscriptions, inspecting delivery history, and — on the receiving end — verifying and decoding the payloads PCO sends to your endpoint.

Two separate things are called "webhook subscription" and "event" here: the **`WebhookSubscription`** resource is what you manage through the API (register/list/delete). An **`EventDelivery`** is the JSON body PCO actually POSTs to your URL each time something happens; the **`WebhookEvent`** resource (`GetWebhookEvents` etc.) is PCO's own record of those deliveries, fetched back through the API for retrying/auditing.

## Subscriptions

**[webhookSubscriptions.go](../webhookSubscriptions.go)**

| Function | Notes |
|---|---|
| `GetWebhookSubscriptions(ctx context.Context, params *WebhookSubscriptionsParams) (WebhookSubscriptionListResponse, error)` | `params.ApplicationID` filters by app; `nil` lists all. |
| `GetWebhookSubscription(ctx context.Context, id string) (WebhookSubscriptionResponse, error)` | |
| `CreateWebhookSubscription(ctx context.Context, params *CreateWebhookSubscriptionParams) (WebhookSubscriptionResponse, error)` | See below. |
| `UpdateWebhookSubscriptionActive(ctx context.Context, id string, active bool) (WebhookSubscriptionResponse, error)` | `active` is the only attribute PCO allows updating. |
| `DeleteWebhookSubscription(ctx context.Context, id string) error` | |
| `RotateWebhookSubscriptionSecret(ctx context.Context, id string) (WebhookSubscriptionResponse, error)` | Invalidates the old `authenticity_secret` and returns the new one. |

```go
type CreateWebhookSubscriptionParams struct {
	Name   string // must match an AvailableEvent name, e.g. "people.v2.events.person.created"
	URL    string // your receiver endpoint
	Active bool
}
```

```go
sub, err := pco.CreateWebhookSubscription(ctx, &pco.CreateWebhookSubscriptionParams{
	Name:   "people.v2.events.person.created",
	URL:    "https://example.com/webhooks/pco",
	Active: true,
})
// sub.Data.Attributes.AuthenticitySecret is only ever returned in full here
// (and again from RotateWebhookSubscriptionSecret) — store it, PCO won't
// show it to you again.
```

## Available events

**[availableEvents.go](../availableEvents.go)**

### `GetAvailableEvents(ctx context.Context, params *AvailableEventsParams) (AvailableEventListResponse, error)`

Lists every event PCO can notify you about, across all products. Use an entry's `Attributes.Name` as `CreateWebhookSubscriptionParams.Name`.

```go
events, err := pco.GetAvailableEvents(ctx, nil)
for _, e := range events.Data {
	fmt.Println(e.Attributes.Name, "-", e.Attributes.App)
}
```

## Event delivery history

**[webhookEvents.go](../webhookEvents.go)**

| Function | Notes |
|---|---|
| `GetWebhookEvents(ctx context.Context, subscriptionID string, params *WebhookEventsParams) (WebhookEventListResponse, error)` | |
| `GetWebhookEvent(ctx context.Context, subscriptionID, eventID string) (WebhookEventResponse, error)` | |
| `IgnoreWebhookEvent(ctx context.Context, subscriptionID, eventID string) (WebhookEventResponse, error)` | Marks a pending event ignored so PCO stops retrying it. |
| `RedeliverWebhookEvent(ctx context.Context, subscriptionID, eventID string) (WebhookEventResponse, error)` | Asks PCO to resend a failed/skipped event. |

`WebhookEventAttributes.Status` is one of `pending`, `delivered`, `failed`, `skipped`, `duplicated` (and possibly others PCO adds later — treat it as an open string, not an enum).

## Receiving webhooks

**[webhookReceiver.go](../webhookReceiver.go)**

PCO signs every delivery with an `X-PCO-Webhooks-Authenticity` header: an HMAC-SHA256 hex digest of the raw request body, keyed with the subscription's `authenticity_secret`. Three functions handle the receiving side:

- `ReadWebhookRequest(r *http.Request, secret string) ([]byte, error)` — reads the body, verifies the signature, and returns the raw bytes (or `pco.ErrInvalidSignature`).
- `ParseWebhookDelivery(body []byte) (WebhookDelivery, error)` — decodes the outer envelope. A single delivery can carry more than one event in `Data`.
- `ParseEventPayload[T any](payload string) (T, error)` — each event's `Attributes.Payload` is itself a JSON-encoded string (PCO double-encodes it); decode it into whatever response type matches the event, e.g. `pco.PersonCreateResponse` for a `people.v2.events.person.*` event.

`VerifyWebhookSignature(secret string, body []byte, signature string) bool` is exported separately if you want to verify signatures without going through `ReadWebhookRequest` (e.g. if you're not using `net/http`).

### Full receiver example

```go
const webhookSecret = "..." // sub.Data.Attributes.AuthenticitySecret, stored at creation time

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := pco.ReadWebhookRequest(r, webhookSecret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	delivery, err := pco.ParseWebhookDelivery(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, event := range delivery.Data {
		switch event.Attributes.Name {
		case "people.v2.events.person.created":
			person, err := pco.ParseEventPayload[pco.PersonCreateResponse](event.Attributes.Payload)
			if err != nil {
				log.Printf("decoding person payload: %v", err)
				continue
			}
			log.Printf("new person: %s %s", person.Data.Attributes.FirstName, person.Data.Attributes.LastName)
		default:
			log.Printf("unhandled event: %s (attempt %d)", event.Attributes.Name, event.Attributes.Attempt)
		}
	}

	w.WriteHeader(http.StatusOK)
}
```
