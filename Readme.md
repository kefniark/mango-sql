# MangoSQL

## Description

MangoSQL is a fresh and juicy SQL code generator.

1. You provide your database schema (.sql files)
2. You run MangoSQL cli to generate a client with type-safe interfaces and queries
3. You write application code that calls the generated code

This is not an ORM, and you can easily inspect the code generated.
This is inspired by [SQLC](https://github.com/sqlc-dev/sqlc) but pushes the idea farther by supporting batching, relations and dynamic queries.

## Features

* **Convenient**: All the structs are generated for you, No need for manual DTO/PDO
* **Time Saver**: All the basic queries (CRUD) are generated from your schema alone, less queries to write
* **Developer Friendly**: The code generated contains comments, examples and is designed with IDE autocompletion in mind 
* **Flexible**: Provide a way to run dynamic queries (pagination, search, ...)
* **Composable**: Use auto-generated query filters or make your owns and reuse them across queries
* **Safe**: All the SQL queries use prepared statement to avoid injection
* **Consistent**: Easy to use transaction API to rollback when an error occurs

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
user, err := db.User.Create(database.UserCreate{
    Name: "user1",
    Email: "user1@email.com"
})

// Handle transactions
err := db.Transaction(func(tx *database.DBClient) error {
    // ...
})

// Typed dynamic clauses (filters, pagination, ...) with typed helpers
users, err := db.User.FindMany(
    db.User.Query.Name.Like("%user%"),
    db.User.Query.Amount.MoreThan(0),
    db.User.Query.Limit(20)
)

// Raw dynamic clauses
users, err := db.User.FindMany(func(query SelectBuilder) SelectBuilder {
	return query.Where("name ILIKE $1 OR name ILIKE $2", "%user1%", "%user2%")
})

// Handle Batching
ids, err := db.User.UpsertMany([]database.UserUpdate{
    {Email: "usernew@localhost", Name: "usernew"}, // this entry will be inserted
    {Id: id, Email: "user1-updated", Name: "user1-updated"}, // this entry will be updated
})

// Use Struct Helpers
user, _ := db.User.FindById(id)
user.name = "NewName"
user.Save(db)

// ...

```
