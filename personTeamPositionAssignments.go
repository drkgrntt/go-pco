package pco

import "fmt"

// personTeamPositionAssignmentsPath is the team_position-scoped path used to
// list and create assignments; PCO documents update/delete against the same
// nested path plus the assignment id (unlike PlanPerson, which switches to a
// person-scoped path for those). TeamPosition isn't addressable as a
// top-level resource (confirmed live: /services/v2/team_positions/{id}
// 404s) - it only exists nested under its team, same as GetTeamPositions.
func personTeamPositionAssignmentsPath(teamID, teamPositionID string) string {
	return fmt.Sprintf("%s/%s/person_team_position_assignments", teamPositionsPath(teamID), teamPositionID)
}

type PersonTeamPositionAssignmentRelationships struct {
	Person struct {
		Data *General `json:"data"`
	} `json:"person"`
	TeamPosition struct {
		Data *General `json:"data"`
	} `json:"team_position"`
}

// PersonTeamPositionAssignmentAttributes is both "this person can serve this
// position" and, via SchedulePreference, how often they want to - see the
// SchedulePreference* consts for the exact documented values. PreferredWeeks
// only applies when SchedulePreference is SchedulePreferenceChooseWeeks.
type PersonTeamPositionAssignmentAttributes struct {
	SchedulePreference string   `json:"schedule_preference"`
	PreferredWeeks     []string `json:"preferred_weeks"`
}

type PersonTeamPositionAssignmentData struct {
	Type          string                                    `json:"type"`
	ID            string                                    `json:"id"`
	Attributes    PersonTeamPositionAssignmentAttributes    `json:"attributes"`
	Relationships PersonTeamPositionAssignmentRelationships `json:"relationships"`
}

type PersonTeamPositionAssignmentListResponse struct {
	Data     []PersonTeamPositionAssignmentData `json:"data"`
	Included []any                              `json:"included"`
	Links    Links                              `json:"links"`
	Meta     Meta                               `json:"meta"`
}

type PersonTeamPositionAssignmentResponse struct {
	Data     PersonTeamPositionAssignmentData `json:"data"`
	Included []any                            `json:"included"`
	Links    Links                            `json:"links"`
	Meta     Meta                             `json:"meta"`
}

// SchedulePreference* is the verbatim, complete set of values PCO documents
// for PersonTeamPositionAssignment.schedule_preference. EveryWeek..
// Every6thWeek cover "1 every X weeks"; OnceAMonth../ThreeTimesAMonth cover
// "X weeks per month" - between them, exactly the two modes asked for.
// OnceAQuarter and ChooseWeeks are PCO options beyond those two modes, kept
// available but not specially interpreted by this app's frequency warning.
const (
	SchedulePreferenceUnavailable      = "Unavailable"
	SchedulePreferenceEveryWeek        = "Every week"
	SchedulePreferenceEveryOtherWeek   = "Every other week"
	SchedulePreferenceEvery3rdWeek     = "Every 3rd week"
	SchedulePreferenceEvery4thWeek     = "Every 4th week"
	SchedulePreferenceEvery5thWeek     = "Every 5th week"
	SchedulePreferenceEvery6thWeek     = "Every 6th week"
	SchedulePreferenceOnceAMonth       = "Once a month"
	SchedulePreferenceTwiceAMonth      = "Twice a month"
	SchedulePreferenceThreeTimesAMonth = "Three times a month"
	SchedulePreferenceOnceAQuarter     = "Once a quarter"
	SchedulePreferenceChooseWeeks      = "Choose Weeks"
)

type PersonTeamPositionAssignmentsParams struct {
	PerPage int
	Offset  int
}

// GetPersonTeamPositionAssignments lists every person eligible for a
// position, along with their schedule_preference - this doubles as both the
// position's candidate pool for the Team Builder (mirroring how the full
// song library is Build a Set's candidate pool) and the source of each
// candidate's serving-frequency preference.
func GetPersonTeamPositionAssignments(teamID, teamPositionID string, params *PersonTeamPositionAssignmentsParams) (response PersonTeamPositionAssignmentListResponse, err error) {
	if params == nil {
		params = &PersonTeamPositionAssignmentsParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, personTeamPositionAssignmentsPath(teamID, teamPositionID), q.Encode())

	response, err = NewRequest[PersonTeamPositionAssignmentListResponse]("GET", url, nil)

	return
}

type CreatePersonTeamPositionAssignmentParams struct {
	PersonID           string
	SchedulePreference string
}

// CreatePersonTeamPositionAssignment adds a person to a position's eligible
// roster with a serving-frequency preference in one call - PCO doesn't
// separate "can serve this position" from "how often," so setting a
// preference for someone not yet eligible for the position also makes them
// eligible.
func CreatePersonTeamPositionAssignment(teamID, teamPositionID string, params *CreatePersonTeamPositionAssignmentParams) (response PersonTeamPositionAssignmentResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s", baseURL, personTeamPositionAssignmentsPath(teamID, teamPositionID))

	attributes := map[string]any{
		"schedule_preference": params.SchedulePreference,
	}
	relationships := map[string]any{
		"person": map[string]any{
			"data": map[string]any{"type": "Person", "id": params.PersonID},
		},
	}

	response, err = NewRequest[PersonTeamPositionAssignmentResponse]("POST", url, NewRequestBodyWithRelationships(attributes, relationships))

	return
}

type UpdatePersonTeamPositionAssignmentParams struct {
	SchedulePreference string
}

// UpdatePersonTeamPositionAssignment changes an existing assignment's
// serving-frequency preference.
func UpdatePersonTeamPositionAssignment(teamID, teamPositionID, assignmentID string, params *UpdatePersonTeamPositionAssignmentParams) (response PersonTeamPositionAssignmentResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s/%s", baseURL, personTeamPositionAssignmentsPath(teamID, teamPositionID), assignmentID)

	attributes := map[string]any{
		"schedule_preference": params.SchedulePreference,
	}

	response, err = NewRequest[PersonTeamPositionAssignmentResponse]("PATCH", url, NewRequestBody(attributes))

	return
}

// DeletePersonTeamPositionAssignment removes a person's eligibility (and
// preference) for a position entirely.
func DeletePersonTeamPositionAssignment(teamID, teamPositionID, assignmentID string) (err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, personTeamPositionAssignmentsPath(teamID, teamPositionID), assignmentID)

	_, err = NewRequest[struct{}]("DELETE", url, nil)

	return
}
