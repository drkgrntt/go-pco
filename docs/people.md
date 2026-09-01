# People

Wraps the [People v2 API](https://api.planningcenteronline.com/docs/apps/people): people records and three of their sub-resources (addresses, emails, phone numbers). All paths are rooted at `people/v2/people`.

## Organization

**[organization.go](../organization.go)**

### `GetOrganization(ctx context.Context) (OrganizationResponse, error)`

```go
org, err := pco.GetOrganization(ctx)
name := org.Data.Attributes.Name
```

No `id` parameter - PCO's People API is single-tenant per credential (a PAT or an access token is always scoped to exactly one organization), and the organization itself is the root of that API: `GET https://api.planningcenteronline.com/people/v2` returns it directly as `data`, unlike every other resource here, which is addressed by an `id` segment under `people/v2/...`. PCO's own docs list `Organization` as a People API vertex without spelling out its path; `people/v2/organization` 404s.

`OrganizationAttributes` covers `Name`, `ChurchCenterSubdomain`, `TimeZone`, and `CountryCode` - the fields most callers actually need - not every attribute PCO's docs list (e.g. `Grades`, used by children's ministry check-in).

## People

**[people.go](../people.go)**

### `GetPeople(ctx context.Context, params *PeopleParams) (PersonListResponse, error)`

```go
type PeopleParams struct {
	FirstName         string
	LastName          string
	Email             string // matched against PCO's search_name_or_email filter - fuzzy, name or email
	SearchPhoneNumber string
	IDs               []string // where[id]=1,2,3 - fetch a known batch of ids in one call
	Include           []string // e.g. "addresses", "emails", "phone_numbers", "primary_campus"
	OrderBy           string   // a can_order_by field, e.g. "last_name" or "-created_at"
	PerPage           int
	Offset            int
}
```

`params` may be `nil` to list everyone (paginated). Any non-empty field adds a `where[]` filter; `PerPage`/`Offset` control pagination (see [Pagination](../README.md#pagination) in the root README). `Include` pulls related resources into the response's `Included []any` array.

```go
people, err := pco.GetPeople(ctx, &pco.PeopleParams{LastName: "Lovelace", Include: []string{"emails"}})
```

`IDs` resolves a known set of person ids in one request instead of one
`GetPerson` call each - confirmed live that People#index's `can_query_by`
includes `id`, and it's a normal comma-joined `where[]` filter like any
other. Still paginates like any other list call, so set `PerPage` to at
least `len(IDs)` (up to PCO's per-page ceiling) or the response silently
truncates:

```go
people, err := pco.GetPeople(ctx, &pco.PeopleParams{IDs: []string{"1", "2", "3"}, PerPage: 3})
```

### `GetPerson(ctx context.Context, id string) (PersonResponse, error)`

```go
person, err := pco.GetPerson(ctx, personID)
```

### `CreatePerson(ctx context.Context, params *CreatePersonParams) (PersonCreateResponse, error)`

```go
type CreatePersonParams struct {
	FirstName string
	LastName  string
}
```

`params` must not be `nil`.

```go
person, err := pco.CreatePerson(ctx, &pco.CreatePersonParams{FirstName: "Ada", LastName: "Lovelace"})
personID := person.Data.ID
```

### `PersonAttributes`

The full set of attributes PCO returns for a person (name fields, `Status`, `Birthdate`, `Membership`, permission flags, `ResourcePermissionFlags`, timestamps, etc.) — see [people.go](../people.go) for the complete list. `PersonRelationships` covers every relationship the People#index/#show endpoints document as includable (`PrimaryCampus`, `InactiveReason`, `MaritalStatus`, `NamePrefix`, `NameSuffix`, `Organization`, `School` as `HasOneRelationship`; `Addresses`, `Emails`, `FieldData`, `Households`, `PersonApps`, `PhoneNumbers`, `PlatformNotifications`, `SocialProfiles` as `HasManyRelationship`) - each only has data once the matching value is passed to `PeopleParams.Include`; an unrequested relationship just decodes to its zero value.

## Addresses

**[addresses.go](../addresses.go)**

### `CreateAddress(ctx context.Context, personID string, params *AddressCreateParams) (AddressCreateResponse, error)`

```go
type AddressCreateParams struct {
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	Zip          string
	CountryCode  string
	Location     string // e.g. "Home", "Work"
	Primary      bool
}
```

```go
address, err := pco.CreateAddress(ctx, personID, &pco.AddressCreateParams{
	AddressLine1: "123 Main St",
	City:         "Springfield",
	State:        "IL",
	Zip:          "62704",
	CountryCode:  "US",
	Location:     "Home",
	Primary:      true,
})
```

## Emails

**[emails.go](../emails.go)**

### `CreateEmail(ctx context.Context, personID string, params *EmailCreateParams) (EmailCreateResponse, error)`

```go
type EmailCreateParams struct {
	Address  string
	Location string
	Primary  bool
}
```

```go
email, err := pco.CreateEmail(ctx, personID, &pco.EmailCreateParams{
	Address:  "ada@example.com",
	Location: "Home",
	Primary:  true,
})
```

## Phone numbers

**[phoneNumbers.go](../phoneNumbers.go)**

### `CreatePhoneNumber(ctx context.Context, personID string, params *PhoneNumberCreateParams) (PhoneNumberCreateResponse, error)`

```go
type PhoneNumberCreateParams struct {
	Number   string
	Location string
	Primary  bool
}
```

```go
phone, err := pco.CreatePhoneNumber(ctx, personID, &pco.PhoneNumberCreateParams{
	Number:   "555-1234",
	Location: "Mobile",
	Primary:  true,
})
```

---

All four `Create*` functions return an error if `params` is `nil`, and a `*pco.RequestError` if PCO rejects the request (see [Errors](../README.md#errors)).
