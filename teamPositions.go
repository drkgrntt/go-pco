package pco

import "fmt"

func teamPositionsPath(teamID string) string {
	return fmt.Sprintf("services/v2/teams/%s/team_positions", teamID)
}

type TeamPositionRelationships struct {
	Team struct {
		Data *General `json:"data"`
	} `json:"team"`
}

type TeamPositionAttributes struct {
	Name     string `json:"name"`
	Sequence int    `json:"sequence"`
}

type TeamPositionData struct {
	Type          string                    `json:"type"`
	ID            string                    `json:"id"`
	Attributes    TeamPositionAttributes    `json:"attributes"`
	Relationships TeamPositionRelationships `json:"relationships"`
}

type TeamPositionListResponse struct {
	Data     []TeamPositionData `json:"data"`
	Included []any              `json:"included"`
	Links    Links              `json:"links"`
	Meta     Meta               `json:"meta"`
}

type TeamPositionsParams struct {
	OrderBy string
	PerPage int
	Offset  int
}

// GetTeamPositions lists a team's positions (e.g. "Vocals", "Drums", "Sound
// Tech"). Read-only in PCO - positions are configured in Planning Center
// itself, not through this API.
func GetTeamPositions(teamID string, params *TeamPositionsParams) (response TeamPositionListResponse, err error) {
	if params == nil {
		params = &TeamPositionsParams{}
	}

	q := NewQueryParams().OrderBy(params.OrderBy).PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, teamPositionsPath(teamID), q.Encode())

	response, err = NewRequest[TeamPositionListResponse]("GET", url, nil)

	return
}
