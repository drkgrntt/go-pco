package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetTeamMembers(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + teamMembersPath("1", "2"); r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{
			"type": "PlanPerson",
			"id": "1",
			"attributes": {"name": "Derek Garnett", "status": "U", "team_position_name": "Drums"},
			"relationships": {"person": {"data": {"type": "Person", "id": "5"}}}
		}]}`)
	})

	response, err := GetTeamMembers(context.Background(), "1", "2", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.Name != "Derek Garnett" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestCreateTeamMember(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if want := "/" + teamMembersPath("1", "2"); r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		body := decodeBody(t, r)
		attrs := attributes(t, body)
		if attrs["team_position_name"] != "Drums" {
			t.Errorf("expected team_position_name Drums, got %v", attrs["team_position_name"])
		}
		if attrs["status"] != PlanPersonStatusUnconfirmed {
			t.Errorf("expected default status %q, got %v", PlanPersonStatusUnconfirmed, attrs["status"])
		}

		rels := relationships(t, body)
		team, _ := rels["team"].(map[string]any)["data"].(map[string]any)
		if team["id"] != "7" {
			t.Errorf("expected team id 7, got %v", team["id"])
		}
		person, _ := rels["person"].(map[string]any)["data"].(map[string]any)
		if person["id"] != "5" {
			t.Errorf("expected person id 5, got %v", person["id"])
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"PlanPerson","id":"1","attributes":{"team_position_name":"Drums"}}}`)
	})

	response, err := CreateTeamMember(context.Background(), "1", "2", &CreateTeamMemberParams{
		PersonID:         "5",
		TeamID:           "7",
		TeamPositionName: "Drums",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "1" {
		t.Errorf("expected id 1, got %q", response.Data.ID)
	}
}

func TestCreateTeamMemberExplicitStatus(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attrs := attributes(t, decodeBody(t, r))
		if attrs["status"] != PlanPersonStatusConfirmed {
			t.Errorf("expected explicit status %q, got %v", PlanPersonStatusConfirmed, attrs["status"])
		}
		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"PlanPerson","id":"1"}}`)
	})

	if _, err := CreateTeamMember(context.Background(), "1", "2", &CreateTeamMemberParams{
		PersonID: "5", TeamID: "7", Status: PlanPersonStatusConfirmed,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateTeamMemberNilParams(t *testing.T) {
	if _, err := CreateTeamMember(context.Background(), "1", "2", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

// TestDeleteTeamMemberUsesPlanScopedPath confirms DeleteTeamMember addresses
// the same plan-scoped path a member was created on, not PCO's documented
// person-scoped path - see teamMembersPath's doc comment for why (a
// person-scoped delete 404s for at least one real, older plan on a live
// account, while the plan-scoped path deletes reliably regardless of the
// plan's age).
func TestCreateTeamMemberWithNotes(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attrs := attributes(t, decodeBody(t, r))
		if attrs["notes"] != "needs the tenor mic" {
			t.Errorf("expected notes %q, got %v", "needs the tenor mic", attrs["notes"])
		}
		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"PlanPerson","id":"1"}}`)
	})

	_, err := CreateTeamMember(context.Background(), "1", "2", &CreateTeamMemberParams{
		PersonID: "5",
		TeamID:   "7",
		Notes:    "needs the tenor mic",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateTeamMember(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if want := "/" + teamMembersPath("1", "2") + "/9"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		attrs := attributes(t, decodeBody(t, r))
		if attrs["notes"] != "swapped to guitar" {
			t.Errorf("expected notes %q, got %v", "swapped to guitar", attrs["notes"])
		}
		// Only Notes was set on params - Status/DeclineReason/
		// TeamPositionName should be omitted entirely, not sent as "".
		if _, ok := attrs["status"]; ok {
			t.Errorf("expected status to be omitted, got %v", attrs["status"])
		}

		writeJSON(t, w, http.StatusOK, `{"data":{"type":"PlanPerson","id":"9","attributes":{"notes":"swapped to guitar"}}}`)
	})

	response, err := UpdateTeamMember(context.Background(), "1", "2", "9", &UpdateTeamMemberParams{
		Notes: "swapped to guitar",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.Notes != "swapped to guitar" {
		t.Errorf("expected notes %q, got %q", "swapped to guitar", response.Data.Attributes.Notes)
	}
}

func TestUpdateTeamMemberNilParams(t *testing.T) {
	if _, err := UpdateTeamMember(context.Background(), "1", "2", "9", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestDeleteTeamMemberUsesPlanScopedPath(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if want := "/" + teamMembersPath("1", "2") + "/9"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := DeleteTeamMember(context.Background(), "1", "2", "9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPersonPlanPeople(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + planPeoplePath("5"); r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("where[team_id]") != "7" {
			t.Errorf("expected where[team_id]=7, got %q", q.Get("where[team_id]"))
		}
		if q.Get("include") != "plan" {
			t.Errorf("expected include=plan, got %q", q.Get("include"))
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{
			"type": "PlanPerson",
			"id": "1",
			"attributes": {"team_position_name": "Drums"},
			"relationships": {"plan": {"data": {"type": "Plan", "id": "2"}}}
		}],"included":[{"type":"Plan","id":"2","attributes":{"sort_date":"2026-09-20T00:00:00Z"}}]}`)
	})

	response, err := GetPersonPlanPeople(context.Background(), "5", &PersonPlanPeopleParams{TeamID: "7", Include: []string{"plan"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Relationships.Plan.Data.ID != "2" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
	if len(response.Included) != 1 {
		t.Errorf("expected 1 included resource, got %d", len(response.Included))
	}
}
