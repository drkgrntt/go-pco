package pco

import (
	"context"
	"fmt"
)

func tagsPath(tagGroupID string) string {
	return fmt.Sprintf("%s/%s/tags", tagGroupsPath, tagGroupID)
}

func songTagsPath(songID string) string {
	return fmt.Sprintf("%s/%s/tags", songsPath, songID)
}

// TagRelationships covers a Tag's one documented relationship, its parent
// TagGroup - a to-one bare resource identifier (type+id only), confirmed
// live (2026-09-18) against GET /services/v2/songs/{songID}/tags.
type TagRelationships struct {
	TagGroup HasOneRelationship `json:"tag_group"`
}

type TagAttributes struct {
	Name string `json:"name"`
}

type TagData struct {
	Type          string           `json:"type"`
	ID            string           `json:"id"`
	Attributes    TagAttributes    `json:"attributes"`
	Relationships TagRelationships `json:"relationships"`
}

type TagResponse struct {
	Data     TagData `json:"data"`
	Included []any   `json:"included"`
	Links    Links   `json:"links"`
	Meta     Meta    `json:"meta"`
}

type TagListResponse struct {
	Data     []TagData `json:"data"`
	Included []any     `json:"included"`
	Links    Links     `json:"links"`
	Meta     Meta      `json:"meta"`
}

type TagsParams struct {
	PerPage int
	Offset  int
}

// GetTags lists tagGroupID's own tags directly - an alternative to
// sideloading them via GetTagGroups' Include: []string{"tags"}.
func GetTags(ctx context.Context, tagGroupID string, params *TagsParams) (response TagListResponse, err error) {
	if params == nil {
		params = &TagsParams{}
	}

	q := NewQueryParams().PerPage(params.PerPage).Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, tagsPath(tagGroupID), q.Encode())

	response, err = NewRequest[TagListResponse](ctx, "GET", url, nil)

	return
}

// GetSongTags lists songID's currently-assigned tags across all of its tag
// groups - confirmed live (2026-09-18) against GET
// /services/v2/songs/{songID}/tags.
func GetSongTags(ctx context.Context, songID string) (response TagListResponse, err error) {
	url := fmt.Sprintf("%s/%s", baseURL, songTagsPath(songID))

	response, err = NewRequest[TagListResponse](ctx, "GET", url, nil)

	return
}

// AssignSongTags sets songID's tags for a single tag group via POST
// /services/v2/songs/{songID}/assign_tags.
//
// **Confirmed live (2026-09-18) - this is the one place this SDK differs
// from the "obvious" JSON:API shape.** Two more natural-looking bodies were
// tried first and both failed: a flat/attribute-style body
// (`{"data":{"attributes":{"tag_ids":[...]}}}`, with or without
// `"type":"TagGroup"`) got a 400 ("Can't assign nil tags, please pass
// tag_ids you want to assign"); a bare array body (`{"data":[...]}`) got a
// 422 ("Resource object must be an object"). The only shape that actually
// worked is a `relationships.tags.data` array on a single `TagGroup`-typed
// data object:
//
//	{
//	  "data": {
//	    "type": "TagGroup",
//	    "relationships": {
//	      "tags": {"data": [{"type": "Tag", "id": "233148"}, {"type": "Tag", "id": "233151"}]}
//	    }
//	  }
//	}
//
// PCO returns 204 No Content on success, so this returns just an error, not
// a response envelope.
//
// **Critical: this is a full replace of that tag group's assignment on the
// song, not an additive/incremental change.** Confirmed live: calling this
// a second time with a different, non-overlapping set of tag IDs (all from
// the same tag group as the first call) *removes* the tags the first call
// set rather than adding to them. An empty tagIDs slice clears that group's
// assignment on the song entirely (still 204, not rejected). This function
// passes tagIDs through as-is - the caller is responsible for always
// supplying the *complete* desired tag-id set for whichever group is being
// changed; never call this with just the one tag you want to add.
//
// Not yet confirmed live: what happens when tagIDs mixes tags belonging to
// two different tag groups in one call - no second real song-scoped tag
// group was available to test this against, so don't assume either a clean
// per-group split or a rejection.
func AssignSongTags(ctx context.Context, songID string, tagIDs []string) (err error) {
	url := fmt.Sprintf("%s/%s/%s/assign_tags", baseURL, songsPath, songID)

	tagRefs := make([]map[string]any, len(tagIDs))
	for i, id := range tagIDs {
		tagRefs[i] = map[string]any{"type": "Tag", "id": id}
	}

	body := RequestBody{
		Data: RequestData{
			Type: "TagGroup",
			Relationships: map[string]any{
				"tags": map[string]any{
					"data": tagRefs,
				},
			},
		},
	}

	_, err = NewRequest[struct{}](ctx, "POST", url, body)

	return
}
