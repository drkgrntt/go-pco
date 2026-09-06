package pco

import (
	"context"
	"fmt"
	"time"
)

func attachmentsPath(songID string) string {
	return fmt.Sprintf("%s/%s/attachments", songsPath, songID)
}

// AttachmentAttributes covers a Song Attachment - an arbitrary file a church
// uploads to a song in Planning Center (a lead sheet/chord chart PDF from
// SongSelect/PraiseCharts, a scan, an audio file, ...), labeled with an
// org-defined "attachment type" that isn't a fixed enum PCO exposes here
// (see pco-assistant's song-lead-sheet-attachments-plan.md for why this app
// deliberately doesn't try to guess which attachment is "the" lead sheet).
// This is a completely different resource from Arrangement's own
// ChordChart/Lyrics text fields - those are chord symbols/lyrics text with
// no melodic notation; an Attachment is an arbitrary file blob, structured
// notation or not.
//
// Field names here are inferred from PCO's ordinary JSON:API attribute
// naming convention (matching every other resource in this package,
// including Arrangement's own filename-adjacent fields) rather than
// confirmed against a real live response - this package's own doc comments
// elsewhere in the codebase (see arrangements.go, songs.go) are normally
// only written after a live check, but that step could not be completed
// this session (sandboxed build environment with no reachable PCO
// credentials or dev database). In particular, whether URL is a long-lived
// link or an expiring pre-signed one is NOT verified - pco-assistant's own
// integration deliberately treats it as expiring (never caches it, always
// live-fetches on click) specifically because that assumption couldn't be
// confirmed either way. Treat every field name here as provisional until
// checked against a real GetAttachments response for a song with a real
// uploaded file.
type AttachmentAttributes struct {
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
	Filename    string    `json:"filename"`
	FileSize    int       `json:"file_size"`
	UpdatedAt   time.Time `json:"updated_at"`
	// URL downloads/displays the actual file. See this struct's own doc
	// comment above - not confirmed live whether this is long-lived or an
	// expiring pre-signed link.
	URL string `json:"url"`
}

type AttachmentData struct {
	Type       string               `json:"type"`
	ID         string               `json:"id"`
	Attributes AttachmentAttributes `json:"attributes"`
}

type AttachmentListResponse struct {
	Data     []AttachmentData `json:"data"`
	Included []any            `json:"included"`
	Links    Links            `json:"links"`
	Meta     Meta             `json:"meta"`
}

// AttachmentsParams mirrors ArrangementsParams' shape, minus Include - a
// song's attachments have no documented includable sub-resources this
// package needs yet.
type AttachmentsParams struct {
	// OrderBy sorts by a can_order_by field, e.g. "created_at"/"updated_at".
	// Prefix with "-" for descending.
	OrderBy string
	PerPage int
	Offset  int
}

// GetAttachments lists songID's attachments - read-only, list-only: this
// package deliberately doesn't wrap create/update/delete for this resource,
// since pco-assistant only ever surfaces attachments as reference links, it
// never manages them.
func GetAttachments(ctx context.Context, songID string, params *AttachmentsParams) (response AttachmentListResponse, err error) {
	if params == nil {
		params = &AttachmentsParams{}
	}

	q := NewQueryParams().
		OrderBy(params.OrderBy).
		PerPage(params.PerPage).
		Offset(params.Offset)

	url := fmt.Sprintf("%s/%s%s", baseURL, attachmentsPath(songID), q.Encode())

	response, err = NewRequest[AttachmentListResponse](ctx, "GET", url, nil)

	return
}
