package pco

import (
	"context"
	"fmt"
)

func itemAssignmentsPath(serviceTypeID, planID, itemID string) string {
	return fmt.Sprintf("%s/%s/item_assignments", itemsPath(serviceTypeID, planID), itemID)
}

// ItemAssignmentAssignableType covers ItemAssignment.assignable_type's
// documented values (GET .../documentation/2018-11-01/vertices/
// item_assignment, confirmed live) - this is the same field PCO's own
// Services UI calls "Song leader" on a plan item's edit panel (there's no
// separate "song leader" resource; it's this one, scoped to a Person). PCO
// also allows assigning a TeamPosition (a placeholder to be filled later,
// the same idea as a Team's own open/needed position) rather than a
// specific person, but this SDK only exercises the Person case so far - a
// TeamPosition create hasn't been confirmed live.
const (
	ItemAssignmentAssignableTypePerson       = "Person"
	ItemAssignmentAssignableTypeTeamPosition = "TeamPosition"
)

type ItemAssignmentRelationships struct {
	Plan struct {
		Data *General `json:"data"`
	} `json:"plan"`
	Item struct {
		Data *General `json:"data"`
	} `json:"item"`
	// Assignable is polymorphic (Person or TeamPosition, per
	// AssignableType) - General's bare Type/ID shape covers either without
	// needing two separate relationship fields.
	Assignable struct {
		Data *General `json:"data"`
	} `json:"assignable"`
}

type ItemAssignmentAttributes struct {
	AssignableType string `json:"assignable_type"`
}

type ItemAssignmentData struct {
	Type          string                      `json:"type"`
	ID            string                      `json:"id"`
	Attributes    ItemAssignmentAttributes    `json:"attributes"`
	Relationships ItemAssignmentRelationships `json:"relationships"`
}

type ItemAssignmentResponse struct {
	Data     ItemAssignmentData `json:"data"`
	Included []any              `json:"included"`
	Links    Links              `json:"links"`
	Meta     Meta               `json:"meta"`
}

type ItemAssignmentListResponse struct {
	Data     []ItemAssignmentData `json:"data"`
	Included []any                `json:"included"`
	Links    Links                `json:"links"`
	Meta     Meta                 `json:"meta"`
}

type ItemAssignmentsParams struct {
	PerPage int
	Offset  int
}

// GetItemAssignments lists everyone (or every open position) assigned to
// itemID - for a song item, this is PCO's "Song leader" list. No
// documented query/order/include support (confirmed against the live
// vertex docs: can_query/can_order/can_include are all empty) - a caller
// wanting just the leader filters this client-side by
// Attributes.AssignableType/Relationships.Assignable.
func GetItemAssignments(ctx context.Context, serviceTypeID, planID, itemID string, params *ItemAssignmentsParams) (response ItemAssignmentListResponse, err error) {
	if params == nil {
		params = &ItemAssignmentsParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, itemAssignmentsPath(serviceTypeID, planID, itemID), q.Encode())

	response, err = NewRequest[ItemAssignmentListResponse](ctx, "GET", url, nil)

	return
}

// CreateItemAssignmentParams - PersonID is required; AssignableType
// defaults to "Person" when left empty, since that's the only case this
// SDK has confirmed live (see ItemAssignmentAssignableTypeTeamPosition's
// own doc comment). PCO documents assignable_type/assignable_id as plain
// *attributes* on create (create_assignable: ["assignable_type",
// "assignable_id"]), not a relationships object the way CreateTeamMember
// links person/team - confirmed by that same vertex doc, so this sends
// them as attributes rather than following CreateTeamMember's shape.
type CreateItemAssignmentParams struct {
	PersonID       string
	AssignableType string
}

// CreateItemAssignment assigns a person (or, per PCO's docs, an open
// TeamPosition placeholder - not yet exercised by this SDK) to a plan
// item - this is how a song gets its "Song leader" set. There's no update
// endpoint (PCO's vertex docs list can_update: false) - re-assigning is
// delete-and-recreate, the same convention CreateTeamMember/
// UpdateTeamMember already use for reassigning a person.
func CreateItemAssignment(ctx context.Context, serviceTypeID, planID, itemID string, params *CreateItemAssignmentParams) (response ItemAssignmentResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	assignableType := params.AssignableType
	if assignableType == "" {
		assignableType = ItemAssignmentAssignableTypePerson
	}

	url := fmt.Sprintf("%s/%s", baseURL, itemAssignmentsPath(serviceTypeID, planID, itemID))

	attributes := map[string]any{
		"assignable_type": assignableType,
		"assignable_id":   params.PersonID,
	}

	response, err = NewRequest[ItemAssignmentResponse](ctx, "POST", url, NewRequestBody(attributes))

	return
}

// DeleteItemAssignment removes one assignment (confirmed against the live
// vertex docs: can_destroy: true) - used both to clear a song's leader
// outright and as the first half of reassigning it to someone else (see
// CreateItemAssignment's own doc comment on why there's no update).
func DeleteItemAssignment(ctx context.Context, serviceTypeID, planID, itemID, assignmentID string) (err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, itemAssignmentsPath(serviceTypeID, planID, itemID), assignmentID)

	_, err = NewRequest[struct{}](ctx, "DELETE", url, nil)

	return
}
