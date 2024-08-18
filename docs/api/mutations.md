# Database Client Mutations

::: info

For the following page, we will take as an example the following postgres schema. The usage should be similar regardless of the driver or settings.

::: code-group

```sql [Schema]
CREATE TABLE users (
  id          UUID PRIMARY KEY,
  name        VARCHAR(64) NOT NULL,
  email       VARCHAR(64) NOT NULL,
  created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  deleted_at  TIMESTAMP
);
```

```go [Generated Structs]
// Base Model Returned by most API
type UserModel struct {
	Id        uuid.UUID  `json:"id" db:"id"`
	Email     string     `json:"email" db:"email"`
	Name      string     `json:"name" db:"name"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at"`
}

// Input struct used only for creation
type UserCreate struct {
	Email string `json:"email" db:"email"`
	Name  string `json:"name" db:"name"`
}

// Input struct used only for update
type UserUpdate struct {
	Id    uuid.UUID `json:"id" db:"id"`
	Email string    `json:"email" db:"email"`
	Name  string    `json:"name" db:"name"`
}
```

```go [Generated Mutations]
// Insert
db.User.Insert(input UserCreate) (*UserModel, error)
db.User.InsertMany(inputs []UserCreate) ([]UserPrimaryKeySerialized, error)

// Update
db.User.Update(input UserUpdate) (*UserModel, error)
db.User.UpdateMany(inputs []UserUpdate) ([]UserPrimaryKeySerialized, error)

// Upsert
db.User.Upsert(input UserUpdate) (*UserModel, error)
db.User.UpsertMany(inputs []UserUpdate) ([]UserPrimaryKeySerialized, error)

// Delete
db.User.DeleteSoft(id UserPrimaryKey) error
db.User.DeleteHard(id UserPrimaryKey)
```

:::

## Insert

```go
user, err := db.User.Insert(database.UserCreate{
    Name: "John Doe",
    Email: "john@email.com",
})
```

::: tip

If you have more than one entry to add to database, you also have a bulk insert alternative which takes a slices of input

```go
userIds, err := db.User.InsertMany([]database.UserCreate{
    // ... many users
})
```

:::

## Update

```go
user, err := db.User.Update(database.UserUpdate{
    Id: "00000000-0000-0000-0000-000000000000",
    Name: "John Doe 2",
    Email: "john@email.com",
})
```

::: tip

If you have more than one entry to add to database, you also have a bulk update alternative which takes a slices of input

```go
userIds, err := db.User.UpdateMany([]database.UserUpdate{
    // ... many users
})
```

:::

## Upsert

Upsert stands for Insert in database if the entry doesn't exist yet, or update the existing entry. In both cases, the entry is returned.

This is really convenient for operations like user creation or session, when the api doesn't know if the user already exists in the database or not.

```go
user, err := db.User.Upsert(database.UserUpdate{
    Id: "00000000-0000-0000-0000-000000000000",
    Name: "John Doe 2",
    Email: "john@email.com",
})
```

::: tip

If you have more than one entry to add to database, you also have a bulk upsert alternative which takes a slices of input

```go
userIds, err := db.User.UpsertMany([]database.UserUpdate{
    // ... many users
})
```

:::

## Delete

```go
// Default SQL Delete
err = db.User.DeleteHard(3)
```

And if you are using Soft Delete, a second method will we automatically generated

```go
// Soft Delete backed by a Timestamp field
err = db.User.DeleteSoft(3)
```
