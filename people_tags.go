package pco

import (
	"context"
	"fmt"
)

// personTagsPath is scoped under services/v2/people, not the People app's
// own people/v2/people (peoplePath in people.go) - Services exposes its own
// "people" sub-resource keyed by the same organization-wide person id, and
// tags (like blockouts, team_members, and person_team_position_assignments)
// only exist on that side of the API for this purpose. Confirmed live
// (2026-09-20) against a real Planning Center dev org.
func personTagsPath(personID string) string {
	return fmt.Sprintf("services/v2/people/%s/tags", personID)
}

// GetPersonTags lists personID's currently-assigned tags across all of its
// tag groups (TagGroupAttributes.TagsFor == "person") - the Person
// equivalent of GetSongTags (tags.go), reusing the same TagListResponse
// shape since PCO's Tag resource doesn't vary by taggable type. Confirmed
// live (2026-09-20) against a real Planning Center dev org.
//
// No AssignPersonTags - unlike AssignSongTags (tags.go), a Person-side
// assign_tags call hasn't been probed live yet, so this SDK doesn't guess
// its shape. Add it only after confirming the real request/response
// against a live account, the same way AssignSongTags' own shape turned
// out not to match the "obvious" JSON:API convention.
func GetPersonTags(ctx context.Context, personID string) (response TagListResponse, err error) {
	url := fmt.Sprintf("%s/%s", baseURL, personTagsPath(personID))

	response, err = NewRequest[TagListResponse](ctx, "GET", url, nil)

	return
}
