# Services

Wraps a starting subset of the [Services v2 API](https://api.planningcenteronline.com/docs/apps/services): Service Types, the Plans within them, and the Items within those. Full CRUD is implemented for all three, following the same nested-resource pattern (`service_types/{id}/plans/{id}/items/{id}`).

The Services API has ~65 resources total (Songs, Arrangements, Teams, Team Members, Plan People, Attachments, Schedules, and more) — only this core trio is implemented so far. See [Extending](../README.md#extending) in the root README for how to add another one; `https://api.planningcenteronline.com/docs/apps/services` lists every resource and its documentation path.

## Service Types

**[serviceTypes.go](../serviceTypes.go)**

A Service Type is a container for plans (e.g. "Sunday Service", "Youth Service").

| Function | Notes |
|---|---|
| `GetServiceTypes(params *ServiceTypesParams) (ServiceTypeListResponse, error)` | `params` may be `nil`. |
| `GetServiceType(id string) (ServiceTypeResponse, error)` | |
| `CreateServiceType(params *CreateServiceTypeParams) (ServiceTypeResponse, error)` | `{Name string}` |
| `UpdateServiceType(id string, params *UpdateServiceTypeParams) (ServiceTypeResponse, error)` | `{Name string}` |
| `DeleteServiceType(id string) error` | |

`CreateServiceTypeParams`/`UpdateServiceTypeParams` only expose `Name` — PCO's docs didn't enumerate the full set of creatable/updatable attributes for this resource the way they did for Plan and WebhookSubscription, so this is a conservative starting point. `ServiceTypeAttributes` (read-only beyond `Name`) also includes `Frequency`, `Sequence`, `ArchivedAt`, `AttachmentTypesEnabled`, permission strings, etc. — see [serviceTypes.go](../serviceTypes.go).

```go
st, err := pco.CreateServiceType(&pco.CreateServiceTypeParams{Name: "Youth Service"})
```

## Plans

**[plans.go](../plans.go)**

A Plan is a single service event within a Service Type (e.g. "This Sunday"). Every function takes the parent `serviceTypeID`.

| Function | Notes |
|---|---|
| `GetPlans(serviceTypeID string, params *PlansParams) (PlanListResponse, error)` | |
| `GetPlan(serviceTypeID, planID string) (PlanResponse, error)` | |
| `CreatePlan(serviceTypeID string, params *CreatePlanParams) (PlanResponse, error)` | See below. |
| `UpdatePlan(serviceTypeID, planID string, params *UpdatePlanParams) (PlanResponse, error)` | See below. |
| `DeletePlan(serviceTypeID, planID string) error` | |

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
plan, err := pco.CreatePlan(serviceTypeID, &pco.CreatePlanParams{Title: "This Sunday", Public: true})

remindersDisabled := true
plan, err = pco.UpdatePlan(serviceTypeID, plan.Data.ID, &pco.UpdatePlanParams{RemindersDisabled: &remindersDisabled})
```

`PlanAttributes` also includes read-only fields like `Dates`, `ItemsCount`, `PlanPeopleCount`, `NeededPositionsCount`, `SortDate`. `PlanRelationships` covers `ServiceType`, `Series`, `PreviousPlan`/`NextPlan`, `CreatedBy`/`UpdatedBy`.

## Items

**[items.go](../items.go)**

An Item is a row within a Plan's order of service (a song, a header, media, or a generic item). Every function takes the parent `serviceTypeID` and `planID`.

| Function | Notes |
|---|---|
| `GetItems(serviceTypeID, planID string, params *ItemsParams) (ItemListResponse, error)` | |
| `GetItem(serviceTypeID, planID, itemID string) (ItemResponse, error)` | |
| `CreateItem(serviceTypeID, planID string, params *CreateItemParams) (ItemResponse, error)` | |
| `UpdateItem(serviceTypeID, planID, itemID string, params *UpdateItemParams) (ItemResponse, error)` | |
| `DeleteItem(serviceTypeID, planID, itemID string) error` | |

```go
type CreateItemParams struct {
	Title           string
	Description     string
	ItemType        string // one of the ItemType* constants
	ServicePosition string // one of the ServicePosition* constants
	Length          int    // seconds
	Sequence        int
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
item, err := pco.CreateItem(serviceTypeID, planID, &pco.CreateItemParams{
	Title:           "Opening Song",
	ItemType:        pco.ItemTypeSong,
	ServicePosition: pco.ServicePositionPre,
})
```

`ItemRelationships` covers `Plan`, `Song`, `Arrangement`, `Key` — decoding those requires the not-yet-implemented Song/Arrangement/Key resources; for now you'll get back a bare `General{Type, ID}` for each.

---

All `Create*`/`Update*` functions return an error if `params` is `nil`, and a `*pco.RequestError` if PCO rejects the request (see [Errors](../README.md#errors)).
