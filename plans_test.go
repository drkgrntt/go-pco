package pco

import (
	"net/http"
	"testing"
)

func TestGetPlans(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"Plan","id":"1","attributes":{"title":"This Sunday"}}]}`)
	})

	response, err := GetPlans("st-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.Title != "This Sunday" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestGetPlansBuildsOrderByAndFilterParams(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("order"); got != "-sort_date" {
			t.Errorf("expected order=-sort_date, got %q", got)
		}
		if got := q.Get("filter"); got != "past" {
			t.Errorf("expected filter=past, got %q", got)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[]}`)
	})

	if _, err := GetPlans("st-1", &PlansParams{OrderBy: "-sort_date", Filter: "past"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPlan(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Plan","id":"p-1","attributes":{"title":"This Sunday"}}}`)
	})

	response, err := GetPlan("st-1", "p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "p-1" {
		t.Errorf("expected id p-1, got %q", response.Data.ID)
	}
}

func TestCreatePlan(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attrs := attributes(t, decodeBody(t, r))
		if attrs["title"] != "This Sunday" {
			t.Errorf("expected title This Sunday, got %v", attrs["title"])
		}
		if attrs["public"] != true {
			t.Errorf("expected public true, got %v", attrs["public"])
		}
		if _, present := attrs["series_id"]; present {
			t.Errorf("expected series_id to be omitted when unset, got %v", attrs["series_id"])
		}
		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Plan","id":"p-1","attributes":{"title":"This Sunday","public":true}}}`)
	})

	response, err := CreatePlan("st-1", &CreatePlanParams{Title: "This Sunday", Public: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "p-1" {
		t.Errorf("expected id p-1, got %q", response.Data.ID)
	}
}

func TestCreatePlanIncludesSeriesWhenSet(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attrs := attributes(t, decodeBody(t, r))
		if attrs["series_id"] != "s-1" {
			t.Errorf("expected series_id s-1, got %v", attrs["series_id"])
		}
		if attrs["series_title"] != "Advent" {
			t.Errorf("expected series_title Advent, got %v", attrs["series_title"])
		}
		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Plan","id":"p-1","attributes":{}}}`)
	})

	if _, err := CreatePlan("st-1", &CreatePlanParams{SeriesID: "s-1", SeriesTitle: "Advent"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreatePlanNilParams(t *testing.T) {
	if _, err := CreatePlan("st-1", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestUpdatePlan(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attrs := attributes(t, decodeBody(t, r))
		if len(attrs) != 1 || attrs["reminders_disabled"] != true {
			t.Errorf("expected only reminders_disabled=true in body, got %+v", attrs)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"Plan","id":"p-1","attributes":{"reminders_disabled":true}}}`)
	})

	remindersDisabled := true
	response, err := UpdatePlan("st-1", "p-1", &UpdatePlanParams{RemindersDisabled: &remindersDisabled})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !response.Data.Attributes.RemindersDisabled {
		t.Errorf("expected reminders_disabled true, got false")
	}
}

func TestDeletePlan(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := DeletePlan("st-1", "p-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
