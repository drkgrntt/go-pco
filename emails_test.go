package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestCreateEmail(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + peoplePath + "/123/emails"; r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}

		attrs := attributes(t, decodeBody(t, r))
		if attrs["address"] != "ada@example.com" {
			t.Errorf("expected address ada@example.com, got %v", attrs["address"])
		}

		writeJSON(t, w, http.StatusCreated, `{"data":{"type":"Email","id":"1","attributes":{"address":"ada@example.com","location":"Home"}}}`)
	})

	response, err := CreateEmail(context.Background(), "123", &EmailCreateParams{Address: "ada@example.com", Location: "Home"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Data.Attributes.Location != "Home" {
		t.Errorf("expected location Home, got %q", response.Data.Attributes.Location)
	}
}

func TestCreateEmailNilParams(t *testing.T) {
	if _, err := CreateEmail(context.Background(), "123", nil); err == nil {
		t.Fatal("expected an error for nil params")
	}
}
