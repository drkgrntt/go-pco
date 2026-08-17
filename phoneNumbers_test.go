package pco

import (
	"net/http"
	"testing"
)

func TestCreatePhoneNumber(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + peoplePath + "/123/phone_numbers"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		attrs := attributes(t, decodeBody(t, r))
		if attrs["number"] != "555-1234" {
			t.Errorf("expected number 555-1234, got %v", attrs["number"])
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"PhoneNumber","id":"1","attributes":{"number":"555-1234","location":"Mobile"}}}`)
	})

	response, err := CreatePhoneNumber("123", &PhoneNumberCreateParams{Number: "555-1234", Location: "Mobile"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.Number != "555-1234" {
		t.Errorf("expected number 555-1234, got %q", response.Data.Attributes.Number)
	}
}

func TestCreatePhoneNumberNilParams(t *testing.T) {
	if _, err := CreatePhoneNumber("123", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}
