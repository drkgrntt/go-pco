package pco

import (
	"context"
	"fmt"
)

// teamMembersPath is PlanPerson scoped to a plan - PCO's "who's actually
// filling positions on this plan" list (as opposed to needed_positions,
// which is "what's still open"). Used for both create and delete - PCO's
// docs point delete at the person-scoped planPeoplePath below instead, but
// that path returned 404 on a real (older/past) plan's assignment when this
// one didn't; confirmed live that the plan-scoped path deletes reliably
// regardless of the plan's age, so it's used for both.
func teamMembersPath(serviceTypeID, planID string) string {
	return fmt.Sprintf("%s/%s/team_members", plansPath(serviceTypeID), planID)
}

// planPeoplePath is the person-scoped path used for a person's own
// PlanPerson history (see GetPersonPlanPeople) - PCO's docs also point
// update/delete at this path, but see teamMembersPath's doc comment for why
// this SDK's DeleteTeamMember uses the plan-scoped path instead.
func planPeoplePath(personID string) string {
	return fmt.Sprintf("services/v2/people/%s/plan_people", personID)
}

type PlanPersonRelationships struct {
	Person struct {
		Data *General `json:"data"`
	} `json:"person"`
	Plan struct {
		Data *General `json:"data"`
	} `json:"plan"`
	Team struct {
		Data *General `json:"data"`
	} `json:"team"`
}

// PlanPersonAttributes covers a filled position assignment. Status is one of
// PCO's documented values (letter code or word) - see the PlanPersonStatus*
// consts. TeamPositionName is plain text, matching NeededPosition's shape,
// not a relationship to a TeamPosition id.
type PlanPersonAttributes struct {
	Name             string `json:"name"`
	Status           string `json:"status"`
	DeclineReason    string `json:"decline_reason"`
	Notes            string `json:"notes"`
	TeamPositionName string `json:"team_position_name"`
}

type PlanPersonData struct {
	Type          string                  `json:"type"`
	ID            string                  `json:"id"`
	Attributes    PlanPersonAttributes    `json:"attributes"`
	Relationships PlanPersonRelationships `json:"relationships"`
}

type PlanPersonListResponse struct {
	Data     []PlanPersonData `json:"data"`
	Included []any            `json:"included"`
	Links    Links            `json:"links"`
	Meta     Meta             `json:"meta"`
}

type PlanPersonResponse struct {
	Data     PlanPersonData `json:"data"`
	Included []any          `json:"included"`
	Links    Links          `json:"links"`
	Meta     Meta           `json:"meta"`
}

// PlanPersonStatus* covers PlanPerson.status's documented values - PCO
// accepts either the letter code or the word, and returns the letter code
// in responses (confirmed live: a created assignment came back with
// "status": "U"), so these consts use that form.
const (
	PlanPersonStatusConfirmed   = "C"
	PlanPersonStatusUnconfirmed = "U"
	PlanPersonStatusDeclined    = "D"
)

type TeamMembersParams struct {
	PerPage int
	Offset  int
}

// GetTeamMembers lists every person currently assigned to fill a position on
// this plan (any status - confirmed, unconfirmed, or declined).
func GetTeamMembers(ctx context.Context, serviceTypeID, planID string, params *TeamMembersParams) (response PlanPersonListResponse, err error) {
	if params == nil {
		params = &TeamMembersParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, teamMembersPath(serviceTypeID, planID), q.Encode())

	response, err = NewRequest[PlanPersonListResponse](ctx, "GET", url, nil)

	return
}

type CreateTeamMemberParams struct {
	PersonID         string
	TeamID           string
	TeamPositionName string
	// Status defaults to PlanPersonStatusUnconfirmed when left empty - a new
	// assignment this app creates hasn't actually been confirmed by anyone
	// yet, so that's the honest default rather than guessing Confirmed.
	Status string
	// Notes is confirmed both create- and update-assignable against PCO's
	// own documentation API (GET .../documentation/2018-11-01/vertices/
	// plan_person, its permissions.create_assignable/update_assignable
	// lists both include "notes") and against the live API directly (a
	// created assignment's notes came back exactly as sent) - unlike
	// Item.Sequence, which is documented as assignable but rejected live,
	// this one actually works both ways.
	Notes string
}

// CreateTeamMember assigns a person to a position on a plan - the
// scheduling-side counterpart to CreateItem for songs. Confirmed against
// PCO's documented "Assignable on Create" fields for PlanPerson
// (person_id/team_id/status/team_position_name/notes via
// relationships+attributes on the plan-scoped team_members path).
func CreateTeamMember(ctx context.Context, serviceTypeID, planID string, params *CreateTeamMemberParams) (response PlanPersonResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	status := params.Status
	if status == "" {
		status = PlanPersonStatusUnconfirmed
	}

	url := fmt.Sprintf("%s/%s", baseURL, teamMembersPath(serviceTypeID, planID))

	attributes := map[string]any{
		"status":             status,
		"team_position_name": params.TeamPositionName,
		"notes":              params.Notes,
	}
	relationships := map[string]any{
		"team": map[string]any{
			"data": map[string]any{"type": "Team", "id": params.TeamID},
		},
		"person": map[string]any{
			"data": map[string]any{"type": "Person", "id": params.PersonID},
		},
	}

	response, err = NewRequest[PlanPersonResponse](ctx, "POST", url, NewRequestBodyWithRelationships(attributes, relationships))

	return
}

// UpdateTeamMemberParams is a partial update - only set fields are sent,
// mirroring UpdateItemParams. Deliberately doesn't cover every field PCO's
// update_assignable list allows (person_id/team_id are in that list too,
// meaning PCO would let you re-point an existing assignment at a different
// person/team entirely) - re-assigning who's filling a position is better
// modeled as delete-and-recreate, matching how the rest of this SDK treats
// person/song relationships as fixed at creation, not something to sneak
// into a "partial update" call.
type UpdateTeamMemberParams struct {
	Status           string
	DeclineReason    string
	Notes            string
	TeamPositionName string
}

// UpdateTeamMember patches an existing plan assignment - confirmed live
// against the real API (a PATCH with a new notes value came back with that
// exact value on a subsequent GET), addressed through the same plan-scoped
// path CreateTeamMember/DeleteTeamMember already use (see teamMembersPath's
// doc comment for why this SDK avoids the person-scoped path PCO's docs
// otherwise point update/delete at).
func UpdateTeamMember(ctx context.Context, serviceTypeID, planID, planPersonID string, params *UpdateTeamMemberParams) (response PlanPersonResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s/%s", baseURL, teamMembersPath(serviceTypeID, planID), planPersonID)

	attributes := map[string]any{}
	if params.Status != "" {
		attributes["status"] = params.Status
	}
	if params.DeclineReason != "" {
		attributes["decline_reason"] = params.DeclineReason
	}
	if params.Notes != "" {
		attributes["notes"] = params.Notes
	}
	if params.TeamPositionName != "" {
		attributes["team_position_name"] = params.TeamPositionName
	}

	response, err = NewRequest[PlanPersonResponse](ctx, "PATCH", url, NewRequestBody(attributes))

	return
}

// DeleteTeamMember removes a plan assignment, addressed through the same
// plan-scoped path it was created on - see teamMembersPath's doc comment
// for why this SDK doesn't use PCO's documented person-scoped delete path.
func DeleteTeamMember(ctx context.Context, serviceTypeID, planID, planPersonID string) (err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, teamMembersPath(serviceTypeID, planID), planPersonID)

	_, err = NewRequest[struct{}](ctx, "DELETE", url, nil)

	return
}

type PersonPlanPeopleParams struct {
	// TeamID filters to assignments on one team (confirmed live: this
	// endpoint's can_query_by includes "team_id", sent as where[team_id]).
	TeamID string
	// Include adds related resources to the response's "included" array -
	// pass "plan" to get each assignment's plan (with its sort_date) in one
	// call instead of a separate GetPlan per assignment.
	Include []string
	PerPage int
	Offset  int
}

// GetPersonPlanPeople lists a person's own PlanPerson history - every plan
// they've been assigned to, across any team - filterable to one team via
// TeamID. This is how the church app looks up someone's past serving dates
// for a position to check against their schedule_preference.
func GetPersonPlanPeople(ctx context.Context, personID string, params *PersonPlanPeopleParams) (response PlanPersonListResponse, err error) {
	if params == nil {
		params = &PersonPlanPeopleParams{}
	}

	q := NewQueryParams().
		Where("team_id", params.TeamID).
		Include(params.Include...).
		PerPage(params.PerPage).
		Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, planPeoplePath(personID), q.Encode())

	response, err = NewRequest[PlanPersonListResponse](ctx, "GET", url, nil)

	return
}
