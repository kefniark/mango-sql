# Custom Queries

One of the key features or Mango SQL is the ability to write your own queries.

Unlike most ORM or client RAW queries, these will be like the rest of MangoSQL fully typed and prepared

## queries.sql
Create a `queries.sql` next to your schema, and here you can write your queries and name them to more easily recognize them in your codebase.

```sql [queries.sql]
-- queryMany: UserNotDeleted
SELECT *
FROM users
WHERE users.deleted_at IS NULL;
```

## Usage

All the custom queries can be found under `db.Queries.*`

```go
users, err := db.Queries.UserNotDeleted()
```

And these custom queries also accept filters, so the `WHERE` condition can be changed dynamically at runtime.

```go
users, err := db.Queries.UserNotDeleted(
    db.User.Query.Name.NotLike("%user3%"),
)
```