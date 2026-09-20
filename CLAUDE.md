# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

An unofficial Go SDK for the Planning Center API (`github.com/drkgrntt/go-pco`), built resource-by-resource against PCO's live API rather than generated from a spec. Requires Go 1.26+ (generics used throughout).

## Commands

```sh
go build ./...          # compile
go test ./...            # run all tests
go test -run TestGetPlans ./...   # run a single test
go vet ./...
```

There is no live-API test suite - all tests run against a local `httptest.Server`, so no PCO credentials are needed to develop or test.

## Architecture

Every resource lives in one `<resource>.go` + `<resource>_test.go` pair at the repo root (no subpackages). Each file follows an identical shape, described in detail in the README's "Core concepts" and "Extending" sections - **read those before adding or modifying a resource**. In short, per resource:

1. `XAttributes` / `XRelationships` structs matching PCO's documented fields.
2. `XData` combining `Type`/`ID` with those.
3. `XResponse` (single) / `XListResponse` (collection) envelopes: `Data`, `Included []any`, `Links`, `Meta`.
4. `XParams` for list filters (`PerPage`/`Offset` plus `where[]` filters), and separate `CreateXParams`/`UpdateXParams` with only the attributes that endpoint documents as assignable (pointer fields where the API distinguishes "omit" from "clear to empty/null").
5. Plain functions (not methods) - `GetXs`, `GetX`, `CreateX`, `UpdateX`, `DeleteX` - built on `pco.NewQueryParams()` / `pco.NewRequestBody()` (or `NewRequestBodyWithRelationships` when linking another resource) / `pco.NewRequest[...]`.

`pco.go` holds everything shared: `NewRequest[Response]` (the generic HTTP call - auth, body encoding, error decoding, JSON unmarshal into the given envelope type), `QueryParams` (builds `where[]`/`include`/`order`/`filter`/`per_page`/`offset`), `RequestBody`/`RequestData` (JSON:API request shape), `RequestError` (wraps the JSON:API `errors[]` array from any non-2xx response), and `WithAccessToken` (see Auth below).

Before writing a new resource's structs, look up its documented attributes/relationships/actions at `https://api.planningcenteronline.com/docs/apps/<product>/versions/<version>/vertices/<resource>` - don't guess field names, PCO's docs are the source of truth. Copy the shape of an existing file with similar nesting: `serviceTypes.go` for a top-level resource, `plans.go`/`items.go` for one nested under another.

### Auth

Every request authenticates one of two ways, decided per-call inside `NewRequest`:
- Default: HTTP Basic Auth from the `PCO_CLIENT_ID`/`PCO_SECRET` env vars (a Personal Access Token or OAuth client credentials) - fine for scripts/single-tenant use and is what the test suite implicitly relies on being absent/irrelevant.
- Per-user: if the `context.Context` passed in carries a token via `pco.WithAccessToken(ctx, token)`, that token is sent as `Authorization: Bearer <token>` instead - this is how a multi-user app makes requests as a specific signed-in person against their own PCO org/permissions. The SDK does not perform the OAuth flow or refresh tokens; the caller supplies a valid token.

### Testing pattern

`pco_test.go` holds the shared harness used by every `*_test.go`:
- `startTestServer(t, handler)` - spins up an `httptest.Server` and points the package-level `baseURL` var at it for the test's duration (this is *why* `baseURL` is a `var`, not a `const`).
- `decodeBody` / `attributes` / `relationships` - decode a JSON:API request body in the handler to assert on.
- `writeJSON` - write a canned JSON:API response from the handler.

Each resource test asserts the HTTP method, path, query string, and/or request body a function produced, and that the response unmarshals correctly - mirroring the resource file 1:1 (`plans_test.go` tests `plans.go`).

## Known API quirks worth knowing before touching these files

- Needed Positions are read-only in this SDK despite PCO's docs describing create/update/destroy: every create attempt against a real account 500s regardless of request shape. Don't add writes here without re-verifying against a live account.
- `ArrangementSequenceStep.Number` (arrangements.go) is `StringOrNumber`, not a plain `string` - PCO usually sends it as a string but has sent a bare JSON number for at least one real arrangement in production, which broke that song's entire arrangement fetch until this was caught (via a downstream app's prod logs). `StringOrNumber`/`StringOrStrings` (songs.go) are the two examples so far of this "PCO's JSON typing for a field isn't stable" class of quirk - default to a flexible-decode type rather than a plain typed field when a new field looks similarly fuzzy in PCO's docs.
- `docs/*.md` (people.md, services.md, webhooks.md) capture per-module detail and quirks like the above discovered against a real PCO account - check the relevant one before assuming an endpoint behaves exactly as documented upstream.
- `AssignSongTags` (tags.go) does **not** take the "obvious" JSON:API attribute shape - confirmed live (2026-09-18): the only body PCO accepts for `POST /services/v2/songs/{songID}/assign_tags` is `{"data":{"type":"TagGroup","relationships":{"tags":{"data":[{"type":"Tag","id":"..."}]}}}}` (a flat `tag_ids` attribute body 400s; a bare-array `data` body 422s). It's also a **full replace of that tag group's assignment on the song per call, not additive** - calling it again with a different, non-overlapping tag set removes the first call's tags rather than adding to the new ones; an empty slice clears the group entirely. Callers must always pass the complete desired tag-id set for whichever group they're changing.
- TagGroup/Tag create/update/destroy are deliberately unimplemented (tagGroups.go) - like Needed Positions above, this now looks like a resource PCO deliberately keeps off the OAuth write surface entirely, not a solvable permission question. `POST /services/v2/tag_groups` 403'd ("cannot create a TagGroup") for two separate real accounts on two different days (2026-09-18, 2026-09-19), and the second attempt ruled out every softer explanation live: the person held the Services **Administrator** role in that exact org, the org already had real tag groups (not an empty-org quirk), creating the same tag group by hand worked fine in PCO's own Services UI as that admin, and `/people/v2/tag_groups` 404s (confirming `/services/v2/tag_groups` - which 403s, not 404s - is the right resource, just not writable this way). Don't guess the body shape from JSON:API convention the way `AssignSongTags` above already proved wrong for a different tag endpoint - if this ever needs revisiting, it's PCO's API surface that would need to change, not the caller's permission level.
