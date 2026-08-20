# Services

Wraps a starting subset of the [Services v2 API](https://api.planningcenteronline.com/docs/apps/services): the order-of-service side (Service Types, Plans, Items, Songs) and the people-scheduling side (Teams, Team Positions, Needed Positions, Team Members / PlanPerson, Person Team Position Assignments, Blockouts).

The Services API has ~65 resources total — the ones below plus Arrangements, Attachments, Schedules, and more remain unimplemented. See [Extending](../README.md#extending) in the root README for how to add another one; `https://api.planningcenteronline.com/docs/apps/services` lists every resource and its documentation path.

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

A Song is an organization-wide song in the library (not scoped to a service type). No `UpdateSong`/`DeleteSong` yet - only create and read.

| Function | Notes |
|---|---|
| `GetSongs(ctx context.Context, params *SongsParams) (SongListResponse, error)` | `params` may be `nil`. |
| `GetSong(ctx context.Context, id string) (SongResponse, error)` | |
| `CreateSong(ctx context.Context, params *CreateSongParams) (SongResponse, error)` | See below. |

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

`ItemRelationships` covers `Plan`, `Song`, `Arrangement`, `Key` — decoding those requires the not-yet-implemented Arrangement/Key resources; for now you'll get back a bare `General{Type, ID}` for those two (`Song` decodes against the Songs resource above).

`UpdateItemParams` deliberately has no `Sequence` field - PATCHing an item's sequence directly is rejected by PCO (`"sequence cannot be assigned"`, confirmed live). Reordering is its own bulk action:

```go
err := pco.ReorderItems(ctx, serviceTypeID, planID, []string{item1.ID, item2.ID, item3.ID})
```

`ReorderItems` calls PCO's `item_reorder` plan action - confirmed directly against PCO's own machine-readable documentation API (`GET .../services/v2/documentation/2018-11-01/vertices/plan`, itself a plain JSON endpoint even though the human-facing docs site is a JS SPA that isn't crawlable). It expects **every** item's id in the plan, in the final order - there's no documented partial/delta form, so omitting an item likely misplaces it rather than leaving it alone.

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
| `DeleteTeamMember(ctx context.Context, serviceTypeID, planID, planPersonID string) error` | See note below. |
| `GetPersonPlanPeople(ctx context.Context, personID string, params *PersonPlanPeopleParams) (PlanPersonListResponse, error)` | A person's own assignment history - see below. |

```go
type CreateTeamMemberParams struct {
	PersonID         string
	TeamID           string
	TeamPositionName string
	Status           string // PlanPersonStatus* constant; defaults to Unconfirmed if empty
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
