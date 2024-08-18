# Automatic Date

## Soft Delete

This feature is automatic, if mangoSQL detect a field named:
* `deletedAt` or `deleted_at`

It will automatically generate a setter method for it:

```go
err = db.User.DeleteSoft(3)
```

## Created At / Updated At

If mango sql detect fields named:
* `created_at` / `createdAt`
* `updated_at` / `updatedAt`

without SQL default values, it will try to automatically populate them when Insert/Update/Upsert mutations are executed.

### Example

::: code-group

```sql [Schema]
CREATE TABLE users (
  id          INTEGER PRIMARY KEY,
  name        VARCHAR(64) NOT NULL,
  created_at  TIMESTAMP NOT NULL,
  updated_at  TIMESTAMP NOT NULL,
  deleted_at  TIMESTAMP
);
```

```sql [SQL Query Generated]
-- insert
INSERT INTO users (id, name, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())

-- update
UPDATE users SET name=$2, updated_at=NOW() WHERE id=$1
```

:::

In this case, the fields are not exposed through `UserCreate` / `UserUpdate` and are automatically set in the SQL functions.