package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetPeopleBuildsWhereFilters(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/"+peoplePath {
			t.Errorf("expected path /%s, got %s", peoplePath, got)
		}

		q := r.URL.Query()
		if q.Get("where[first_name]") != "Ada" {
			t.Errorf("expected where[first_name]=Ada, got %q", q.Get("where[first_name]"))
		}
		if q.Get("where[last_name]") != "Lovelace" {
			t.Errorf("expected where[last_name]=Lovelace, got %q", q.Get("where[last_name]"))
		}
		if q.Get("where[search_name_or_email]") != "ada@example.com" {
			t.Errorf("expected where[search_name_or_email]=ada@example.com, got %q", q.Get("where[search_name_or_email]"))
		}
		if q.Get("per_page") != "10" {
			t.Errorf("expected per_page=10, got %q", q.Get("per_page"))
		}

		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"Person","id":"1","attributes":{"first_name":"Ada","last_name":"Lovelace"}}]}`)
	})

	response, err := GetPeople(context.Background(), &PeopleParams{
		FirstName: "Ada",
		LastName:  "Lovelace",
		Email:     "ada@example.com",
		PerPage:   10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.FirstName != "Ada" {
		t.Errorf("unexpected response data: %+v", response.Data)
	}
}

func TestGetPeopleBuildsIDsFilter(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("where[id]"); got != "1,2,3" {
			t.Errorf("expected where[id]=1,2,3, got %q", got)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[
			{"type":"Person","id":"1","attributes":{"first_name":"Ada"}},
			{"type":"Person","id":"2","attributes":{"first_name":"Grace"}},
			{"type":"Person","id":"3","attributes":{"first_name":"Katherine"}}
		]}`)
	})

	response, err := GetPeople(context.Background(), &PeopleParams{IDs: []string{"1", "2", "3"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 3 {
		t.Errorf("expected 3 people, got %d", len(response.Data))
	}
}

func TestGetPeopleBuildsIncludeParam(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("include"); got != "emails,addresses" {
			t.Errorf("expected include=emails,addresses, got %q", got)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetPeople(context.Background(), &PeopleParams{Include: []string{"emails", "addresses"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPeopleBuildsOrderByParam(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("order"); got != "-created_at" {
			t.Errorf("expected order=-created_at, got %q", got)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetPeople(context.Background(), &PeopleParams{OrderBy: "-created_at"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPeopleDecodesRelationshipsAndIncluded(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, `{
			"data": [{
				"type": "Person",
				"id": "1",
				"attributes": {"first_name": "Ada", "last_name": "Lovelace"},
				"relationships": {
					"primary_campus": {"data": null},
					"organization": {"data": {"type": "Organization", "id": "539527"}},
					"emails": {"data": [{"type": "Email", "id": "10"}]},
					"addresses": {"data": [{"type": "Address", "id": "20"}, {"type": "Address", "id": "21"}]},
					"phone_numbers": {"data": []}
				}
			}],
			"included": [
				{"type": "Email", "id": "10", "attributes": {"address": "ada@example.com"}}
			]
		}`)
	})

	response, err := GetPeople(context.Background(), &PeopleParams{Include: []string{"emails", "addresses", "phone_numbers", "organization"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rel := response.Data[0].Relationships
	if rel.PrimaryCampus.Data != nil {
		t.Errorf("expected nil PrimaryCampus.Data, got %+v", rel.PrimaryCampus.Data)
	}
	if rel.Organization.Data == nil || rel.Organization.Data.ID != "539527" {
		t.Errorf("expected Organization.Data.ID 539527, got %+v", rel.Organization.Data)
	}
	if len(rel.Emails.Data) != 1 || rel.Emails.Data[0].ID != "10" {
		t.Errorf("expected one Email relationship with id 10, got %+v", rel.Emails.Data)
	}
	if len(rel.Addresses.Data) != 2 {
		t.Errorf("expected two Address relationships, got %+v", rel.Addresses.Data)
	}
	if len(rel.PhoneNumbers.Data) != 0 {
		t.Errorf("expected zero PhoneNumber relationships, got %+v", rel.PhoneNumbers.Data)
	}

	if len(response.Included) != 1 {
		t.Fatalf("expected one included resource, got %d", len(response.Included))
	}
}

func TestGetPeopleNilParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params for nil PeopleParams, got %q", r.URL.RawQuery)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetPeople(context.Background(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPerson(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + peoplePath + "/1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Person","id":"1","attributes":{"first_name":"Ada","last_name":"Lovelace"}}}`)
	})

	response, err := GetPerson(context.Background(), "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "1" || response.Data.Attributes.FirstName != "Ada" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestCreatePerson(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		attrs := attributes(t, decodeBody(t, r))
		if attrs["first_name"] != "Ada" || attrs["last_name"] != "Lovelace" {
			t.Errorf("unexpected attributes: %+v", attrs)
		}
		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Person","id":"42","attributes":{"first_name":"Ada","last_name":"Lovelace"}}}`)
	})

	response, err := CreatePerson(context.Background(), &CreatePersonParams{FirstName: "Ada", LastName: "Lovelace"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "42" {
		t.Errorf("expected id 42, got %q", response.Data.ID)
	}
}

func TestCreatePersonNilParams(t *testing.T) {
	if _, err := CreatePerson(context.Background(), nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}
