package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestCreateAddress(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if want := "/" + peoplePath + "/123/addresses"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		attrs := attributes(t, decodeBody(t, r))
		if attrs["street_line_1"] != "123 Main St" {
			t.Errorf("expected street_line_1 123 Main St, got %v", attrs["street_line_1"])
		}
		if attrs["primary"] != true {
			t.Errorf("expected primary true, got %v (%T)", attrs["primary"], attrs["primary"])
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Address","id":"1","attributes":{"street_line_1":"123 Main St","primary":true}}}`)
	})

	response, err := CreateAddress(context.Background(), "123", &AddressCreateParams{AddressLine1: "123 Main St", Primary: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !response.Data.Attributes.Primary {
		t.Errorf("expected primary true in response, got %+v", response.Data.Attributes)
	}
}

func TestCreateAddressNilParams(t *testing.T) {
	if _, err := CreateAddress(context.Background(), "123", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}

func TestCreateAddressErrorStatus(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusUnprocessableEntity, `{"errors":[{"detail":"zip is invalid"}]}`)
	})

	_, err := CreateAddress(context.Background(), "123", &AddressCreateParams{Zip: "not-a-zip"})
	if err == nil {
		t.Fatal("expected an error for a 422 response")
	}
	if reqErr, ok := err.(*RequestError); !ok || reqErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected a 422 *RequestError, got %v", err)
	}
}
