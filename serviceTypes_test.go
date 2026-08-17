package pco

import (
	"net/http"
	"testing"
)

func TestGetServiceTypes(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{"type":"ServiceType","id":"1","attributes":{"name":"Sunday Service"}}]}`)
	})

	response, err := GetServiceTypes(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Attributes.Name != "Sunday Service" {
		t.Errorf("unexpected response: %+v", response.Data)
	}
}

func TestGetServiceType(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + serviceTypesPath + "/1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"ServiceType","id":"1","attributes":{"name":"Sunday Service"}}}`)
	})

	response, err := GetServiceType("1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "1" {
		t.Errorf("expected id 1, got %q", response.Data.ID)
	}
}

func TestCreateServiceType(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		attrs := attributes(t, decodeBody(t, r))
		if attrs["name"] != "Youth Service" {
			t.Errorf("expected name Youth Service, got %v", attrs["name"])
		}
		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"ServiceType","id":"2","attributes":{"name":"Youth Service"}}}`)
	})

	response, err := CreateServiceType(&CreateServiceTypeParams{Name: "Youth Service"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.ID != "2" {
		t.Errorf("expected id 2, got %q", response.Data.ID)
	}
}

func TestCreateServiceTypeNilParams(t *testing.T) {
	if _, err := CreateServiceType(nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestUpdateServiceType(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if want := "/" + serviceTypesPath + "/1"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":{"type":"ServiceType","id":"1","attributes":{"name":"Renamed"}}}`)
	})

	response, err := UpdateServiceType("1", &UpdateServiceTypeParams{Name: "Renamed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.Name != "Renamed" {
		t.Errorf("expected name Renamed, got %q", response.Data.Attributes.Name)
	}
}

func TestDeleteServiceType(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if err := DeleteServiceType("1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
