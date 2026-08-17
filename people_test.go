package pco

import (
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

	response, err := GetPeople(&PeopleParams{
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

func TestGetPeopleNilParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params for nil PeopleParams, got %q", r.URL.RawQuery)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetPeople(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
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

	response, err := CreatePerson(&CreatePersonParams{FirstName: "Ada", LastName: "Lovelace"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "42" {
		t.Errorf("expected id 42, got %q", response.Data.ID)
	}
}

func TestCreatePersonNilParams(t *testing.T) {
	if _, err := CreatePerson(nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}
