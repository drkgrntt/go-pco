# go-pco

An unofficial Go SDK for the [Planning Center](https://www.planningcenter.com/) API. Not affiliated with or endorsed by Planning Center.

This is a work in progress, built out resource-by-resource as needed rather than generated from a spec. See [Status & roadmap](#status--roadmap) for what's covered today.

## Installation

```sh
go get github.com/drkgrntt/go-pco
```

Requires Go 1.26+ (generics are used throughout).

## Authentication

Every request is signed with HTTP Basic Auth using two environment variables:

```sh
export PCO_CLIENT_ID=your_application_id
export PCO_SECRET=your_secret
```

These can be a [Personal Access Token](https://developer.planning.center/docs/#/overview/authentication) (simplest, good for scripts/single-org use — the "app ID" and "secret" PCO gives you when you create a PAT) or an OAuth application's client credentials. This is the fallback auth every call uses when nothing more specific is provided.

For a real multi-user app - each signed-in person's requests acting as them, against their own PCO organization and permissions, rather than everyone sharing one PAT - carry their OAuth access token through `context.Context` instead:

```go
ctx := pco.WithAccessToken(context.Background(), userAccessToken)
people, err := pco.GetPeople(ctx, &pco.PeopleParams{LastName: "Lovelace"})
```

When a context carries a token this way, every function authenticates with `Authorization: Bearer <token>` instead of the PAT - per call, not per-process, so concurrent requests for different users on the same server never cross streams. The SDK doesn't handle obtaining that token itself (the OAuth authorization-code flow, or refreshing an expired one) - that's still on the caller.

## Quick start

```go
package main

import (
	"fmt"
	"log"

	"github.com/drkgrntt/go-pco"
)

func main() {
	people, err := pco.GetPeople(ctx, &pco.PeopleParams{LastName: "Lovelace"})
	if err != nil {
		log.Fatal(err)
	}

	for _, person := range people.Data {
		fmt.Println(person.Attributes.FirstName, person.Attributes.LastName)
	}
}
```

## Core concepts

Every resource file follows the same shape: an `Attributes` struct, a `Relationships` struct, a `*Data` struct combining them, response envelope struct(s) (`*Response` for a single resource, `*ListResponse` for a collection), a `*Params` struct for inputs, and a set of plain functions (not methods) that call `pco.NewRequest`. The pieces below (from [pco.go](pco.go)) are shared by all of them.

### Errors

Any non-2xx response is returned as a `*pco.RequestError`, which carries the parsed JSON:API `errors[]` array from the response body:

```go
_, err := pco.CreatePerson(ctx, &pco.CreatePersonParams{})
var reqErr *pco.RequestError
if errors.As(err, &reqErr) {
	fmt.Println(reqErr.StatusCode) // e.g. 422
	for _, apiErr := range reqErr.Errors {
		fmt.Println(apiErr.Title, apiErr.Detail)
	}
}
```

### Pagination

List params structs (`PeopleParams`, `PlansParams`, etc.) accept `PerPage` and `Offset`. Every list response's `Links.Next` is set to the next page's URL when there is one:

```go
params := &pco.PeopleParams{PerPage: 100}
for {
	page, err := pco.GetPeople(ctx, params)
	if err != nil {
		log.Fatal(err)
	}
	// ...use page.Data...

	if page.Links.Next == "" {
		break
	}
	page, err = pco.NewRequest[pco.PersonListResponse](ctx, "GET", page.Links.Next, nil)
	if err != nil {
		log.Fatal(err)
	}
	params = nil // page.Links.Next already has offset/per_page baked in
}
```

### Filtering & querying

Internally, params get turned into `where[]`/`include`/`order`/`per_page`/`offset` query params via `pco.QueryParams`. You won't normally build one of these yourself — it's exposed mainly so you can call `pco.NewRequest` directly against an endpoint this SDK doesn't wrap yet:

```go
q := pco.NewQueryParams().Where("first_name", "Ada").Include("emails").PerPage(25)
response, err := pco.NewRequest[pco.PersonListResponse](ctx, "GET", "https://api.planningcenteronline.com/people/v2/people"+q.Encode(), nil)
```

### Request bodies

Create/update functions build their body with `pco.NewRequestBody(map[string]any{...})`, which wraps attributes in the `{"data":{"attributes":{...}}}` shape PCO's JSON:API expects. `any` values are used (not `string`) so booleans, numbers, and arrays serialize correctly. `pco.NewRequestBodyWithRelationships(attributes, relationships)` is the same, plus a `{"data":{"relationships":{...}}}` object, for creates/updates that link another resource (e.g. an Item linked to a Song) rather than just set plain attributes.

## Modules

| Module | Docs | Covers |
|---|---|---|
| People | [docs/people.md](docs/people.md) | People, Addresses, Emails, Phone Numbers |
| Webhooks | [docs/webhooks.md](docs/webhooks.md) | Subscriptions, Available Events, Event deliveries, receiving & verifying incoming webhooks |
| Services | [docs/services.md](docs/services.md) | Service Types, Plans, Items, Songs, Teams, Team Positions, Needed Positions, Team Members (PlanPerson), Person Team Position Assignments, Blockouts |

## Testing

```sh
go test ./...
```

Tests run against a local `httptest.Server` — no live PCO account or credentials needed. Each test file mirrors the resource file it covers (e.g. `plans_test.go` tests `plans.go`) and asserts the HTTP method, path, query string, and/or request body the function produced, plus that the response unmarshals correctly. `pco_test.go` holds the shared harness (`startTestServer`, `decodeBody`, `attributes`, `relationships`, `writeJSON`) and the core `NewRequest`/`QueryParams`/error-handling tests.

## Extending

To add a new resource, copy the shape of an existing file with a similar level of nesting — [serviceTypes.go](serviceTypes.go) for a top-level resource, [plans.go](plans.go) or [items.go](items.go) for one nested under another. In outline:

1. `XAttributes` / `XRelationships` structs matching the resource's documented attributes/relationships.
2. `XData` combining them with `Type`/`ID`.
3. `XResponse` (single) and `XListResponse` (collection) envelopes — `Data`, `Included []any`, `Links`, `Meta`.
4. An `XParams` struct for list filters (`PerPage`/`Offset`, plus any `where[]` filters) and separate `CreateXParams`/`UpdateXParams` structs for only the attributes that endpoint documents as assignable.
5. `GetXs`, `GetX`, `CreateX`, `UpdateX`, `DeleteX` functions following the existing naming pattern, using `pco.NewQueryParams()` / `pco.NewRequestBody()` / `pco.NewRequest[...]`.
6. A matching `_test.go` asserting method, path, query/body construction, and response decoding — see any existing `*_test.go` for the pattern.

Look up the resource's attributes/relationships/actions at `https://api.planningcenteronline.com/docs/apps/<product>/versions/<version>/vertices/<resource>` before writing the struct — don't guess field names, PCO's docs are the source of truth.

## Status & roadmap

Implemented: People (People, Addresses, Emails, Phone Numbers), Webhooks (full), Services (Service Types, Plans, Items, Songs - create/read; Teams, Team Positions - read-only; Needed Positions - read-only, see below; Team Members/PlanPerson - create/read/delete; Person Team Position Assignments - full CRUD; Blockouts - read-only by design).

Not yet implemented, in roughly the order they'd likely matter for a Services-focused integration:
- Services: Needed Position create/update/delete (every request shape tried 500s live on a real account regardless of body - see [docs/services.md](docs/services.md#needed-positions)), Song update/delete, Arrangements, Item Notes, Schedules, Attachments, and the ~45 remaining resources listed under `/docs/apps/services`.
- Other PCO products entirely: Giving, Check-Ins, Calendar, Groups, Publishing, Registrations, Resources.

Pull requests / additions following the pattern in [Extending](#extending) are the intended way this grows.
