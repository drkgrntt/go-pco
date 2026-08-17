package pco

import (
	"fmt"
	"time"
)

const peoplePath = "people/v2/people"

type PrimaryCampus struct {
	Data *General `json:"data"`
}

type Gender struct {
	Data *General `json:"data"`
}

type PersonRelationships struct {
	PrimaryCampus PrimaryCampus `json:"primary_campus"`
	Gender        Gender        `json:"gender"`
}

type PersonAttributes struct {
	Avatar                   string          `json:"avatar"`
	DemographicAvatarUrl     string          `json:"demographic_avatar_url"`
	FirstName                string          `json:"first_name"`
	Name                     string          `json:"name"`
	Status                   string          `json:"status"`
	RemoteId                 int             `json:"remote_id"`
	AccountingAdministrator  bool            `json:"accounting_administrator"`
	Anniversary              string          `json:"anniversary"`
	Birthdate                string          `json:"birthdate"`
	Child                    bool            `json:"child"`
	GivenName                string          `json:"given_name"`
	Grade                    int             `json:"grade"`
	GraduationYear           int             `json:"graduation_year"`
	LastName                 string          `json:"last_name"`
	MiddleName               string          `json:"middle_name"`
	Nickname                 string          `json:"nickname"`
	PeoplePermissions        string          `json:"people_permissions"`
	SiteAdministrator        bool            `json:"site_administrator"`
	Gender                   string          `json:"gender"`
	InactivatedAt            string          `json:"inactivated_at"`
	MedicalNotes             string          `json:"medical_notes"`
	Membership               string          `json:"membership"`
	CreatedAt                time.Time       `json:"created_at"`
	UpdatedAt                time.Time       `json:"updated_at"`
	CanCreateForms           bool            `json:"can_create_forms"`
	CanEmailLists            bool            `json:"can_email_lists"`
	DirectorySharedInfo      string          `json:"directory_shared_info"`
	DirectoryStatus          string          `json:"directory_status"`
	PassedBackgroundCheck    bool            `json:"passed_background_check"`
	ResourcePermissionFlags  map[string]bool `json:"resource_permission_flags"`
	SchoolType               string          `json:"school_type"`
	LoginIdentifier          string          `json:"login_identifier"`
	MfaConfigured            bool            `json:"mfa_configured"`
	StripeCustomerIdentifier string          `json:"stripe_customer_identifier"`
}

type PersonData struct {
	Type          string              `json:"type"`
	ID            string              `json:"id"`
	Attributes    PersonAttributes    `json:"attributes"`
	Relationships PersonRelationships `json:"relationships"`
}

type PersonListResponse struct {
	Data     []PersonData `json:"data"`
	Included []any        `json:"included"`
	Links    Links        `json:"links"`
	Meta     Meta         `json:"meta"`
}

type PersonCreateResponse struct {
	Data     PersonData `json:"data"`
	Included []any      `json:"included"`
	Links    Links      `json:"links"`
	Meta     Meta       `json:"meta"`
}

type PeopleParams struct {
	FirstName         string
	LastName          string
	Email             string
	SearchPhoneNumber string
	PerPage           int
	Offset            int
}

func GetPeople(params *PeopleParams) (response PersonListResponse, err error) {
	if params == nil {
		params = &PeopleParams{}
	}

	q := NewQueryParams().
		Where("first_name", params.FirstName).
		Where("last_name", params.LastName).
		Where("search_name_or_email", params.Email).
		Where("search_phone_number", params.SearchPhoneNumber).
		PerPage(params.PerPage).
		Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, peoplePath, q.Encode())

	response, err = NewRequest[PersonListResponse]("GET", url, nil)

	return
}

type CreatePersonParams struct {
	FirstName string
	LastName  string
}

func CreatePerson(params *CreatePersonParams) (response PersonCreateResponse, err error) {
	if params == nil {
		return response, fmt.Errorf("params cannot be nil")
	}

	url := fmt.Sprintf("%s/%s", baseURL, peoplePath)

	body := NewRequestBody(map[string]any{
		"first_name": params.FirstName,
		"last_name":  params.LastName,
	})

	response, err = NewRequest[PersonCreateResponse]("POST", url, body)

	return
}
