package pco

import (
	"fmt"
	"time"
)

// blockoutsPath is scoped under services/v2/people, not the People app's own
// people/v2/people (peoplePath in people.go) - Services exposes its own
// "people" sub-resource keyed by the same organization-wide person id, and
// blockouts (like team_members and person_team_position_assignments) only
// exist on that side of the API.
func blockoutsPath(personID string) string {
	return fmt.Sprintf("services/v2/people/%s/blockouts", personID)
}

type BlockoutRelationships struct {
	Person struct {
		Data *General `json:"data"`
	} `json:"person"`
}

// BlockoutAttributes covers a person's self-declared unavailability window in
// Planning Center Services. StartsAt/EndsAt bound a single (non-recurring)
// window; RepeatFrequency/RepeatInterval/RepeatPeriod/RepeatUntil describe a
// recurring blockout, which this SDK exposes but the church app's warning
// logic doesn't expand yet (see blockoutWarning's doc comment).
type BlockoutAttributes struct {
	Reason          string    `json:"reason"`
	StartsAt        time.Time `json:"starts_at"`
	EndsAt          time.Time `json:"ends_at"`
	RepeatFrequency string    `json:"repeat_frequency"`
	RepeatInterval  int       `json:"repeat_interval"`
	RepeatPeriod    string    `json:"repeat_period"`
	RepeatUntil     string    `json:"repeat_until"`
	GroupIdentifier string    `json:"group_identifier"`
}

type BlockoutData struct {
	Type          string                `json:"type"`
	ID            string                `json:"id"`
	Attributes    BlockoutAttributes    `json:"attributes"`
	Relationships BlockoutRelationships `json:"relationships"`
}

type BlockoutListResponse struct {
	Data     []BlockoutData `json:"data"`
	Included []any          `json:"included"`
	Links    Links          `json:"links"`
	Meta     Meta           `json:"meta"`
}

type BlockoutsParams struct {
	// Filter is PCO's named filter for this endpoint: "past" or "future".
	Filter  string
	PerPage int
	Offset  int
}

// GetBlockouts lists a person's self-declared unavailability windows - used
// read-only here (see the church app's blockoutWarning) to flag scheduling
// someone during a date they've already marked unavailable. This SDK doesn't
// create/update/destroy blockouts; that stays a volunteer's own action in
// Planning Center.
func GetBlockouts(personID string, params *BlockoutsParams) (response BlockoutListResponse, err error) {
	if params == nil {
		params = &BlockoutsParams{}
	}

	q := NewQueryParams().Filter(params.Filter).PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, blockoutsPath(personID), q.Encode())

	response, err = NewRequest[BlockoutListResponse]("GET", url, nil)

	return
}
