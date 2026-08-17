# People

Wraps the [People v2 API](https://api.planningcenteronline.com/docs/apps/people): people records and three of their sub-resources (addresses, emails, phone numbers). All paths are rooted at `people/v2/people`.

## People

**[people.go](../people.go)**

### `GetPeople(params *PeopleParams) (PersonListResponse, error)`

```go
type PeopleParams struct {
	FirstName         string
	LastName          string
	Email             string // matched against pco's search_name_or_email filter
	SearchPhoneNumber string
	PerPage           int
	Offset            int
}
```

`params` may be `nil` to list everyone (paginated). Any non-empty field adds a `where[]` filter; `PerPage`/`Offset` control pagination (see [Pagination](../README.md#pagination) in the root README).

```go
people, err := pco.GetPeople(&pco.PeopleParams{LastName: "Lovelace"})
```

### `CreatePerson(params *CreatePersonParams) (PersonCreateResponse, error)`

```go
type CreatePersonParams struct {
	FirstName string
	LastName  string
}
```

`params` must not be `nil`.

```go
person, err := pco.CreatePerson(&pco.CreatePersonParams{FirstName: "Ada", LastName: "Lovelace"})
personID := person.Data.ID
```

### `PersonAttributes`

The full set of attributes PCO returns for a person (name fields, `Status`, `Birthdate`, `Membership`, permission flags, `ResourcePermissionFlags`, timestamps, etc.) — see [people.go](../people.go) for the complete list. `PersonRelationships` currently exposes `PrimaryCampus` and `Gender`.

## Addresses

**[addresses.go](../addresses.go)**

### `CreateAddress(personID string, params *AddressCreateParams) (AddressCreateResponse, error)`

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
address, err := pco.CreateAddress(personID, &pco.AddressCreateParams{
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

### `CreateEmail(personID string, params *EmailCreateParams) (EmailCreateResponse, error)`

```go
type EmailCreateParams struct {
	Address  string
	Location string
	Primary  bool
}
```

```go
email, err := pco.CreateEmail(personID, &pco.EmailCreateParams{
	Address:  "ada@example.com",
	Location: "Home",
	Primary:  true,
})
```

## Phone numbers

**[phoneNumbers.go](../phoneNumbers.go)**

### `CreatePhoneNumber(personID string, params *PhoneNumberCreateParams) (PhoneNumberCreateResponse, error)`

```go
type PhoneNumberCreateParams struct {
	Number   string
	Location string
	Primary  bool
}
```

```go
phone, err := pco.CreatePhoneNumber(personID, &pco.PhoneNumberCreateParams{
	Number:   "555-1234",
	Location: "Mobile",
	Primary:  true,
})
```

---

All four `Create*` functions return an error if `params` is `nil`, and a `*pco.RequestError` if PCO rejects the request (see [Errors](../README.md#errors)).
