package pco

import (
	"net/http"
	"testing"
)

func TestGetPersonTeamPositionAssignments(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + personTeamPositionAssignmentsPath("1", "2"); r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{
			"type": "PersonTeamPositionAssignment",
			"id": "1",
			"attributes": {"schedule_preference": "Every other week"},
			"relationships": {"person": {"data": {"type": "Person", "id": "5"}}}
		}]}`)
	})

	response, err := GetPersonTeamPositionAssignments("1", "2", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.SchedulePreference != "Every other week" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestCreatePersonTeamPositionAssignment(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if want := "/" + personTeamPositionAssignmentsPath("1", "2"); r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		body := decodeBody(t, r)
		attrs := attributes(t, body)
		if attrs["schedule_preference"] != SchedulePreferenceEveryOtherWeek {
			t.Errorf("expected schedule_preference %q, got %v", SchedulePreferenceEveryOtherWeek, attrs["schedule_preference"])
		}

		rels := relationships(t, body)
		person, _ := rels["person"].(map[string]any)["data"].(map[string]any)
		if person["id"] != "5" {
			t.Errorf("expected person id 5, got %v", person["id"])
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"PersonTeamPositionAssignment","id":"1","attributes":{"schedule_preference":"Every other week"}}}`)
	})

	response, err := CreatePersonTeamPositionAssignment("1", "2", &CreatePersonTeamPositionAssignmentParams{
		PersonID:           "5",
		SchedulePreference: SchedulePreferenceEveryOtherWeek,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "1" {
		t.Errorf("expected id 1, got %q", response.Data.ID)
	}
}

func TestCreatePersonTeamPositionAssignmentNilParams(t *testing.T) {
	if _, err := CreatePersonTeamPositionAssignment("1", "2", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestUpdatePersonTeamPositionAssignment(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if want := "/" + personTeamPositionAssignmentsPath("1", "2") + "/3"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		attrs := attributes(t, decodeBody(t, r))
		if attrs["schedule_preference"] != SchedulePreferenceOnceAMonth {
			t.Errorf("expected schedule_preference %q, got %v", SchedulePreferenceOnceAMonth, attrs["schedule_preference"])
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"PersonTeamPositionAssignment","id":"3","attributes":{"schedule_preference":"Once a month"}}}`)
	})

	response, err := UpdatePersonTeamPositionAssignment("1", "2", "3", &UpdatePersonTeamPositionAssignmentParams{
		SchedulePreference: SchedulePreferenceOnceAMonth,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.SchedulePreference != SchedulePreferenceOnceAMonth {
		t.Errorf("unexpected schedule_preference: %q", response.Data.Attributes.SchedulePreference)
	}
}

func TestDeletePersonTeamPositionAssignment(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if want := "/" + personTeamPositionAssignmentsPath("1", "2") + "/3"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := DeletePersonTeamPositionAssignment("1", "2", "3"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
