package pco

import (
	"context"
	"fmt"
)

const tagGroupsPath = "services/v2/tag_groups"

// TagGroupRelationships covers the one relationship PCO documents for a Tag
// Group: its own Tags, as a to-many relationship of bare resource
// identifiers (type+id only, no attributes) - the same JSON:API shape
// HasManyRelationship already models for e.g. a Person's Emails. Sideload
// the full Tag attributes with TagGroupsParams.Include (?include=tags), or
// fetch them directly with GetTags.
type TagGroupRelationships struct {
	Tags HasManyRelationship `json:"tags"`
}

// TagGroupAttributes - field names and shapes confirmed live (2026-09-18)
// against a real Planning Center dev org, GET /services/v2/tag_groups.
// ServiceTypeFolderName is a pointer (not a plain string) since it was
// observed null in practice and PCO's docs don't say null and "" mean the
// same thing here. TagsFor is a plain string, not a Go enum/const set -
// "song"/"arrangement"/"person"/"media" have all been observed, but PCO may
// support taggable types this SDK hasn't seen yet.
type TagGroupAttributes struct {
	AllowMultipleSelections bool    `json:"allow_multiple_selections"`
	Name                    string  `json:"name"`
	Required                bool    `json:"required"`
	ServiceTypeFolderName   *string `json:"service_type_folder_name"`
	TagsFor                 string  `json:"tags_for"`
}

type TagGroupData struct {
	Type          string                `json:"type"`
	ID            string                `json:"id"`
	Attributes    TagGroupAttributes    `json:"attributes"`
	Relationships TagGroupRelationships `json:"relationships"`
}

type TagGroupResponse struct {
	Data     TagGroupData `json:"data"`
	Included []any        `json:"included"`
	Links    Links        `json:"links"`
	Meta     Meta         `json:"meta"`
}

type TagGroupListResponse struct {
	Data     []TagGroupData `json:"data"`
	Included []any          `json:"included"`
	Links    Links          `json:"links"`
	Meta     Meta           `json:"meta"`
}

// TagGroupsParams. Include supports PCO's documented sideload of each
// group's own Tags (Include: []string{"tags"}) - confirmed live - following
// the same Include []string idiom as PeopleParams/ArrangementsParams rather
// than a fixed helper, since this SDK's existing multi-value Include params
// are all plain string slices.
type TagGroupsParams struct {
	Include []string
	PerPage int
	Offset  int
}

// GetTagGroups lists every tag group configured for the org (e.g. "Type",
// "Service", "Team") - top-level, not nested under anything.
func GetTagGroups(ctx context.Context, params *TagGroupsParams) (response TagGroupListResponse, err error) {
	if params == nil {
		params = &TagGroupsParams{}
	}

	q := NewQueryParams().
		Include(params.Include...).
		PerPage(params.PerPage).
		Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, tagGroupsPath, q.Encode())

	response, err = NewRequest[TagGroupListResponse](ctx, "GET", url, nil)

	return
}

func GetTagGroup(ctx context.Context, id string) (response TagGroupResponse, err error) {
	url := fmt.Sprintf("%s/%s/%s", baseURL, tagGroupsPath, id)

	response, err = NewRequest[TagGroupResponse](ctx, "GET", url, nil)

	return
}

// CreateTagGroup/UpdateTagGroup/DeleteTagGroup are deliberately NOT
// implemented. A live probe (2026-09-18) against a real Planning Center dev
// org attempted POST /services/v2/tag_groups with a plausible JSON:API body
// (`{"data":{"type":"TagGroup","attributes":{"name":"..."}}}`) from a normal
// signed-in session and got back 403 Forbidden ("cannot create a
// TagGroup") - meaning both the actual required request body shape AND the
// PCO permission tier needed to create one are unconfirmed. Don't guess the
// body shape from JSON:API convention (this package's own AssignSongTags,
// in tags.go, already turned out not to follow the "obvious" shape for a
// different tag endpoint) - re-verify live from a higher-permission PCO
// session before adding writes here.
