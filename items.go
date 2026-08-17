package pco

import (
	"fmt"
	"time"
)

func itemsPath(serviceTypeID, planID string) string {
	return fmt.Sprintf("%s/%s/items", plansPath(serviceTypeID), planID)
}

type ItemRelationships struct {
	Plan struct {
		Data *General `json:"data"`
	} `json:"plan"`
	Song struct {
		Data *General `json:"data"`
	} `json:"song"`
	Arrangement struct {
		Data *General `json:"data"`
	} `json:"arrangement"`
	Key struct {
		Data *General `json:"data"`
	} `json:"key"`
}

type ItemAttributes struct {
	CustomArrangementSequence []string  `json:"custom_arrangement_sequence"`
	Description               string    `json:"description"`
	HTMLDetails               string    `json:"html_details"`
	ItemType                  string    `json:"item_type"`
	KeyName                   string    `json:"key_name"`
	Length                    int       `json:"length"`
	Sequence                  int       `json:"sequence"`
	ServicePosition           string    `json:"service_position"`
	Title                     string    `json:"title"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

type ItemData struct {
	Type          string            `json:"type"`
	ID            string            `json:"id"`
	Attributes    ItemAttributes    `json:"attributes"`
	Relationships ItemRelationships `json:"relationships"`
}

type ItemResponse struct {
	Data     ItemData `json:"data"`
	Included []any    `json:"included"`
	Links    Links    `json:"links"`
	Meta     Meta     `json:"meta"`
}

type ItemListResponse struct {
	Data     []ItemData `json:"data"`
	Included []any      `json:"included"`
	Links    Links      `json:"links"`
	Meta     Meta       `json:"meta"`
}

type ItemsParams struct {
	PerPage int
	Offset  int
}

func GetItems(serviceTypeID, planID string, params *ItemsParams) (response ItemListResponse, err error) {
	if params == nil {
		params = &ItemsParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, itemsPath(serviceTypeID, planID), q.Encode())

	response, err = NewRequest[ItemListResponse]("GET", url, nil)

	return
}

func GetItem(serviceTypeID, planID, itemID string) (response ItemResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, itemsPath(serviceTypeID, planID), itemID)

	response, err = NewRequest[ItemResponse]("GET", url, nil)

	return
}

// ItemType covers the documented `item_type` values.
const (
	ItemTypeSong   = "song"
	ItemTypeHeader = "header"
	ItemTypeMedia  = "media"
	ItemTypeItem   = "item"
)

// ServicePosition covers the documented `service_position` values.
const (
	ServicePositionPre    = "pre"
	ServicePositionDuring = "during"
	ServicePositionPost   = "post"
)

type CreateItemParams struct {
	Title           string
	Description     string
	ItemType        string
	ServicePosition string
	Length          int
	Sequence        int
}

func CreateItem(serviceTypeID, planID string, params *CreateItemParams) (response ItemResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s", baseURL, itemsPath(serviceTypeID, planID))

	body := NewRequestBody(map[string]any{
		"title":            params.Title,
		"description":      params.Description,
		"item_type":        params.ItemType,
		"service_position": params.ServicePosition,
		"length":           params.Length,
		"sequence":         params.Sequence,
	})

	response, err = NewRequest[ItemResponse]("POST", url, body)

	return
}

type UpdateItemParams struct {
	Title           string
	Description     string
	ServicePosition string
	Length          int
	Sequence        int
}

func UpdateItem(serviceTypeID, planID, itemID string, params *UpdateItemParams) (response ItemResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s/%s", baseURL, itemsPath(serviceTypeID, planID), itemID)

	body := NewRequestBody(map[string]any{
		"title":            params.Title,
		"description":      params.Description,
		"service_position": params.ServicePosition,
		"length":           params.Length,
		"sequence":         params.Sequence,
	})

	response, err = NewRequest[ItemResponse]("PATCH", url, body)

	return
}

func DeleteItem(serviceTypeID, planID, itemID string) (err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, itemsPath(serviceTypeID, planID), itemID)

	_, err = NewRequest[struct{}]("DELETE", url, nil)

	return
}
