# Services

Wraps a starting subset of the [Services v2 API](https://api.planningcenteronline.com/docs/apps/services): the order-of-service side (Service Types, Plans, Items, Item Notes, Item Note Categories, Songs, Arrangements, Keys) and the people-scheduling side (Teams, Team Positions, Needed Positions, Team Members / PlanPerson, Person Team Position Assignments, Blockouts).

The Services API has ~65 resources total — the ones below plus Attachments, Schedules, and more remain unimplemented. See [Extending](../README.md#extending) in the root README for how to add another one; `https://api.planningcenteronline.com/docs/apps/services` lists every resource and its documentation path.

## Service Types

**[serviceTypes.go](../serviceTypes.go)**

A Service Type is a container for plans (e.g. "Sunday Service", "Youth Service").

| Function | Notes |
|---|---|
| `GetServiceTypes(ctx context.Context, params *ServiceTypesParams) (ServiceTypeListResponse, error)` | `params` may be `nil`. |
| `GetServiceType(ctx context.Context, id string) (ServiceTypeResponse, error)` | |
| `CreateServiceType(ctx context.Context, params *CreateServiceTypeParams) (ServiceTypeResponse, error)` | `{Name string}` |
| `UpdateServiceType(ctx context.Context, id string, params *UpdateServiceTypeParams) (ServiceTypeResponse, error)` | `{Name string}` |
| `DeleteServiceType(ctx context.Context, id string) error` | |

`CreateServiceTypeParams`/`UpdateServiceTypeParams` only expose `Name` — PCO's docs didn't enumerate the full set of creatable/updatable attributes for this resource the way they did for Plan and WebhookSubscription, so this is a conservative starting point. `ServiceTypeAttributes` (read-only beyond `Name`) also includes `Frequency`, `Sequence`, `ArchivedAt`, `AttachmentTypesEnabled`, permission strings, etc. — see [serviceTypes.go](../serviceTypes.go).

```go
st, err := pco.CreateServiceType(ctx, &pco.CreateServiceTypeParams{Name: "Youth Service"})
```

## Plans

**[plans.go](../plans.go)**

A Plan is a single service event within a Service Type (e.g. "This Sunday"). Every function takes the parent `serviceTypeID`.

| Function | Notes |
|---|---|
| `GetPlans(ctx context.Context, serviceTypeID string, params *PlansParams) (PlanListResponse, error)` | |
| `GetPlan(ctx context.Context, serviceTypeID, planID string) (PlanResponse, error)` | |
| `CreatePlan(ctx context.Context, serviceTypeID string, params *CreatePlanParams) (PlanResponse, error)` | See below. |
| `UpdatePlan(ctx context.Context, serviceTypeID, planID string, params *UpdatePlanParams) (PlanResponse, error)` | See below. |
| `DeletePlan(ctx context.Context, serviceTypeID, planID string) error` | |

```go
type CreatePlanParams struct {
	Title       string
	Public      bool
	SeriesID    string // omitted from the request body if empty
	SeriesTitle string // omitted from the request body if empty
}

type UpdatePlanParams struct {
	Title             string
	Public            *bool // nil = don't change
	RemindersDisabled *bool // nil = don't change; update-only attribute
}
```

`Public`/`RemindersDisabled` on `UpdatePlanParams` are pointers so "leave unchanged" and "explicitly set to false" are distinguishable — only non-nil fields are sent.

```go
plan, err := pco.CreatePlan(ctx, serviceTypeID, &pco.CreatePlanParams{Title: "This Sunday", Public: true})

remindersDisabled := true
plan, err = pco.UpdatePlan(ctx, serviceTypeID, plan.Data.ID, &pco.UpdatePlanParams{RemindersDisabled: &remindersDisabled})
```

`PlanAttributes` also includes read-only fields like `Dates`, `ItemsCount`, `PlanPeopleCount`, `NeededPositionsCount`, `SortDate`. `PlanRelationships` covers `ServiceType`, `Series`, `PreviousPlan`/`NextPlan`, `CreatedBy`/`UpdatedBy`.

## Songs

**[songs.go](../songs.go)**

A Song is an organization-wide song in the library (not scoped to a service type). No `UpdateSong` yet - create, read, and delete only.

| Function | Notes |
|---|---|
| `GetSongs(ctx context.Context, params *SongsParams) (SongListResponse, error)` | `params` may be `nil`. |
| `GetSong(ctx context.Context, id string) (SongResponse, error)` | |
| `CreateSong(ctx context.Context, params *CreateSongParams) (SongResponse, error)` | See below. |
| `DeleteSong(ctx context.Context, id string) error` | |

```go
type SongsParams struct {
	OrderBy string // "title", "created_at", "updated_at", or "last_scheduled_at"; prefix "-" to descend
	PerPage int
	Offset  int
}

type CreateSongParams struct {
	Title      string
	Author     string
	Admin      string
	Copyright  string
	CCLINumber int
	Themes     string
	Hidden     bool
}
```

```go
songs, err := pco.GetSongs(ctx, &pco.SongsParams{OrderBy: "-created_at", PerPage: 25})

song, err := pco.CreateSong(ctx, &pco.CreateSongParams{Title: "Great Are You Lord", Author: "Leslie Jordan, David Leonard, Jason Ingram"})
```

`CreateSongParams` has no `Notes` field on purpose - PCO rejects `notes` on create outright (`"notes cannot be assigned"`, confirmed live); a song's notes are their own sub-resource, not a plain Song attribute. Every other field is only sent when set (`Hidden` only when `true`), so a params struct with just `Title` set sends just `title`.

`SongAttributes` covers `Title`, `Author`, `CCLINumber`, `Copyright`, `Admin`, `Themes`, `Notes`, `Hidden`, `LastScheduledAt`, `CreatedAt`/`UpdatedAt`. Unlike other resources, `SongData` has no `Relationships` field - PCO's Song vertex doesn't return a `relationships` object; related data (arrangements, attachments, tags) is exposed as action links this SDK doesn't wrap yet.

## Arrangements & Keys

**[arrangements.go](../arrangements.go), [keys.go](../keys.go)**

An Arrangement is a named version of a Song (chord chart, BPM, meter, lyrics) - every function takes the parent `songID`. A Key is a named starting/ending key nested under one Arrangement - every Key function additionally takes `arrangementID`.

| Function | Notes |
|---|---|
| `GetArrangements(ctx context.Context, songID string, params *ArrangementsParams) (ArrangementListResponse, error)` | `params` may be `nil`. |
| `GetArrangement(ctx context.Context, songID, id string) (ArrangementResponse, error)` | |
| `CreateArrangement(ctx context.Context, songID string, params *CreateArrangementParams) (ArrangementResponse, error)` | See below. |
| `UpdateArrangement(ctx context.Context, songID, arrangementID string, params *UpdateArrangementParams) (ArrangementResponse, error)` | Same fields as `CreateArrangementParams` - see below. |
| `GetKeys(ctx context.Context, songID, arrangementID string, params *KeysParams) (KeyListResponse, error)` | `params` may be `nil`. |
| `GetKey(ctx context.Context, songID, arrangementID, id string) (KeyResponse, error)` | |
| `CreateKey(ctx context.Context, songID, arrangementID string, params *CreateKeyParams) (KeyResponse, error)` | See below. |

```go
type CreateArrangementParams struct {
	Name          string
	BPM           float64
	Meter         string
	ChordChartKey string
	Notes         string
	Length        int
}

type CreateKeyParams struct {
	Name        string
	StartingKey string
	EndingKey   string
}
```

```go
arrangement, err := pco.CreateArrangement(ctx, song.ID, &pco.CreateArrangementParams{
	Name:          "Default",
	ChordChartKey: "G",
	BPM:           72,
})

key, err := pco.CreateKey(ctx, song.ID, arrangement.Data.ID, &pco.CreateKeyParams{StartingKey: "G"})
```

`CreateArrangementParams`/`CreateKeyParams` only cover the fields worth setting when creating one from scratch, not PCO's full creatable set (chord chart formatting/print options are left at PCO's own defaults). Every field is only sent when set, same convention as `CreateSongParams`.

PCO auto-creates one "Default Arrangement" the moment a Song itself is created (confirmed live: `chord_chart_key`/`bpm`/`meter` all start empty/zero) - setting a real key/tempo on a freshly-created song means updating that one with `UpdateArrangement`, not calling `CreateArrangement` again (which would just leave two arrangements on the song):

```go
arrangements, err := pco.GetArrangements(ctx, song.ID, nil) // arrangements.Data[0] is the auto-created default

arrangement, err := pco.UpdateArrangement(ctx, song.ID, arrangements.Data[0].ID, &pco.UpdateArrangementParams{
	ChordChartKey: "G",
	BPM:           72,
})
```

`UpdateArrangementParams` mirrors `CreateArrangementParams` field-for-field, with one exception: `Notes` is a `*string`, not a plain `string`, so it can be explicitly cleared back to empty (`Notes: &empty`) - a plain string field can't distinguish "leave notes alone" from "clear notes," since both would send the zero value. Every other field is only sent when non-zero, same as create:

```go
type UpdateArrangementParams struct {
	Name          string
	BPM           float64
	Meter         string
	ChordChartKey string
	Notes         *string // nil = don't change; non-nil (including "") is sent
	Length        int
}
```

`KeyAttributes` has no `Capo` field - PCO's API doesn't expose one. The capo number shown in PCO's own UI is computed there from a starting key relative to an instrument's preferred key, not stored as data on this resource; there's nothing here to read or write for it.

`KeyAttributes.AlternateKeys` is `[]AlternateKey` (`Name`/`Pitch`), confirmed live against a real response - PCO's own attribute table documents this field as a bare `string`, which is wrong. `CreateKeyParams` has no `AlternateKeys` field since its create/update wire shape isn't confirmed the same way.

`ArrangementAttributes.SequenceFull` is `[]ArrangementSequenceStep` (`Label`/`Number`/`T`/`SID`), not `[]string` like its `Sequence`/`SequenceShort` siblings - PCO's own docs just say "array" with no element shape, and a live response (a manually-created song with a real chord chart) proved it's one object per section, not a bare string. `ArrangementSequenceStep.Number` is `StringOrNumber` rather than a plain `string` - PCO usually sends it as a string but has been observed (production, one real arrangement) sending a bare JSON number instead, which would otherwise fail the whole arrangement fetch.

## Items

**[items.go](../items.go)**

An Item is a row within a Plan's order of service (a song, a header, media, or a generic item). Every function takes the parent `serviceTypeID` and `planID`.

| Function | Notes |
|---|---|
| `GetItems(ctx context.Context, serviceTypeID, planID string, params *ItemsParams) (ItemListResponse, error)` | |
| `GetItem(ctx context.Context, serviceTypeID, planID, itemID string) (ItemResponse, error)` | |
| `CreateItem(ctx context.Context, serviceTypeID, planID string, params *CreateItemParams) (ItemResponse, error)` | |
| `UpdateItem(ctx context.Context, serviceTypeID, planID, itemID string, params *UpdateItemParams) (ItemResponse, error)` | No `Sequence` field - see `ReorderItems` below. |
| `DeleteItem(ctx context.Context, serviceTypeID, planID, itemID string) error` | |
| `ReorderItems(ctx context.Context, serviceTypeID, planID string, itemIDs []string) error` | See below. |

```go
type CreateItemParams struct {
	Title           string
	Description     string
	ItemType        string // one of the ItemType* constants
	ServicePosition string // one of the ServicePosition* constants
	Length          int    // seconds
	Sequence        int
	SongID          string // links the item to a library Song; "" for no link
	// ArrangementID links the item to one of that song's Arrangements
	// (relationships.arrangement) - an item created with just SongID gets
	// no arrangement relationship at all, unlike a song added through
	// Planning Center's own UI, so its chord chart/lyrics/structure won't
	// show up on the plan without this.
	ArrangementID string
	// KeyID links the item to one of that arrangement's Key sub-resources
	// (relationships.key) - a genuinely separate relationship from
	// ArrangementID: an item with only ArrangementID set still comes back
	// with no KeyName/key relationship. Setting KeyID is what actually
	// populates it.
	KeyID string
}
```

`item_type` and `service_position` are documented enums; use the exported constants rather than raw strings:

```go
const (
	ItemTypeSong   = "song"
	ItemTypeHeader = "header"
	ItemTypeMedia  = "media"
	ItemTypeItem   = "item" // default
)

const (
	ServicePositionPre    = "pre"
	ServicePositionDuring = "during"
	ServicePositionPost   = "post"
)
```

```go
item, err := pco.CreateItem(ctx, serviceTypeID, planID, &pco.CreateItemParams{
	Title:           "Opening Song",
	ItemType:        pco.ItemTypeSong,
	ServicePosition: pco.ServicePositionPre,
})
```

`ItemRelationships` covers `Plan`, `Song`, `Arrangement`, `Key` - all four decode as a bare `General{Type, ID}` regardless (that's the JSON:API relationship shape everywhere in this SDK), but `Arrangement`/`Key` can now be resolved to their full attributes with `GetArrangement`/`GetKey` above (`Song` likewise resolves against the Songs resource above).

```go
type UpdateItemParams struct {
	Title           *string
	Description     *string
	ServicePosition *string
	Length          *int
	// ArrangementID/KeyID link (or, set to a pointer to "", unlink) the
	// item's arrangement/key relationships - confirmed live: PATCHing
	// these works, unlike Sequence below.
	ArrangementID *string
	KeyID         *string
}
```

Every `UpdateItemParams` field is a pointer, not a plain `string`/`int`, for the same reason `Length` always was: a zero value (`0`, `""`) is legitimate data (an empty `Description`, a cleared `ArrangementID`), so it can't also double as "leave this alone" - only non-nil fields are sent, and a nil-vs-`""` pointer is what lets a caller explicitly clear `ArrangementID`/`KeyID` back to unlinked rather than merely omit them.

`UpdateItemParams` deliberately has no `Sequence` field - PATCHing an item's sequence directly is rejected by PCO (`"sequence cannot be assigned"`, confirmed live). Reordering is its own bulk action:

```go
err := pco.ReorderItems(ctx, serviceTypeID, planID, []string{item1.ID, item2.ID, item3.ID})
```

`ReorderItems` calls PCO's `item_reorder` plan action - confirmed directly against PCO's own machine-readable documentation API (`GET .../services/v2/documentation/2018-11-01/vertices/plan`, itself a plain JSON endpoint even though the human-facing docs site is a JS SPA that isn't crawlable). It expects **every** item's id in the plan, in the final order - there's no documented partial/delta form, so omitting an item likely misplaces it rather than leaving it alone.

## Item Notes & Item Note Categories

**[itemNotes.go](../itemNotes.go), [itemNoteCategories.go](../itemNoteCategories.go)**

An Item Note Category is configured once per Service Type (e.g. "Audio/Visual", "Band", "Vocals") and shared by every plan under it - read-only in this SDK. An Item Note is a note on one Item, belonging to exactly one category; every Item Note function takes the parent `serviceTypeID`/`planID`/`itemID`.

| Function | Notes |
|---|---|
| `GetItemNoteCategories(ctx context.Context, serviceTypeID string, params *ItemNoteCategoriesParams) (ItemNoteCategoryListResponse, error)` | `params` may be `nil`. Read-only - no write functions. |
| `GetItemNotes(ctx context.Context, serviceTypeID, planID, itemID string, params *ItemNotesParams) (ItemNoteListResponse, error)` | Lists every note on the item, across every category - see below. |
| `CreateItemNote(ctx context.Context, serviceTypeID, planID, itemID string, params *CreateItemNoteParams) (ItemNoteResponse, error)` | See below. |
| `UpdateItemNote(ctx context.Context, serviceTypeID, planID, itemID, noteID string, params *UpdateItemNoteParams) (ItemNoteResponse, error)` | `Content` only - see below. |
| `DeleteItemNote(ctx context.Context, serviceTypeID, planID, itemID, noteID string) error` | |

```go
type CreateItemNoteParams struct {
	Content            string
	ItemNoteCategoryID string
}

type UpdateItemNoteParams struct {
	Content string
}
```

```go
categories, err := pco.GetItemNoteCategories(ctx, serviceTypeID, nil)

note, err := pco.CreateItemNote(ctx, serviceTypeID, planID, itemID, &pco.CreateItemNoteParams{
	Content:            "Bring in-ear packs for the full band",
	ItemNoteCategoryID: categories.Data[0].ID,
})

note, err = pco.UpdateItemNote(ctx, serviceTypeID, planID, itemID, note.Data.ID, &pco.UpdateItemNoteParams{
	Content: "Bring in-ear packs for the full band, including the sub",
})
```

`ItemNoteCategoryID` is required on create - PCO rejects a create with no category (confirmed live) - and, per PCO's own documentation, can never be changed on an existing note afterward: to move a note to a different category, delete it and create a new one. `UpdateItemNoteParams` only has `Content` accordingly - it's the only update-assignable attribute PCO documents for Item Note.

`GetItemNotes` has no per-category filter param documented, so a caller wanting "the note for category X" filters the returned list client-side by `Attributes.CategoryName` (or the `ItemNoteCategory` relationship id) - `CategoryName` is denormalized onto the note itself (confirmed live), so displaying existing notes doesn't need a separate category fetch/join.

## Teams

**[teams.go](../teams.go)**

A Team is a group people serve on within a Service Type (e.g. "Band", "Tech", "Hospitality"). Read-only - PCO doesn't document create/update/destroy for this resource.

| Function | Notes |
|---|---|
| `GetTeams(ctx context.Context, serviceTypeID string, params *TeamsParams) (TeamListResponse, error)` | `params` may be `nil`. |

```go
teams, err := pco.GetTeams(ctx, serviceTypeID, &pco.TeamsParams{OrderBy: "name"})
```

`TeamAttributes` covers `Name`, `Sequence`, `ScheduleTo` (`"plan"` for a normal plan-level team, or a split/"time"-level team), `DefaultStatus` (the default `PlanPerson.status` for new assignments on this team), `SecureTeam`, `RehearsalTeam`, and more.

## Team Positions

**[teamPositions.go](../teamPositions.go)**

A Team Position is a named slot within a Team (e.g. "Drums", "Sound Tech"). Read-only - like Team, PCO doesn't document write support. **Not addressable as a top-level resource** - `GET /services/v2/team_positions/{id}` 404s live; a position only exists nested under its team, which is why every function below takes `teamID`.

| Function | Notes |
|---|---|
| `GetTeamPositions(ctx context.Context, teamID string, params *TeamPositionsParams) (TeamPositionListResponse, error)` | `params` may be `nil`. |

```go
positions, err := pco.GetTeamPositions(ctx, teamID, &pco.TeamPositionsParams{OrderBy: "name"})
```

## Needed Positions

**[neededPositions.go](../neededPositions.go)**

A Needed Position is one open slot a plan needs filled - team + position name + how many. Read-only in this SDK; PCO documents create/update/destroy for it, but every create attempt against a real account returned a 500 regardless of request shape (team relationship, `team_position_id` attribute, with/without a `time_id` - none worked), so this SDK doesn't wrap writes for it yet. Configure needed positions in Planning Center's own UI for now.

| Function | Notes |
|---|---|
| `GetNeededPositions(ctx context.Context, serviceTypeID, planID string, params *NeededPositionsParams) (NeededPositionListResponse, error)` | `params` may be `nil`. |

```go
needed, err := pco.GetNeededPositions(ctx, serviceTypeID, planID, nil)
```

`NeededPositionAttributes` covers `Quantity`, `TeamPositionName` (plain text, matching `PlanPerson.team_position_name` - not a relationship to a Team Position id), and `ScheduledTo`. `NeededPositionRelationships` covers `Team` and `Plan`.

## Team Members (PlanPerson)

**[teamMembers.go](../teamMembers.go)**

A Team Member (PCO's `PlanPerson` type) is a person actually assigned to fill a position on a specific plan - as opposed to Needed Position, which is "what's still open." Every function takes the parent `serviceTypeID`/`planID` except the person-scoped history lookup.

| Function | Notes |
|---|---|
| `GetTeamMembers(ctx context.Context, serviceTypeID, planID string, params *TeamMembersParams) (PlanPersonListResponse, error)` | `params` may be `nil`. |
| `CreateTeamMember(ctx context.Context, serviceTypeID, planID string, params *CreateTeamMemberParams) (PlanPersonResponse, error)` | See below. |
| `UpdateTeamMember(ctx context.Context, serviceTypeID, planID, planPersonID string, params *UpdateTeamMemberParams) (PlanPersonResponse, error)` | Partial update - see below. |
| `DeleteTeamMember(ctx context.Context, serviceTypeID, planID, planPersonID string) error` | See note below. |
| `GetPersonPlanPeople(ctx context.Context, personID string, params *PersonPlanPeopleParams) (PlanPersonListResponse, error)` | A person's own assignment history - see below. |

```go
type CreateTeamMemberParams struct {
	PersonID         string
	TeamID           string
	TeamPositionName string
	Status           string // PlanPersonStatus* constant; defaults to Unconfirmed if empty
	Notes            string // confirmed both create- and update-assignable, unlike Item.Sequence
}
```

```go
member, err := pco.CreateTeamMember(ctx, serviceTypeID, planID, &pco.CreateTeamMemberParams{
	PersonID:         personID,
	TeamID:           teamID,
	TeamPositionName: "Drums",
})

err = pco.DeleteTeamMember(ctx, serviceTypeID, planID, member.Data.ID)
```

```go
type UpdateTeamMemberParams struct {
	Status           string
	DeclineReason    string
	Notes            string
	TeamPositionName string
}
```

```go
member, err = pco.UpdateTeamMember(ctx, serviceTypeID, planID, member.Data.ID, &pco.UpdateTeamMemberParams{
	Status: pco.PlanPersonStatusConfirmed,
})
```

`UpdateTeamMemberParams` is a partial update - only non-empty fields are sent - and deliberately doesn't cover every field PCO's `update_assignable` list allows (`person_id`/`team_id` are in that list too, meaning PCO would let you re-point an existing assignment at a different person/team entirely). Re-assigning who's filling a position is modeled as delete-and-recreate instead, matching how the rest of this SDK treats person/song relationships as fixed at creation, not something to sneak into a partial update.

**`DeleteTeamMember` uses the plan-scoped path, not PCO's documented person-scoped one.** PCO's docs point delete at `/people/{person_id}/plan_people/{plan_person_id}`, but that path returned a 404 live against a real (older/past) plan's assignment while the plan-scoped `/service_types/{id}/plans/{id}/team_members/{id}` path it was created on deleted it without issue. Confirmed reliable regardless of the plan's age, so this SDK always uses the plan-scoped path for both create and delete.

`PlanPersonStatus*` covers the documented `status` values - PCO accepts either the letter code or the word, and returns the letter code in responses (confirmed live), so these consts use that form: `PlanPersonStatusConfirmed` (`"C"`), `PlanPersonStatusUnconfirmed` (`"U"`), `PlanPersonStatusDeclined` (`"D"`).

```go
history, err := pco.GetPersonPlanPeople(ctx, personID, &pco.PersonPlanPeopleParams{
	TeamID:  teamID,
	Include: []string{"plan"}, // pulls each assignment's Plan (with sort_date) into Included, one call instead of N
})
```

`GetPersonPlanPeople` filters by team server-side (`where[team_id]`, confirmed via `can_query_by`) but not by position name - filter `Data` client-side against `Attributes.TeamPositionName` if you need one specific position's history.

## Person Team Position Assignments

**[personTeamPositionAssignments.go](../personTeamPositionAssignments.go)**

A Person Team Position Assignment is "this person can serve this position" - and, via `SchedulePreference`, how often they want to. **Important:** `schedule_preference` reads/writes as **one value per person**, not per position, despite living on this per-position resource - confirmed live by patching one assignment's preference and observing every other position the same person is eligible for immediately reflect the new value, with no other write. Plan around a person having a single serving-frequency preference, not a different one per position.

Not addressable as a top-level resource, same as Team Position - every function takes `teamID` and `teamPositionID`.

| Function | Notes |
|---|---|
| `GetPersonTeamPositionAssignments(ctx context.Context, teamID, teamPositionID string, params *PersonTeamPositionAssignmentsParams) (PersonTeamPositionAssignmentListResponse, error)` | `params` may be `nil`. Also the position's "who's eligible" list. |
| `CreatePersonTeamPositionAssignment(ctx context.Context, teamID, teamPositionID string, params *CreatePersonTeamPositionAssignmentParams) (PersonTeamPositionAssignmentResponse, error)` | Also makes the person eligible for the position - PCO doesn't separate the two. |
| `UpdatePersonTeamPositionAssignment(ctx context.Context, teamID, teamPositionID, assignmentID string, params *UpdatePersonTeamPositionAssignmentParams) (PersonTeamPositionAssignmentResponse, error)` | |
| `DeletePersonTeamPositionAssignment(ctx context.Context, teamID, teamPositionID, assignmentID string) error` | Removes eligibility entirely. |

```go
assignment, err := pco.CreatePersonTeamPositionAssignment(ctx, teamID, teamPositionID, &pco.CreatePersonTeamPositionAssignmentParams{
	PersonID:           personID,
	SchedulePreference: pco.SchedulePreferenceEveryOtherWeek,
})
```

`SchedulePreference*` is the verbatim, complete set PCO documents: `Unavailable`, `EveryWeek`, `EveryOtherWeek`, `Every3rdWeek` .. `Every6thWeek`, `OnceAMonth`, `TwiceAMonth`, `ThreeTimesAMonth`, `OnceAQuarter`, `ChooseWeeks`. `ChooseWeeks` needs a matching `PreferredWeeks` (`[]string` of week numbers, e.g. `["1", "3", "5"]`) - this SDK exposes the field but doesn't specially validate it.

Adding a brand-new (never-logged-in) Person here fails with `"person must exist"` even when the id is valid - confirmed live that Planning Center's Services side needs its own record for a person before they're referenceable there at all (`GET /services/v2/people/{id}` 404s for a person that only exists in the People app), and this SDK found no API path to create that Services-side record (`POST /services/v2/teams/{id}/people` 403s: `"cannot create a Person"`, tried several body shapes). A person needs Services access granted once through Planning Center's own People admin UI before they're assignable here.

## Blockouts

**[blockouts.go](../blockouts.go)**

A Blockout is a person's self-declared unavailability window. Read-only in this SDK by design - PCO documents create/update/destroy, but blockouts are meant to stay a volunteer's own action in Planning Center, not something an integration sets on their behalf.

| Function | Notes |
|---|---|
| `GetBlockouts(ctx context.Context, personID string, params *BlockoutsParams) (BlockoutListResponse, error)` | `params` may be `nil`. `Filter`: `"past"` or `"future"`. |

```go
blockouts, err := pco.GetBlockouts(ctx, personID, &pco.BlockoutsParams{Filter: "future"})
```

`BlockoutAttributes` covers `Reason`, `StartsAt`/`EndsAt` (a single non-recurring window), and `RepeatFrequency`/`RepeatInterval`/`RepeatPeriod`/`RepeatUntil` for a recurring blockout - this SDK decodes the recurrence fields but doesn't expand them into individual dates; that's on the caller.

---

All `Create*`/`Update*` functions return an error if `params` is `nil`, and a `*pco.RequestError` if PCO rejects the request (see [Errors](../README.md#errors)).
