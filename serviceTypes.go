package pco

import (
	"context"
	"fmt"
	"time"
)

const serviceTypesPath = "services/v2/service_types"

type ServiceTypeRelationships struct {
	Parent struct {
		Data *General `json:"data"`
	} `json:"parent"`
}

// ItemTypeDefinition is one entry of a service type's item type palette -
// the row/color PCO's UI uses for a given item type (e.g. "Song", "Header").
// Both CustomItemTypes and StandardItemTypes use this shape; PCO's docs
// describe them identically, though only StandardItemTypes has been
// observed live (this account has no custom item types configured).
type ItemTypeDefinition struct {
	Name  string `json:"name"`
	Index int    `json:"index"`
	Color string `json:"color"`
}

type ServiceTypeAttributes struct {
	ArchivedAt                 *time.Time           `json:"archived_at"`
	AttachmentTypesEnabled     bool                 `json:"attachment_types_enabled"`
	BackgroundCheckPermissions string               `json:"background_check_permissions"`
	CommentPermissions         string               `json:"comment_permissions"`
	CreatedAt                  time.Time            `json:"created_at"`
	CustomItemTypes            []ItemTypeDefinition `json:"custom_item_types"`
	DeletedAt                  *time.Time           `json:"deleted_at"`
	Frequency                  string               `json:"frequency"`
	LastPlanFrom               string               `json:"last_plan_from"`
	Name                       string               `json:"name"`
	Permissions                string               `json:"permissions"`
	ScheduledPublish           bool                 `json:"scheduled_publish"`
	Sequence                   int                  `json:"sequence"`
	StandardItemTypes          []ItemTypeDefinition `json:"standard_item_types"`
	UpdatedAt                  time.Time            `json:"updated_at"`
}

type ServiceTypeData struct {
	Type          string                   `json:"type"`
	ID            string                   `json:"id"`
	Attributes    ServiceTypeAttributes    `json:"attributes"`
	Relationships ServiceTypeRelationships `json:"relationships"`
}

type ServiceTypeResponse struct {
	Data     ServiceTypeData `json:"data"`
	Included []any           `json:"included"`
	Links    Links           `json:"links"`
	Meta     Meta            `json:"meta"`
}

type ServiceTypeListResponse struct {
	Data     []ServiceTypeData `json:"data"`
	Included []any             `json:"included"`
	Links    Links             `json:"links"`
	Meta     Meta              `json:"meta"`
}

type ServiceTypesParams struct {
	PerPage int
	Offset  int
}

func GetServiceTypes(ctx context.Context, params *ServiceTypesParams) (response ServiceTypeListResponse, err error) {
	if params == nil {
		params = &ServiceTypesParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, serviceTypesPath, q.Encode())

	response, err = NewRequest[ServiceTypeListResponse](ctx, "GET", url, nil)

	return
}

func GetServiceType(ctx context.Context, id string) (response ServiceTypeResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, serviceTypesPath, id)

	response, err = NewRequest[ServiceTypeResponse](ctx, "GET", url, nil)

	return
}

type CreateServiceTypeParams struct {
	Name string
}

func CreateServiceType(ctx context.Context, params *CreateServiceTypeParams) (response ServiceTypeResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s", baseURL, serviceTypesPath)

	body := NewRequestBody(map[string]any{
		"name": params.Name,
	})

	response, err = NewRequest[ServiceTypeResponse](ctx, "POST", url, body)

	return
}

type UpdateServiceTypeParams struct {
	Name string
}

func UpdateServiceType(ctx context.Context, id string, params *UpdateServiceTypeParams) (response ServiceTypeResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s/%s", baseURL, serviceTypesPath, id)

	body := NewRequestBody(map[string]any{
		"name": params.Name,
	})

	response, err = NewRequest[ServiceTypeResponse](ctx, "PATCH", url, body)

	return
}

func DeleteServiceType(ctx context.Context, id string) (err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, serviceTypesPath, id)

	_, err = NewRequest[struct{}](ctx, "DELETE", url, nil)

	return
}
