package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetItemAssignments(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/items/i-1/item_assignments"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"ItemAssignment","id":"1","attributes":{"assignable_type":"Person"},"relationships":{"assignable":{"data":{"type":"Person","id":"p-1"}}}}]}`)
	})

	response, err := GetItemAssignments(context.Background(), "st-1", "p-1", "i-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.AssignableType != "Person" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
	if response.Data[0].Relationships.Assignable.Data == nil || response.Data[0].Relationships.Assignable.Data.ID != "p-1" {
		t.Errorf("unexpected assignable relationship: %+v", response.Data[0].Relationships.Assignable)
	}
}

func TestCreateItemAssignment(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/items/i-1/item_assignments"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		attrs := attributes(t, decodeBody(t, r))
		if attrs["assignable_type"] != "Person" || attrs["assignable_id"] != "person-1" {
			t.Errorf("unexpected attributes: %+v", attrs)
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"ItemAssignment","id":"assignment-1","attributes":{"assignable_type":"Person"},"relationships":{"assignable":{"data":{"type":"Person","id":"person-1"}}}}}`)
	})

	response, err := CreateItemAssignment(context.Background(), "st-1", "p-1", "i-1", &CreateItemAssignmentParams{
		PersonID: "person-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "assignment-1" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestCreateItemAssignmentDefaultsAssignableType(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		attrs := attributes(t, decodeBody(t, r))
		if attrs["assignable_type"] != ItemAssignmentAssignableTypePerson {
			t.Errorf("expected assignable_type to default to %q, got %+v", ItemAssignmentAssignableTypePerson, attrs)
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"ItemAssignment","id":"assignment-1"}}`)
	})

	if _, err := CreateItemAssignment(context.Background(), "st-1", "p-1", "i-1", &CreateItemAssignmentParams{PersonID: "person-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateItemAssignmentNilParams(t *testing.T) {
	if _, err := CreateItemAssignment(context.Background(), "st-1", "p-1", "i-1", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestDeleteItemAssignment(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/st-1/plans/p-1/items/i-1/item_assignments/assignment-1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	if err := DeleteItemAssignment(context.Background(), "st-1", "p-1", "i-1", "assignment-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
