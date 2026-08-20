package pco

import (
	"context"
	"fmt"
	"time"
)

type EmailRelationships struct {
	Person struct {
		Data General `json:"data"`
	} `json:"person"`
}

type EmailAttributes struct {
	Address   string    `json:"address"`
	Location  string    `json:"location"`
	Primary   bool      `json:"primary"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Blocked   bool      `json:"blocked"`
}

type EmailData struct {
	Type          string             `json:"type"`
	ID            string             `json:"id"`
	Attributes    EmailAttributes    `json:"attributes"`
	Relationships EmailRelationships `json:"relationships"`
}

type EmailCreateResponse struct {
	Data     EmailData `json:"data"`
	Included []any     `json:"included"`
	Links    Links     `json:"links"`
	Meta     Meta      `json:"meta"`
}

type EmailCreateParams struct {
	Address  string
	Location string
	Primary  bool
}

func CreateEmail(ctx context.Context, personID string, params *EmailCreateParams) (response EmailCreateResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s/%s/emails", baseURL, peoplePath, personID)

	body := NewRequestBody(map[string]any{
		"address":  params.Address,
		"location": params.Location,
		"primary":  params.Primary,
	})

	response, err = NewRequest[EmailCreateResponse](ctx, "POST", url, body)

	return
}
