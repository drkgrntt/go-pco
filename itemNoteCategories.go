package pco

import (
	"context"
	"fmt"
	"time"
)

func itemNoteCategoriesPath(serviceTypeID string) string {
	return fmt.Sprintf("%s/%s/item_note_categories", serviceTypesPath, serviceTypeID)
}

// ItemNoteCategoryAttributes - a note category is configured once per
// Service Type (e.g. "Audio/Visual," "Band," "Vocals") and shared by every
// plan under it; individual ItemNotes (see itemNotes.go) each belong to
// exactly one of these.
type ItemNoteCategoryAttributes struct {
	CreatedAt      time.Time  `json:"created_at"`
	DeletedAt      *time.Time `json:"deleted_at"`
	FrequentlyUsed bool       `json:"frequently_used"`
	Name           string     `json:"name"`
	Sequence       int        `json:"sequence"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type ItemNoteCategoryData struct {
	Type       string                     `json:"type"`
	ID         string                     `json:"id"`
	Attributes ItemNoteCategoryAttributes `json:"attributes"`
}

type ItemNoteCategoryListResponse struct {
	Data     []ItemNoteCategoryData `json:"data"`
	Included []any                  `json:"included"`
	Links    Links                  `json:"links"`
	Meta     Meta                   `json:"meta"`
}

type ItemNoteCategoriesParams struct {
	PerPage int
	Offset  int
}

// GetItemNoteCategories lists every note category configured for
// serviceTypeID - confirmed live to be a short, org-configured list (e.g.
// 3-6 categories), so no per-page-heavy pagination concerns in practice.
func GetItemNoteCategories(ctx context.Context, serviceTypeID string, params *ItemNoteCategoriesParams) (response ItemNoteCategoryListResponse, err error) {
	if params == nil {
		params = &ItemNoteCategoriesParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, itemNoteCategoriesPath(serviceTypeID), q.Encode())

	response, err = NewRequest[ItemNoteCategoryListResponse](ctx, "GET", url, nil)

	return
}
