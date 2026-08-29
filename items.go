package pco

import (
	"context"
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

func GetItems(ctx context.Context, serviceTypeID, planID string, params *ItemsParams) (response ItemListResponse, err error) {
	if params == nil {
		params = &ItemsParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, itemsPath(serviceTypeID, planID), q.Encode())

	response, err = NewRequest[ItemListResponse](ctx, "GET", url, nil)

	return
}

func GetItem(ctx context.Context, serviceTypeID, planID, itemID string) (response ItemResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, itemsPath(serviceTypeID, planID), itemID)

	response, err = NewRequest[ItemResponse](ctx, "GET", url, nil)

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
	// SongID links the item to a library Song (relationships.song in PCO's
	// Item vertex) - the same relationship GetItems/GetItem decode into
	// ItemRelationships.Song. Leave empty for an item with no linked song.
	SongID string
	// ArrangementID links the item to one of that song's Arrangements
	// (relationships.arrangement) - confirmed live: an item created with
	// just SongID gets no arrangement relationship at all (nil), unlike a
	// song added through Planning Center's own UI, so its chord chart/
	// lyrics/structure don't show up anywhere on the plan without this.
	// Leave empty to omit the relationship, same as before this field
	// existed.
	ArrangementID string
}

func CreateItem(ctx context.Context, serviceTypeID, planID string, params *CreateItemParams) (response ItemResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s", baseURL, itemsPath(serviceTypeID, planID))

	attributes := map[string]any{
		"title":            params.Title,
		"description":      params.Description,
		"item_type":        params.ItemType,
		"service_position": params.ServicePosition,
		"length":           params.Length,
		"sequence":         params.Sequence,
	}

	relationships := map[string]any{}
	if params.SongID != "" {
		relationships["song"] = map[string]any{
			"data": map[string]any{"type": "Song", "id": params.SongID},
		}
	}
	if params.ArrangementID != "" {
		relationships["arrangement"] = map[string]any{
			"data": map[string]any{"type": "Arrangement", "id": params.ArrangementID},
		}
	}

	var body RequestBody
	if len(relationships) > 0 {
		body = NewRequestBodyWithRelationships(attributes, relationships)
	} else {
		body = NewRequestBody(attributes)
	}

	response, err = NewRequest[ItemResponse](ctx, "POST", url, body)

	return
}

// UpdateItemParams is a partial update - only set fields are sent, so
// updating (say) just Length leaves Title/Description/etc as they already
// are in PCO instead of clobbering them back to empty. Length is a pointer
// because 0 is a legitimate value for it, so a zero value can't double as
// "leave this alone" the way an empty string does for the string fields.
//
// There's deliberately no Sequence field here - PATCHing an item's sequence
// directly is rejected by PCO ("sequence cannot be assigned", confirmed
// against the live API). Reordering existing items is done via SortItems
// instead, which PCO exposes as a dedicated bulk-sort action.
type UpdateItemParams struct {
	Title           string
	Description     string
	ServicePosition string
	Length          *int
}

func UpdateItem(ctx context.Context, serviceTypeID, planID, itemID string, params *UpdateItemParams) (response ItemResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s/%s", baseURL, itemsPath(serviceTypeID, planID), itemID)

	attributes := map[string]any{}
	if params.Title != "" {
		attributes["title"] = params.Title
	}
	if params.Description != "" {
		attributes["description"] = params.Description
	}
	if params.ServicePosition != "" {
		attributes["service_position"] = params.ServicePosition
	}
	if params.Length != nil {
		attributes["length"] = *params.Length
	}

	response, err = NewRequest[ItemResponse](ctx, "PATCH", url, NewRequestBody(attributes))

	return
}

func DeleteItem(ctx context.Context, serviceTypeID, planID, itemID string) (err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, itemsPath(serviceTypeID, planID), itemID)

	_, err = NewRequest[struct{}](ctx, "DELETE", url, nil)

	return
}

// ReorderItems reorders every item in a plan to match itemIDs - the full,
// final ordering, not a delta - via PCO's "item_reorder" plan action
// (confirmed directly against PCO's own documentation API,
// GET .../services/v2/documentation/2018-11-01/vertices/plan, which isn't
// crawlable through the JS docs site but is itself a plain JSON endpoint).
// PCO's docs note it expects every item's id in the plan, in order -
// there's no documented partial/delta form, so omitting an item likely
// misplaces it rather than leaving it alone. On success PCO returns 204 No
// Content, matched here by discarding the response body (see DeleteItem).
func ReorderItems(ctx context.Context, serviceTypeID, planID string, itemIDs []string) (err error) {
	url := fmt.Sprintf("%s/%s/%s/item_reorder", baseURL, plansPath(serviceTypeID), planID)

	body := RequestBody{
		Data: RequestData{
			Type: "PlanItemReorder",
			Attributes: map[string]any{
				"sequence": itemIDs,
			},
		},
	}

	_, err = NewRequest[struct{}](ctx, "POST", url, body)

	return
}
