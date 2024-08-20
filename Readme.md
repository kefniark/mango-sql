# MangoSQL 🥭

![GitHub License](https://img.shields.io/github/license/kefniark/mango-sql)
![GitHub Release](https://img.shields.io/github/v/release/kefniark/mango-sql)
![GitHub Release Date](https://img.shields.io/github/release-date/kefniark/mango-sql)

## Description

**MangoSQL** is a fresh and juicy SQL code generator.

1. You provide your database schema (.sql files)
2. You run MangoSQL cli to generate a client with type-safe interfaces and queries
3. You write application code that calls the generated code

**MangoSQL** is the perfect choice if you don't want an heavy ORM, but don't want to write all the SQL queries by hand like a caveman either.
Originally inspired by [SQLC](https://github.com/sqlc-dev/sqlc), but pushes the idea farther by natively supporting batching, relations and dynamic queries.

## Features

* **Convenient**: All the structs are generated for you, No need to manually write any [DTO/PDO](https://en.wikipedia.org/wiki/Data_transfer_object)
* **Time Saver**: All the basic [CRUD queries](https://en.wikipedia.org/wiki/Create,_read,_update_and_delete) are generated from your schema alone, less queries to write
* **Safe**: All the SQL queries use prepared statement to avoid injection
* **Consistent**: Easy to use transaction API to rollback when an error occurs
* **Fast**: Get the performances of a handmade `sql.go` in an instant

## Links
* [Getting Started](https://kefniark.github.io/mango-sql/getting-started/)
* [](https://kefniark.github.io/mango-sql/getting-started/)
## Getting Started

```sh
# Install MangoSQL CLI
go install github.com/kefniark/mango-sql/cmd/mangosql

# Use mango to generate a DB Client (by default ./database/client.go)
mangosql ./database/schema.sql

# Generated Go Code can be output somewhere else
mangosql --output ./mydb/myclient.go --package mydb ./database/schema.sql
```

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
sqlDb, err := sqlx.Connect("postgres", "Postgres Connection URL")
if err != nil {
    panic(err)
}

db := database.New(sqlDb)

// then you can use it and make queries or transactions

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
