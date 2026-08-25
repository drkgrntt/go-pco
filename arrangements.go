package pco

import (
	"context"
	"fmt"
	"time"
)

func arrangementsPath(songID string) string {
	return fmt.Sprintf("%s/%s/arrangements", songsPath, songID)
}

type ArrangementRelationships struct {
	Song struct {
		Data *General `json:"data"`
	} `json:"song"`
	CreatedBy struct {
		Data *General `json:"data"`
	} `json:"created_by"`
	UpdatedBy struct {
		Data *General `json:"data"`
	} `json:"updated_by"`
}

type ArrangementAttributes struct {
	ArchivedAt           *time.Time `json:"archived_at"`
	BPM                  float64    `json:"bpm"`
	ChordChart           string     `json:"chord_chart"`
	ChordChartChordColor int        `json:"chord_chart_chord_color"`
	ChordChartColumns    int        `json:"chord_chart_columns"`
	ChordChartFont       string     `json:"chord_chart_font"`
	ChordChartFontSize   int        `json:"chord_chart_font_size"`
	// ChordChartKey is the arrangement's own key as shown on its chord
	// chart - not the same thing as a Key sub-resource (see keys.go), which
	// models one of possibly several named keys (e.g. a capo/transposed
	// variant) an arrangement can offer.
	ChordChartKey       string    `json:"chord_chart_key"`
	CreatedAt           time.Time `json:"created_at"`
	HasChordChart       bool      `json:"has_chord_chart"`
	HasChords           bool      `json:"has_chords"`
	Length              int       `json:"length"`
	Lyrics              string    `json:"lyrics"`
	LyricsEnabled       bool      `json:"lyrics_enabled"`
	Meter               string    `json:"meter"`
	Name                string    `json:"name"`
	Notes               string    `json:"notes"`
	NumberChartEnabled  bool      `json:"number_chart_enabled"`
	NumeralChartEnabled bool      `json:"numeral_chart_enabled"`
	PrintMargin         string    `json:"print_margin"`
	PrintOrientation    string    `json:"print_orientation"`
	PrintPageSize       string    `json:"print_page_size"`
	Sequence            []string  `json:"sequence"`
	SequenceFull        []string  `json:"sequence_full"`
	SequenceShort       []string  `json:"sequence_short"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ArrangementData struct {
	Type          string                   `json:"type"`
	ID            string                   `json:"id"`
	Attributes    ArrangementAttributes    `json:"attributes"`
	Relationships ArrangementRelationships `json:"relationships"`
}

type ArrangementResponse struct {
	Data     ArrangementData `json:"data"`
	Included []any           `json:"included"`
	Links    Links           `json:"links"`
	Meta     Meta            `json:"meta"`
}

type ArrangementListResponse struct {
	Data     []ArrangementData `json:"data"`
	Included []any             `json:"included"`
	Links    Links             `json:"links"`
	Meta     Meta              `json:"meta"`
}

type ArrangementsParams struct {
	// OrderBy sorts by a can_order_by field: "created_at", "name", or
	// "updated_at". Prefix with "-" for descending.
	OrderBy string
	PerPage int
	Offset  int
}

func GetArrangements(ctx context.Context, songID string, params *ArrangementsParams) (response ArrangementListResponse, err error) {
	if params == nil {
		params = &ArrangementsParams{}
	}

	q := NewQueryParams().
		OrderBy(params.OrderBy).
		PerPage(params.PerPage).
		Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, arrangementsPath(songID), q.Encode())

	response, err = NewRequest[ArrangementListResponse](ctx, "GET", url, nil)

	return
}

func GetArrangement(ctx context.Context, songID, id string) (response ArrangementResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, arrangementsPath(songID), id)

	response, err = NewRequest[ArrangementResponse](ctx, "GET", url, nil)

	return
}

// CreateArrangementParams covers the fields worth setting when seeding an
// arrangement rather than PCO's full creatable set (chord chart
// formatting/print options are left at PCO's own defaults) - Name mirrors
// CreateSongParams.Title as the one field PCO actually requires.
type CreateArrangementParams struct {
	Name          string
	BPM           float64
	Meter         string
	ChordChartKey string
	Notes         string
	Length        int
}

func CreateArrangement(ctx context.Context, songID string, params *CreateArrangementParams) (response ArrangementResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s", baseURL, arrangementsPath(songID))

	attributes := map[string]any{
		"name": params.Name,
	}
	if params.BPM != 0 {
		attributes["bpm"] = params.BPM
	}
	if params.Meter != "" {
		attributes["meter"] = params.Meter
	}
	if params.ChordChartKey != "" {
		attributes["chord_chart_key"] = params.ChordChartKey
	}
	if params.Notes != "" {
		attributes["notes"] = params.Notes
	}
	if params.Length != 0 {
		attributes["length"] = params.Length
	}

	response, err = NewRequest[ArrangementResponse](ctx, "POST", url, NewRequestBody(attributes))

	return
}
