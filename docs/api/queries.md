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


## MangoSQL Filters

All the following Queries have one thing in common: MangoSQL Filters.

MangoSQL statically compiles them ahead of time, so the shape and typing of the queries are immutable. But for convenience, some clause like `WHERE`, `LIMIT`, `OFFSET` can be dynamically modified at runtime.

Filters provide a Type and Safe way to modify these clauses and use [Squirrel](https://github.com/Masterminds/squirrel) under the hood.

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

## Filters Details

### Auto-Generated Filters

For each field of your table, a set of filters will be automatically generated based on the Type. This covers the most common operations.

```txt
db.{Table}.Query.{Field}.Equal(input)
db.{Table}.Query.{Field}.NotEqual(input)
db.{Table}.Query.{Field}.In(input)
db.{Table}.Query.{Field}.NotIn(input)
db.{Table}.Query.{Field}.Like(input)
db.{Table}.Query.{Field}.MoreThan(input)
db.{Table}.Query.{Field}.LessThan(input)
db.{Table}.Query.{Field}.Between(low, high)
db.{Table}.Query.{Field}.OrderByAsc()
db.{Table}.Query.{Field}.OrderByDesc()
db.{Table}.Query.Offset(offset)
db.{Table}.Query.Limit(limit)
```

#### Example

::: code-group

```go [Mango Filter Usage]
users, err := db.User.FindMany(
    db.User.Query.Name.In("user1", "user2"),
    db.User.Query.Id.LesserThan(10),
    db.User.Query.Id.OrderAsc(),
    db.User.Query.Offset(25),
    db.User.Query.Limit(10),
)
```

```sql [Prepared SQL Statement]
SELECT id, name, created_at, deleted_at
FROM users
WHERE users.name = ANY($1) AND users.id < $2
ORDER BY users.id ASC
LIMIT 10 OFFSET 25
```

:::

### User Filters

You can also write your own filters, a filter is just a function which takes and returns a QueryBuilder.
This gives way more freedom for advanced field manipulations, specific database syntax or extensions, ...

::: code-group

```go [Mango Filter]
// This is a valid mango filter
func(cond SelectBuilder) SelectBuilder {
    return cond.Where("name = ? OR id = ?", "user1", 2)
}
```

```go [Find Usage]
// find all users which match this filter
users, err := db.User.FindMany(func(cond SelectBuilder) SelectBuilder {
    return cond.Where("name = ? OR id = ?", "user1", 2)
})
```

```go [Composable]
// create a new function to generate these where conditions
func MyFilter(name string, id int) WhereCondition {
	return func(cond SelectBuilder) SelectBuilder {
		return cond.Where("name = ? OR id = ?", name, id)
	}
}

// can use the filter in any User related query
users, err := db.User.FindMany(
    MyFilter("user1", 2),
)
```

:::

::: tip

Even if this looks dynamic, each request will automatically turn into a prepared statement and arguments passed separately.

Be careful to use `?` to use parameters and not concatenate them into the request.

:::