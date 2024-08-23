# MangoSQL 🥭

![GitHub License](https://img.shields.io/github/license/kefniark/mango-sql)
![GitHub Release](https://img.shields.io/github/v/release/kefniark/mango-sql)
![GitHub Release Date](https://img.shields.io/github/release-date/kefniark/mango-sql)

## Description

**MangoSQL** is a fresh and juicy SQL code generator for **Golang**.

1. Provide your database schema and queries (in .sql files)
2. Run MangoSQL cli to generate a client with type-safe interfaces and queries
3. Write application code based on this generated db client

**MangoSQL** is the perfect choice if you don't want an heavy ORM, but don't want to write all the SQL queries by hand like a caveman either.
Originally inspired by [SQLC](https://github.com/sqlc-dev/sqlc), but pushes the idea farther by natively supporting batching, relations and dynamic queries.

**Links**:
🚀 [Getting Started](https://kefniark.github.io/mango-sql/getting-started/) | 💻 [API Reference](https://kefniark.github.io/mango-sql/api/mutations.html) | 📈 [Benchmark](https://kefniark.github.io/mango-sql/bench/bench.html)

## Features

* **Convenient**: All the structs are generated for you, No need to manually write any [DTO/PDO](https://en.wikipedia.org/wiki/Data_transfer_object)
* **Time Saver**: All the basic [CRUD queries](https://en.wikipedia.org/wiki/Create,_read,_update_and_delete) are generated from your schema alone, less queries to write
* **Safe**: All the SQL queries use prepared statement to avoid injection
* **Consistent**: Easy to use transaction API to rollback when an error occurs
* **Fast**: Get the performances of a handmade `sql.go` in an instant


## Example 

So let's see what it means in reality. For the following:
```sql
CREATE TABLE users (
  id          UUID PRIMARY KEY,
  email       VARCHAR(64) NOT NULL,
  name        VARCHAR(64) NOT NULL,
  created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  deleted_at  TIMESTAMP DEFAULT NULL
);
```

Execute the following command to automatically generate a `database/client.go`
```sh
mangosql --output=database schema.sql
```

This is all you need to do, now the client can be used in your code
```go
db := database.New(dbConnection)

// Handle crud operation
user, err := db.User.Insert(database.UserCreate{
    Name: "user1",
    Email: "user1@email.com"
})

// Typed dynamic clauses (filters, pagination, ...) with typed helpers
users, err := db.User.FindMany(
    db.User.Query.Name.Like("%user%"),
    db.User.Query.Limit(20)
)

// Raw dynamic clauses
users, err := db.User.FindMany(func(query SelectBuilder) SelectBuilder {
	return query.Where("name ILIKE $1 OR name ILIKE $2", "%user1%", "%user2%")
})

// To know more about MangoSQL APIs ... RTFM ^^

```
