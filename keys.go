package pco

import (
	"context"
	"fmt"
	"time"
)

func keysPath(songID, arrangementID string) string {
	return fmt.Sprintf("%s/%s/keys", arrangementsPath(songID), arrangementID)
}

type KeyRelationships struct {
	Arrangement struct {
		Data *General `json:"data"`
	} `json:"arrangement"`
}

// AlternateKey is one entry of KeyAttributes.AlternateKeys - confirmed live
// (PCO's own attribute table documents this field as a bare "string", which
// is wrong; a real response sends an array of these).
type AlternateKey struct {
	Name  string `json:"name"`
	Pitch string `json:"pitch"`
}

// KeyAttributes has no Capo field - PCO's API doesn't expose one. The capo
// number shown in PCO's own UI is computed there from a starting key
// relative to an instrument's preferred key, not stored as data on this
// resource.
type KeyAttributes struct {
	AlternateKeys []AlternateKey `json:"alternate_keys"`
	CreatedAt     time.Time      `json:"created_at"`
	EndingKey     string         `json:"ending_key"`
	EndingMinor   bool           `json:"ending_minor"`
	Name          string         `json:"name"`
	StartingKey   string         `json:"starting_key"`
	StartingMinor bool           `json:"starting_minor"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type KeyData struct {
	Type          string           `json:"type"`
	ID            string           `json:"id"`
	Attributes    KeyAttributes    `json:"attributes"`
	Relationships KeyRelationships `json:"relationships"`
}

type KeyResponse struct {
	Data     KeyData `json:"data"`
	Included []any   `json:"included"`
	Links    Links   `json:"links"`
	Meta     Meta    `json:"meta"`
}

type KeyListResponse struct {
	Data     []KeyData `json:"data"`
	Included []any     `json:"included"`
	Links    Links     `json:"links"`
	Meta     Meta      `json:"meta"`
}

type KeysParams struct {
	// OrderBy sorts by a can_order_by field: "created_at" or "updated_at".
	// Prefix with "-" for descending.
	OrderBy string
	PerPage int
	Offset  int
}

func GetKeys(ctx context.Context, songID, arrangementID string, params *KeysParams) (response KeyListResponse, err error) {
	if params == nil {
		params = &KeysParams{}
	}

	q := NewQueryParams().
		OrderBy(params.OrderBy).
		PerPage(params.PerPage).
		Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, keysPath(songID, arrangementID), q.Encode())

	response, err = NewRequest[KeyListResponse](ctx, "GET", url, nil)

	return
}

func GetKey(ctx context.Context, songID, arrangementID, id string) (response KeyResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, keysPath(songID, arrangementID), id)

	response, err = NewRequest[KeyResponse](ctx, "GET", url, nil)

	return
}

// CreateKeyParams has no AlternateKeys field - its create/update wire shape
// isn't confirmed (see AlternateKey; PCO's own docs get even the read-side
// type wrong), so it's left write-only-unsupported until that's verified
// live rather than guessed at.
type CreateKeyParams struct {
	Name        string
	StartingKey string
	EndingKey   string
}

func CreateKey(ctx context.Context, songID, arrangementID string, params *CreateKeyParams) (response KeyResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s", baseURL, keysPath(songID, arrangementID))

	attributes := map[string]any{}
	if params.Name != "" {
		attributes["name"] = params.Name
	}
	if params.StartingKey != "" {
		attributes["starting_key"] = params.StartingKey
	}
	if params.EndingKey != "" {
		attributes["ending_key"] = params.EndingKey
	}

	response, err = NewRequest[KeyResponse](ctx, "POST", url, NewRequestBody(attributes))

	return
}
