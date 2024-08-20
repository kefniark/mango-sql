# Database Client Queries

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

```go [Generated Queries]
db.User.Count(filters ...WhereCondition) (int, error)
db.User.FindMany(filters ...WhereCondition) ([]UserModel, error)
db.User.FindUnique(filters ...WhereCondition) (*UserModel, error)
db.User.FindById(id UserPrimaryKey) (*UserModel, error)
```

:::

## Count

```go
// Count all users
count, err := db.User.Count()

// Use a mango filter and count only users not soft deleted
count, err := db.User.Count(
    db.User.Query.DeletedAt.IsNull(),
)
```

## FindMany

```go
// Get all users
users, err := db.User.FindMany()

// Use a mango filter to paginate users
users, err := db.User.FindMany(
   db.User.Query.Offset(25),
   db.User.Query.Limit(10),
)
```

## FindUnique

```go
// Get the first user which match mango filters
user, err := db.User.FindUnique(
    db.User.Query.Id.Equal(2)
)
```
