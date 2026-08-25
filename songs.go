package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const songsPath = "services/v2/songs"

type SongAttributes struct {
	Admin           string     `json:"admin"`
	Author          string     `json:"author"`
	CCLINumber      int        `json:"ccli_number"`
	Copyright       string     `json:"copyright"`
	CreatedAt       time.Time  `json:"created_at"`
	Hidden          bool       `json:"hidden"`
	LastScheduledAt *time.Time `json:"last_scheduled_at"`
	// LastScheduledShortDates is documented as a list of short date
	// strings, and usually null (no scheduled plans to report on), but
	// PCO has also been observed sending a single bare string rather than
	// a one-element array when there's exactly one - StringOrStrings
	// accepts either shape.
	LastScheduledShortDates StringOrStrings `json:"last_scheduled_short_dates"`
	Notes                   string          `json:"notes"`
	Themes                  string          `json:"themes"`
	Title                   string          `json:"title"`
	UpdatedAt               time.Time       `json:"updated_at"`
}

// StringOrStrings decodes a JSON field PCO sends inconsistently as either a
// single string or an array of strings (and sometimes null), normalizing it
// to a slice either way - single strings become a one-element slice, null
// becomes a nil slice.
type StringOrStrings []string

func (s *StringOrStrings) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = nil
		return nil
	}

	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = StringOrStrings{single}
		return nil
	}

	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*s = many
	return nil
}

// SongData has no Relationships field - unlike most other resources, PCO's
// Song vertex doesn't return a "relationships" object at all (related data
// like arrangements/attachments/tags are exposed as action links instead,
// under paths this SDK doesn't wrap yet).
type SongData struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Attributes SongAttributes `json:"attributes"`
}

type SongResponse struct {
	Data     SongData `json:"data"`
	Included []any    `json:"included"`
	Links    Links    `json:"links"`
	Meta     Meta     `json:"meta"`
}

type SongListResponse struct {
	Data     []SongData `json:"data"`
	Included []any      `json:"included"`
	Links    Links      `json:"links"`
	Meta     Meta       `json:"meta"`
}

type SongsParams struct {
	// OrderBy sorts by a can_order_by field: "title", "created_at",
	// "updated_at", or "last_scheduled_at". Prefix with "-" for descending.
	OrderBy string
	PerPage int
	Offset  int
}

func GetSongs(ctx context.Context, params *SongsParams) (response SongListResponse, err error) {
	if params == nil {
		params = &SongsParams{}
	}

	q := NewQueryParams().
		OrderBy(params.OrderBy).
		PerPage(params.PerPage).
		Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, songsPath, q.Encode())

	response, err = NewRequest[SongListResponse](ctx, "GET", url, nil)

	return
}

func GetSong(ctx context.Context, id string) (response SongResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, songsPath, id)

	response, err = NewRequest[SongResponse](ctx, "GET", url, nil)

	return
}

// CreateSongParams deliberately has no Notes field - PCO rejects "notes" on
// create outright ("notes cannot be assigned", confirmed live), unlike the
// other fields here which just default to PCO's own empty value when left
// unset. A song's notes are managed as their own sub-resource, not a plain
// Song attribute.
type CreateSongParams struct {
	Title      string
	Author     string
	Admin      string
	Copyright  string
	CCLINumber int
	Themes     string
	Hidden     bool
}

func CreateSong(ctx context.Context, params *CreateSongParams) (response SongResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s", baseURL, songsPath)

	attributes := map[string]any{
		"title": params.Title,
	}
	if params.Author != "" {
		attributes["author"] = params.Author
	}
	if params.Admin != "" {
		attributes["admin"] = params.Admin
	}
	if params.Copyright != "" {
		attributes["copyright"] = params.Copyright
	}
	if params.CCLINumber != 0 {
		attributes["ccli_number"] = params.CCLINumber
	}
	if params.Themes != "" {
		attributes["themes"] = params.Themes
	}
	if params.Hidden {
		attributes["hidden"] = params.Hidden
	}

	response, err = NewRequest[SongResponse](ctx, "POST", url, NewRequestBody(attributes))

	return
}

func DeleteSong(ctx context.Context, id string) (err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, songsPath, id)

	_, err = NewRequest[struct{}](ctx, "DELETE", url, nil)

	return
}
