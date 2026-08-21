package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetOrganization(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + organizationPath; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{
			"type": "Organization",
			"id": "539527",
			"attributes": {
				"name": "Derek's Developer Church",
				"church_center_subdomain": "dereks-developer-church-539527",
				"time_zone": "America/Chicago",
				"country_code": "US"
			}
		}}`)
	})

	response, err := GetOrganization(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "539527" {
		t.Errorf("expected id 539527, got %q", response.Data.ID)
	}
	if response.Data.Attributes.Name != "Derek's Developer Church" {
		t.Errorf("expected name %q, got %q", "Derek's Developer Church", response.Data.Attributes.Name)
	}
	if response.Data.Attributes.TimeZone != "America/Chicago" {
		t.Errorf("expected time zone America/Chicago, got %q", response.Data.Attributes.TimeZone)
	}
}
