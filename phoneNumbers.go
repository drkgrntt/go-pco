package pco

import (
	"context"
	"fmt"
	"time"
)

type PhoneNumberRelationships struct {
	Person struct {
		Data General `json:"data"`
	} `json:"person"`
}

type PhoneNumberAttributes struct {
	Number          string    `json:"number"`
	Carrier         string    `json:"carrier"`
	Location        string    `json:"location"`
	Primary         bool      `json:"primary"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	E164            string    `json:"e164"`
	International   string    `json:"international"`
	National        string    `json:"national"`
	CountryCode     string    `json:"country_code"`
	FormattedNumber string    `json:"formatted_number"`
}

type PhoneNumberData struct {
	Type          string                   `json:"type"`
	ID            string                   `json:"id"`
	Attributes    PhoneNumberAttributes    `json:"attributes"`
	Relationships PhoneNumberRelationships `json:"relationships"`
}

type PhoneNumberCreateResponse struct {
	Data     PhoneNumberData `json:"data"`
	Included []any           `json:"included"`
	Links    Links           `json:"links"`
	Meta     Meta            `json:"meta"`
}

type PhoneNumberCreateParams struct {
	Number   string
	Location string
	Primary  bool
}

func CreatePhoneNumber(ctx context.Context, personID string, params *PhoneNumberCreateParams) (response PhoneNumberCreateResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s/%s/phone_numbers", baseURL, peoplePath, personID)

	body := NewRequestBody(map[string]any{
		"number":   params.Number,
		"location": params.Location,
		"primary":  params.Primary,
	})

	response, err = NewRequest[PhoneNumberCreateResponse](ctx, "POST", url, body)

	return
}
