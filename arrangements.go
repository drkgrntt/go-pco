package pco

import (
	"context"
	"encoding/json"
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
	// Sequence is the plain section-label list ("Intro", "Verse 1",
	// "Chorus", ...) - SequenceShort is the same thing abbreviated and
	// run-length-collapsed ("Intro×3", "V1", "C", ...).
	Sequence []string `json:"sequence"`
	// SequenceFull is not a list of strings, despite PCO's own attribute
	// table saying "array" with no further detail (and despite Sequence/
	// SequenceShort actually being string arrays) - confirmed live, it's
	// one ArrangementSequenceStep per section, carrying that section's
	// timing.
	SequenceFull  []ArrangementSequenceStep `json:"sequence_full"`
	SequenceShort []string                  `json:"sequence_short"`
	UpdatedAt     time.Time                 `json:"updated_at"`
}

// ArrangementSequenceStep is one entry of ArrangementAttributes.SequenceFull
// - confirmed live against a real response. T is a timestamp string in the
// arrangement's own "mm:ss:ms"-ish format (e.g. "01:12:972"), left as a
// string rather than parsed, since PCO doesn't document its exact unit
// breakdown.
type ArrangementSequenceStep struct {
	Label string `json:"label"`
	// Number is usually a string (e.g. "2" for the second "Verse") but PCO
	// has also been observed sending it as a bare JSON number for at least
	// one real arrangement - StringOrNumber accepts either shape rather
	// than failing the whole arrangement fetch over one field's type.
	Number StringOrNumber `json:"number"`
	T      string         `json:"t"`
	SID    int            `json:"sid"`
}

// StringOrNumber decodes a JSON field PCO sends inconsistently as either a
// string or a bare number, normalizing it to a string either way - mirrors
// StringOrStrings (songs.go) for the same class of PCO inconsistency.
type StringOrNumber string

func (s *StringOrNumber) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = ""
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = StringOrNumber(str)
		return nil
	}

	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*s = StringOrNumber(num.String())
	return nil
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
	// Include adds related resources (e.g. "keys", "sections") to the
	// response's "included" array. See
	// https://developer.planning.center/docs/#/apps/services/2018-11-01/vertices/arrangement
	// for the full list PCO supports on this endpoint.
	Include []string
	PerPage int
	Offset  int
}

func GetArrangements(ctx context.Context, songID string, params *ArrangementsParams) (response ArrangementListResponse, err error) {
	if params == nil {
		params = &ArrangementsParams{}
	}

	q := NewQueryParams().
		OrderBy(params.OrderBy).
		Include(params.Include...).
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

// UpdateArrangementParams mirrors CreateArrangementParams - useful because
// PCO auto-creates one "Default Arrangement" the moment a Song itself is
// created (confirmed live: chord_chart_key/bpm/meter all start empty/zero),
// so setting a real key/tempo on a freshly-created song's arrangement is an
// update to that one, not a second CreateArrangement call.
type UpdateArrangementParams struct {
	Name          string
	BPM           float64
	Meter         string
	ChordChartKey string
	// Notes is a pointer, unlike every other field here - same reasoning
	// as UpdateItemParams (items.go): a plain string can't tell "leave
	// notes alone" apart from "clear notes to empty," since both would
	// serialize as the zero value. This is the one field of this set an
	// editable notes UI actually needs to be able to clear.
	Notes  *string
	Length int
}

func UpdateArrangement(ctx context.Context, songID, arrangementID string, params *UpdateArrangementParams) (response ArrangementResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s/%s", baseURL, arrangementsPath(songID), arrangementID)

	attributes := map[string]any{}
	if params.Name != "" {
		attributes["name"] = params.Name
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
	if params.Notes != nil {
		attributes["notes"] = *params.Notes
	}
	if params.Length != 0 {
		attributes["length"] = params.Length
	}

	response, err = NewRequest[ArrangementResponse](ctx, "PATCH", url, NewRequestBody(attributes))

	return
}
