# MangoSQL

## Description

MangoSQL is a fresh and juicy SQL code generator.

1. You provide your database schema (.sql files)
2. You run MangoSQL cli to generate a client with type-safe interfaces and queries
3. You write application code that calls the generated code

This is not an ORM, and you can easily inspect the code generated.
This is inspired by [SQLC](https://github.com/sqlc-dev/sqlc) but pushes the idea farther by supporting relations and dynamic queries.

## Features

* **Convenient**: Generate all the structs for you
* **Flexible**: Provide a way to run dynamic queries (pagination, search, ...)
* **Time Saver**: All the basic queries (CRUD) are auto-generated from your schema, nothing to declare
* **Performant**: Support batching out of the box
* **Safe**: All the SQL queries use prepared statement
* **Consistency**: Easy to use transaction API

## Status

This is WIP, features are still not complete and may change

Goals:
* Handle custom relations (join, aggregations, ...)
* Handle views
* Support Mysql/MariaDB/Sqlite3
* Better CLI

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

// Handle dynamic clauses (filters, pagination, ...)
users, err := db.User.Where(func(cond database.SelectBuilder) database.SelectBuilder {
    return cond.Where("name ILIKE $1", "%user%").Offset(0).Limit(20)
})

// Handle Batching
ids, err := db.User.UpsertMany([]database.UserUpdate{
    {Email: "usernew@localhost", Name: "usernew"}, // this entry will be inserted
    {Id: id, Email: "user1-updated", Name: "user1-updated"}, // this entry will be updated
})

// ...
```

## API

Here is the list of all the auto-generated methods for your tables:

## Getters
* db.{Table}.Count()
* db.{Table}.CountWhere(condition)
* db.{Table}.All(offset, limit)
* db.{Table}.Where(condition)
* db.{Table}.GetById(id)

## Mutations
* db.{Table}.Create(input)
* db.{Table}.Update(input)
* db.{Table}.Upsert(input)
* db.{Table}.DeleteSoft(id)
* db.{Table}.DeleteHard(id)

## Batch Mutations
* db.{Table}.CreateMany(inputs)
* db.{Table}.UpdateMany(inputs)
* db.{Table}.UpsertMany(inputs)
