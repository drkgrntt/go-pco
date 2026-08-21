package pco

import (
	"context"
	"fmt"
)

// organizationPath is the People API's root - unlike every other resource
// here, the organization isn't addressed by its own id segment (there's
// only ever the one an access token is scoped to); GET-ing the People API
// root directly returns it as the response's `data` (confirmed against the
// live API: PCO's docs list Organization as a People API vertex but don't
// spell out its path, and /people/v2/organization 404s).
const organizationPath = "people/v2"

// OrganizationAttributes covers the fields this app actually has a use for
// (chiefly Name) plus the handful PCO always sends alongside it - not
// every attribute PCO's docs list for Organization (e.g. Grades, used by
// children's ministry check-in, has no bearing here).
type OrganizationAttributes struct {
	Name                  string `json:"name"`
	ChurchCenterSubdomain string `json:"church_center_subdomain"`
	TimeZone              string `json:"time_zone"`
	CountryCode           string `json:"country_code"`
}

type OrganizationData struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Attributes OrganizationAttributes `json:"attributes"`
}

type OrganizationResponse struct {
	Data  OrganizationData `json:"data"`
	Links Links            `json:"links"`
	Meta  Meta             `json:"meta"`
}

// GetOrganization returns the organization the current access token (see
// WithAccessToken) - or, for a script using the PAT instead, the PAT's own
// account - is scoped to. There's no id parameter: PCO's People API is
// entirely single-tenant per credential, so there's exactly one to fetch.
func GetOrganization(ctx context.Context) (response OrganizationResponse, err error) {
	url := fmt.Sprintf("%s/%s", baseURL, organizationPath)

	response, err = NewRequest[OrganizationResponse](ctx, "GET", url, nil)

	return
}
