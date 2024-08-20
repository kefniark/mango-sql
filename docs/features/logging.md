# Logging

MangoSQL has integration for common Go loggers, this make working with SQL queries easier.

This logging feature is opt-in and comes with:
* Every Query will emit a **Debug** log
  * With the function name
  * With duration measurements
  * With SQL Query and arguments
* **warning** for slow queries, when a query takes more than >500ms
* Every SQL error will be automatically logged as **error**

::: tips

    During development, we recommend to set the log level to `DebugLevel` to see the SQL queries generated and how long they take.

:::

## Logrus (https://github.com/sirupsen/logrus) { #logrus }

Add `--logger logrus` to the cli command

```sh
mangosql --logger logrus ./schema.sql
```

And provide the logger to MangoSQL Client at initialization.

```go
package database

import (
    // ...
    "github.com/sirupsen/logrus"
)

// create your own logger instance
logger := logrus.New()

// instantiate DBClient
return New(db, logger)
```

## Zap (https://github.com/uber-go/zap) { #zap }

Add `--logger zap` to the cli command

```sh
mangosql --logger zap ./schema.sql
```

And provide the logger to MangoSQL Client at initialization.

```go
package database

import (
    // ...
    "go.uber.org/zap"
)

// create your own logger instance
logger, _ := zap.NewProduction()

// instantiate DBClient
return New(db, logger)
```

## Zerolog (https://github.com/rs/zerolog) { #zerolog }

Add `--logger zerolog` to the cli command

```sh
mangosql --logger zerolog ./schema.sql
```

And provide the logger to MangoSQL Client at initialization.

```go
package database

import (
    // ...
    "github.com/rs/zerolog"
)

// create your own logger instance
logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

// instantiate DBClient
return New(db, logger)
```

## Testing / Console

This logger option is a bit special, this is a console writer intended for development and testing only.
It's literally just calling `log.Println()`, has colors and is zero configuration.

Add `--logger console` to the cli command and you have nothing else to modify in your code.

```sh
mangosql --logger console ./schema.sql
```

The output will looks like this
```logs
2024/08/20 04:32:46 [DEBUG] DB.User.FindMany  308.236µs
   | Args: [[1]]
   | SQL: SELECT id, name, created_at, deleted_at FROM users WHERE id = $1 LIMIT 1 OFFSET 0
```