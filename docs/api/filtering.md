# Filtering and Sorting

## MangoSQL Filters

All the Queries have one thing in common: MangoSQL Filters.

MangoSQL statically compiles queries ahead of time, so the shape and typing of the queries are immutable. But for convenience, some clause like `WHERE`, `LIMIT`, `OFFSET` can be dynamically modified at runtime.

Filters provide a Typed and Safe way to modify these clauses and we use [Squirrel](https://github.com/Masterminds/squirrel) under the hood.

```go
// All the Select Queries accepts 0..n filters
db.User.Count(filters...)
db.User.FindMany(filters...)
db.User.FindUnique(filters...)
```

## Sorting

This is also managed though MangoSQL filters

```go
// generate `ORDER BY id ASC, name DESC`
users, err := db.User.FindMany(
    db.User.Query.Id.OrderAsc(),
    db.User.Query.Name.OrderDesc(),
)
```
## Pagination

This is also managed though MangoSQL filters

```go
// generate `OFFSET 10 LIMIT 25`
users, err := db.User.FindMany(
    db.User.Query.Limit(25),
    db.User.Query.Offset(10),
)
```

## Auto-Generated Filters

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
db.{Table}.Query.{Field}.OrderAsc()
db.{Table}.Query.{Field}.OrderDesc()
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

## User Filters

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

::: info

Even if this looks dynamic, each request will automatically turn into a prepared statement and arguments passed separately.

:::
::: danger

Be careful to use `?` to prepare parameters and not concatenate them into the query directly, you could have SQL Injection.

:::