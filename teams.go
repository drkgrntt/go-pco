package pco

import (
	"context"
	"fmt"
)

func teamsPath(serviceTypeID string) string {
	return fmt.Sprintf("%s/%s/teams", serviceTypesPath, serviceTypeID)
}

type TeamRelationships struct {
	ServiceType struct {
		Data *General `json:"data"`
	} `json:"service_type"`
	DefaultRespondsTo struct {
		Data *General `json:"data"`
	} `json:"default_responds_to"`
}

type TeamAttributes struct {
	Name                        string `json:"name"`
	Sequence                    int    `json:"sequence"`
	StageColor                  string `json:"stage_color"`
	StageVariant                string `json:"stage_variant"`
	SecureTeam                  bool   `json:"secure_team"`
	RehearsalTeam               bool   `json:"rehearsal_team"`
	AssignedDirectly            bool   `json:"assigned_directly"`
	DefaultPrepareNotifications bool   `json:"default_prepare_notifications"`
	// DefaultStatus is PCO's default PlanPerson status for new assignments on
	// this team - one of the same "C"/"U"/"D" (or word) values documented on
	// PlanPerson.status.
	DefaultStatus string `json:"default_status"`
	LastPlanFrom  string `json:"last_plan_from"`
	// ScheduleTo distinguishes plan-level teams ("plan", the default) from
	// split/"time"-level teams - not used yet, but worth keeping since it
	// changes what a PlanPerson assignment even means for this team.
	ScheduleTo string `json:"schedule_to"`
}

type TeamData struct {
	Type          string            `json:"type"`
	ID            string            `json:"id"`
	Attributes    TeamAttributes    `json:"attributes"`
	Relationships TeamRelationships `json:"relationships"`
}

type TeamListResponse struct {
	Data     []TeamData `json:"data"`
	Included []any      `json:"included"`
	Links    Links      `json:"links"`
	Meta     Meta       `json:"meta"`
}

type TeamsParams struct {
	OrderBy string
	PerPage int
	Offset  int
}

// GetTeams lists every team configured for a service type (e.g. "Worship",
// "Tech", "Hospitality") - read-only from this SDK's perspective; PCO doesn't
// document create/update/destroy for Team.
func GetTeams(ctx context.Context, serviceTypeID string, params *TeamsParams) (response TeamListResponse, err error) {
	if params == nil {
		params = &TeamsParams{}
	}

	q := NewQueryParams().OrderBy(params.OrderBy).PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, teamsPath(serviceTypeID), q.Encode())

	response, err = NewRequest[TeamListResponse](ctx, "GET", url, nil)

	return
}
