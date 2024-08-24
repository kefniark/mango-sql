# Client Usage

## Instantiate Database Client

When your Application starts, you will most likely have to initialize your database connection.

You can find below examples for the most commons usages, in the rest of your application you can keep reference to `db database.DBClient`

::: code-group

```go [postgres]
import (
    "context"

    "github.com/jackc/pgx/v5/pgxpool"
)

// TODO: Update Postgres connection url
const databaseURL := "{CONNECTION URL TO YOUR DATABASE}"

func NewDBClient() (db *DBClient, close func()) {
    db, err := pgxpool.New(context.Background(), databaseURL)
    if err != nil {
        panic(err)
    }

    return New(db), func() {
		db.Close()
	}
}
```

```go [sqlite]
import (
    "context"

    "github.com/jmoiron/sqlx"
    _ "modernc.org/sqlite"
)

// TODO: Update sqlite connection url
// * `:memory:` can be used for non persistent DB
// * `file.db` can be used to write to file
const databaseURL := "{CONNECTION URL TO YOUR DATABASE}"

func NewDBClient() (db *DBClient, close func()) {
    db, err := sqlx.Open("sqlite", databaseURL)
	if err != nil {
		panic(err)
	}

    return New(db), func() {
		db.Close()
	}
}
```

```go [mariadb/mysql]
import (
    "context"

    "github.com/jmoiron/sqlx"
    _ "github.com/go-sql-driver/mysql"
)

// TODO: Update mysql/mariadb connection url
// Ref: https://github.com/go-sql-driver/mysql
// example: user:password@tcp(127.0.0.1:3306)/dbname?parseTime=true
const databaseURL := "{CONNECTION URL TO YOUR DATABASE}"

func NewDBClient() (db *DBClient, close func()) {
    db, err := sqlx.Open("mysql", databaseURL)
	if err != nil {
		panic(err)
	}

    return New(db), func() {
		db.Close()
	}
}
```

:::

## Enjoy MangoSQL

You are done with the setup, you can now use the Generated Client in your code

```go
db, close := NewDBClient()
defer close()

user, err := db.User.FindById(1)
// ...
```