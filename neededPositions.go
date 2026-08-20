package pco

import "fmt"

func neededPositionsPath(serviceTypeID, planID string) string {
	return fmt.Sprintf("%s/%s/needed_positions", plansPath(serviceTypeID), planID)
}

type NeededPositionRelationships struct {
	Team struct {
		Data *General `json:"data"`
	} `json:"team"`
	Plan struct {
		Data *General `json:"data"`
	} `json:"plan"`
}

// NeededPositionAttributes describes one open slot a plan needs filled -
// e.g. "2 Vocals needed on the Worship team." TeamPositionName is plain text
// (matching the position's name at the time the slot was configured in
// Planning Center), not a relationship to a TeamPosition id - the same
// PlanPerson.team_position_name shape reused here.
type NeededPositionAttributes struct {
	Quantity         int    `json:"quantity"`
	TeamPositionName string `json:"team_position_name"`
	ScheduledTo      string `json:"scheduled_to"`
}

type NeededPositionData struct {
	Type          string                      `json:"type"`
	ID            string                      `json:"id"`
	Attributes    NeededPositionAttributes    `json:"attributes"`
	Relationships NeededPositionRelationships `json:"relationships"`
}

type NeededPositionListResponse struct {
	Data     []NeededPositionData `json:"data"`
	Included []any                `json:"included"`
	Links    Links                `json:"links"`
	Meta     Meta                 `json:"meta"`
}

type NeededPositionsParams struct {
	PerPage int
	Offset  int
}

// GetNeededPositions lists a plan's open position slots (team + position name
// + how many are needed) - the plan-side half of what the Team Builder merges
// with actual filled assignments (see GetTeamMembers), mirroring how
// GetItems' order-of-service data is merged with a local set.
func GetNeededPositions(serviceTypeID, planID string, params *NeededPositionsParams) (response NeededPositionListResponse, err error) {
	if params == nil {
		params = &NeededPositionsParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, neededPositionsPath(serviceTypeID, planID), q.Encode())

	response, err = NewRequest[NeededPositionListResponse]("GET", url, nil)

	return
}
