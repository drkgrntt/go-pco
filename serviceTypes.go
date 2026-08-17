package pco

import (
	"fmt"
	"time"
)

const serviceTypesPath = "services/v2/service_types"

type ServiceTypeRelationships struct {
	Parent struct {
		Data *General `json:"data"`
	} `json:"parent"`
}

type ServiceTypeAttributes struct {
	ArchivedAt                 *time.Time `json:"archived_at"`
	AttachmentTypesEnabled     bool       `json:"attachment_types_enabled"`
	BackgroundCheckPermissions string     `json:"background_check_permissions"`
	CommentPermissions         string     `json:"comment_permissions"`
	CreatedAt                  time.Time  `json:"created_at"`
	CustomItemTypes            []string   `json:"custom_item_types"`
	DeletedAt                  *time.Time `json:"deleted_at"`
	Frequency                  string     `json:"frequency"`
	LastPlanFrom               string     `json:"last_plan_from"`
	Name                       string     `json:"name"`
	Permissions                string     `json:"permissions"`
	ScheduledPublish           bool       `json:"scheduled_publish"`
	Sequence                   int        `json:"sequence"`
	StandardItemTypes          []string   `json:"standard_item_types"`
	UpdatedAt                  time.Time  `json:"updated_at"`
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

func GetServiceTypes(params *ServiceTypesParams) (response ServiceTypeListResponse, err error) {
	if params == nil {
		params = &ServiceTypesParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, serviceTypesPath, q.Encode())

	response, err = NewRequest[ServiceTypeListResponse]("GET", url, nil)

	return
}

func GetServiceType(id string) (response ServiceTypeResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, serviceTypesPath, id)

	response, err = NewRequest[ServiceTypeResponse]("GET", url, nil)

	return
}

type CreateServiceTypeParams struct {
	Name string
}

func CreateServiceType(params *CreateServiceTypeParams) (response ServiceTypeResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s", baseURL, serviceTypesPath)

	body := NewRequestBody(map[string]any{
		"name": params.Name,
	})

	response, err = NewRequest[ServiceTypeResponse]("POST", url, body)

	return
}

type UpdateServiceTypeParams struct {
	Name string
}

func UpdateServiceType(id string, params *UpdateServiceTypeParams) (response ServiceTypeResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s/%s", baseURL, serviceTypesPath, id)

	body := NewRequestBody(map[string]any{
		"name": params.Name,
	})

	response, err = NewRequest[ServiceTypeResponse]("PATCH", url, body)

	return
}

func DeleteServiceType(id string) (err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, serviceTypesPath, id)

	_, err = NewRequest[struct{}]("DELETE", url, nil)

	return
}
