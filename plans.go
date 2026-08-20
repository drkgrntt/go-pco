package pco

import (
	"context"
	"fmt"
	"time"
)

func plansPath(serviceTypeID string) string {
	return fmt.Sprintf("%s/%s/plans", serviceTypesPath, serviceTypeID)
}

type PlanRelationships struct {
	ServiceType struct {
		Data *General `json:"data"`
	} `json:"service_type"`
	Series struct {
		Data *General `json:"data"`
	} `json:"series"`
	PreviousPlan struct {
		Data *General `json:"data"`
	} `json:"previous_plan"`
	NextPlan struct {
		Data *General `json:"data"`
	} `json:"next_plan"`
	CreatedBy struct {
		Data *General `json:"data"`
	} `json:"created_by"`
	UpdatedBy struct {
		Data *General `json:"data"`
	} `json:"updated_by"`
}

type PlanAttributes struct {
	Dates                string    `json:"dates"`
	ItemsCount           int       `json:"items_count"`
	NeededPositionsCount int       `json:"needed_positions_count"`
	PlanPeopleCount      int       `json:"plan_people_count"`
	Public               bool      `json:"public"`
	RemindersDisabled    bool      `json:"reminders_disabled"`
	SeriesTitle          string    `json:"series_title"`
	SortDate             time.Time `json:"sort_date"`
	Title                string    `json:"title"`
}

type PlanData struct {
	Type          string            `json:"type"`
	ID            string            `json:"id"`
	Attributes    PlanAttributes    `json:"attributes"`
	Relationships PlanRelationships `json:"relationships"`
}

type PlanResponse struct {
	Data     PlanData `json:"data"`
	Included []any    `json:"included"`
	Links    Links    `json:"links"`
	Meta     Meta     `json:"meta"`
}

type PlanListResponse struct {
	Data     []PlanData `json:"data"`
	Included []any      `json:"included"`
	Links    Links      `json:"links"`
	Meta     Meta       `json:"meta"`
}

type PlansParams struct {
	// OrderBy sorts by a can_order_by field: "title", "created_at",
	// "updated_at", or "sort_date". Prefix with "-" for descending.
	OrderBy string
	// Filter is PCO's named filter for this endpoint: "past" or "future".
	Filter  string
	PerPage int
	Offset  int
}

func GetPlans(ctx context.Context, serviceTypeID string, params *PlansParams) (response PlanListResponse, err error) {
	if params == nil {
		params = &PlansParams{}
	}

	q := NewQueryParams().
		OrderBy(params.OrderBy).
		Filter(params.Filter).
		PerPage(params.PerPage).
		Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, plansPath(serviceTypeID), q.Encode())

	response, err = NewRequest[PlanListResponse](ctx, "GET", url, nil)

	return
}

func GetPlan(ctx context.Context, serviceTypeID, planID string) (response PlanResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, plansPath(serviceTypeID), planID)

	response, err = NewRequest[PlanResponse](ctx, "GET", url, nil)

	return
}

type CreatePlanParams struct {
	Title       string
	Public      bool
	SeriesID    string
	SeriesTitle string
}

func CreatePlan(ctx context.Context, serviceTypeID string, params *CreatePlanParams) (response PlanResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s", baseURL, plansPath(serviceTypeID))

	attributes := map[string]any{
		"title":  params.Title,
		"public": params.Public,
	}
	if params.SeriesID != "" {
		attributes["series_id"] = params.SeriesID
	}
	if params.SeriesTitle != "" {
		attributes["series_title"] = params.SeriesTitle
	}

	response, err = NewRequest[PlanResponse](ctx, "POST", url, NewRequestBody(attributes))

	return
}

type UpdatePlanParams struct {
	Title             string
	Public            *bool
	RemindersDisabled *bool
}

func UpdatePlan(ctx context.Context, serviceTypeID, planID string, params *UpdatePlanParams) (response PlanResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s/%s", baseURL, plansPath(serviceTypeID), planID)

	attributes := map[string]any{}
	if params.Title != "" {
		attributes["title"] = params.Title
	}
	if params.Public != nil {
		attributes["public"] = *params.Public
	}
	if params.RemindersDisabled != nil {
		attributes["reminders_disabled"] = *params.RemindersDisabled
	}

	response, err = NewRequest[PlanResponse](ctx, "PATCH", url, NewRequestBody(attributes))

	return
}

func DeletePlan(ctx context.Context, serviceTypeID, planID string) (err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, plansPath(serviceTypeID), planID)

	_, err = NewRequest[struct{}](ctx, "DELETE", url, nil)

	return
}
