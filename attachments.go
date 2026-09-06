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
// Field names confirmed live (2026-09-06) by creating and deleting a real
// throwaway attachment against a dev org song (upload via
// upload.planningcenteronline.com/v2/files, then POST .../songs/:id/attachments
// with the returned file_upload_identifier) and reading the response back -
// this package's normal "verify before documenting" convention (see
// arrangements.go, songs.go), delayed past the original build because that
// session's sandbox had no reachable PCO credentials.
//
// The real Attachment resource has more attributes than this struct wraps
// (allow_mp3_download, attachable_type, deleted_at, display_name,
// downloadable, filetype, has_preview, linked_url, page_order, pco_type,
// remote_link, streamable, thumbnail_url, transposable, web_streamable,
// among others) - only the subset pco-assistant actually uses is wrapped
// here, matching this package's usual "wrap what's used" convention, not an
// oversight.
//
// **URL is a Planning Center web page, not the file itself** - confirmed
// live as `https://services.planningcenteronline.com/attachments/:id`, the
// same page a person sees clicking the attachment inside PCO's own UI, not
// a direct/downloadable file link. Getting the actual file bytes requires a
// separate action this package does NOT wrap: `POST
// .../songs/:song_id/attachments/:id/open` (a GET on that path returns only
// its own documentation, confirmed live), which returns an
// `AttachmentActivity` resource whose `attachment_url` attribute holds the
// real file link - not confirmed here whether that link is long-lived or a
// short-lived pre-signed one, since exercising it further tripped this
// session's own permission guardrails. pco-assistant's `OpenSongAttachment`
// redirects to this struct's URL (the PCO web page) rather than the
// unwrapped /open flow - a reasonable v1 (PCO's own page handles auth/
// viewing/downloading for an already-PCO-authenticated user), not a bug,
// but the two are materially different destinations and any future work
// wanting an actual file download/stream needs to wrap the /open action
// and AttachmentActivity, not just read URL harder.
type AttachmentAttributes struct {
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
	Filename    string    `json:"filename"`
	FileSize    int       `json:"file_size"`
	UpdatedAt   time.Time `json:"updated_at"`
	// URL is PCO's own attachment web page, not a raw file link - see this
	// struct's own doc comment above.
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
