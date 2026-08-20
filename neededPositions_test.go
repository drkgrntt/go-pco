package pco

import (
	"context"
	"net/http"
	"testing"
)

func TestGetNeededPositions(t *testing.T) {
	startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if want := "/" + neededPositionsPath("1", "2"); r.URL.Path != want {
			t.Errorf("expected path %s, got %s", want, r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, `{"data":[{
			"type": "NeededPosition",
			"id": "1",
			"attributes": {"quantity": 2, "team_position_name": "Drums", "scheduled_to": "plan"},
			"relationships": {"team": {"data": {"type": "Team", "id": "7"}}, "plan": {"data": {"type": "Plan", "id": "2"}}}
		}]}`)
	})

	response, err := GetNeededPositions(context.Background(), "1", "2", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(response.Data) != 1 {
		t.Fatalf("expected 1 needed position, got %d", len(response.Data))
	}

	np := response.Data[0]
	if np.Attributes.Quantity != 2 || np.Attributes.TeamPositionName != "Drums" {
		t.Errorf("unexpected attributes: %+v", np.Attributes)
	}
	if np.Relationships.Team.Data == nil || np.Relationships.Team.Data.ID != "7" {
		t.Errorf("unexpected team relationship: %+v", np.Relationships.Team)
	}
}
