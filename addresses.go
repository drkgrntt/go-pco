package pco

import (
	"fmt"
	"time"
)

type AddressRelationships struct {
	Person struct {
		Data General `json:"data"`
	} `json:"person"`
}

type AddressAttributes struct {
	City        string    `json:"city"`
	State       string    `json:"state"`
	Zip         string    `json:"zip"`
	CountryCode string    `json:"country_code"`
	Primary     bool      `json:"primary"`
	Location    string    `json:"location"`
	StreetLine1 string    `json:"street_line_1"`
	StreetLine2 string    `json:"street_line_2"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CountryName string    `json:"country_name"`
}

type AddressData struct {
	Type          string               `json:"type"`
	ID            string               `json:"id"`
	Attributes    AddressAttributes    `json:"attributes"`
	Relationships AddressRelationships `json:"relationships"`
}

type AddressCreateResponse struct {
	Data     AddressData `json:"data"`
	Included []any       `json:"included"`
	Links    Links       `json:"links"`
	Meta     Meta        `json:"meta"`
}

type AddressCreateParams struct {
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	Zip          string
	CountryCode  string
	Location     string
	Primary      bool
}

func CreateAddress(personID string, params *AddressCreateParams) (response AddressCreateResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s/%s/addresses", baseURL, peoplePath, personID)

	body := NewRequestBody(map[string]any{
		"street_line_1": params.AddressLine1,
		"street_line_2": params.AddressLine2,
		"city":          params.City,
		"state":         params.State,
		"zip":           params.Zip,
		"country_code":  params.CountryCode,
		"location":      params.Location,
		"primary":       params.Primary,
	})

	response, err = NewRequest[AddressCreateResponse]("POST", url, body)

	return
}
