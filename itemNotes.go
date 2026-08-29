package pco

import (
	"context"
	"fmt"
	"time"
)

func itemNotesPath(serviceTypeID, planID, itemID string) string {
	return fmt.Sprintf("%s/%s/item_notes", itemsPath(serviceTypeID, planID), itemID)
}

type ItemNoteRelationships struct {
	ItemNoteCategory struct {
		Data *General `json:"data"`
	} `json:"item_note_category"`
	Item struct {
		Data *General `json:"data"`
	} `json:"item"`
}

// ItemNoteAttributes - CategoryName is denormalized onto the note itself
// (confirmed live), so a caller only displaying existing notes doesn't
// need to separately fetch/join ItemNoteCategory by id.
type ItemNoteAttributes struct {
	CategoryName string    `json:"category_name"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ItemNoteData struct {
	Type          string                `json:"type"`
	ID            string                `json:"id"`
	Attributes    ItemNoteAttributes    `json:"attributes"`
	Relationships ItemNoteRelationships `json:"relationships"`
}

type ItemNoteResponse struct {
	Data     ItemNoteData `json:"data"`
	Included []any        `json:"included"`
	Links    Links        `json:"links"`
	Meta     Meta         `json:"meta"`
}

type ItemNoteListResponse struct {
	Data     []ItemNoteData `json:"data"`
	Included []any          `json:"included"`
	Links    Links          `json:"links"`
	Meta     Meta           `json:"meta"`
}

type ItemNotesParams struct {
	PerPage int
	Offset  int
}

// GetItemNotes lists every note already on itemID, across every category -
// there's no per-category filter param documented, so a caller wanting
// "the note for category X" filters this list client-side by
// Attributes.CategoryName (or the ItemNoteCategory relationship id).
func GetItemNotes(ctx context.Context, serviceTypeID, planID, itemID string, params *ItemNotesParams) (response ItemNoteListResponse, err error) {
	if params == nil {
		params = &ItemNotesParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, itemNotesPath(serviceTypeID, planID, itemID), q.Encode())

	response, err = NewRequest[ItemNoteListResponse](ctx, "GET", url, nil)

	return
}

// CreateItemNoteParams - ItemNoteCategoryID is required (PCO rejects a
// create with no category - confirmed live) and, per PCO's own
// documentation, can never be changed on an existing note afterward: to
// move a note to a different category, delete it and create a new one.
type CreateItemNoteParams struct {
	Content            string
	ItemNoteCategoryID string
}

func CreateItemNote(ctx context.Context, serviceTypeID, planID, itemID string, params *CreateItemNoteParams) (response ItemNoteResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s", baseURL, itemNotesPath(serviceTypeID, planID, itemID))

	attributes := map[string]any{
		"content": params.Content,
	}
	relationships := map[string]any{
		"item_note_category": map[string]any{
			"data": map[string]any{"type": "ItemNoteCategory", "id": params.ItemNoteCategoryID},
		},
	}

	response, err = NewRequest[ItemNoteResponse](ctx, "POST", url, NewRequestBodyWithRelationships(attributes, relationships))

	return
}

// UpdateItemNoteParams only has Content - Content is the only
// update_assignable attribute PCO documents for ItemNote, and the category
// relationship isn't reassignable at all after create (see
// CreateItemNoteParams's own doc comment).
type UpdateItemNoteParams struct {
	Content string
}

func UpdateItemNote(ctx context.Context, serviceTypeID, planID, itemID, noteID string, params *UpdateItemNoteParams) (response ItemNoteResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s/%s", baseURL, itemNotesPath(serviceTypeID, planID, itemID), noteID)

	attributes := map[string]any{
		"content": params.Content,
	}

	response, err = NewRequest[ItemNoteResponse](ctx, "PATCH", url, NewRequestBody(attributes))

	return
}

func DeleteItemNote(ctx context.Context, serviceTypeID, planID, itemID, noteID string) (err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, itemNotesPath(serviceTypeID, planID, itemID), noteID)

	_, err = NewRequest[struct{}](ctx, "DELETE", url, nil)

	return
}
