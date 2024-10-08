# Getting Started

Starting using MangoSQL is quite straightforward, it's a standalone CLI and there is no runtime overhead 🥭

## CLI Installation

::: code-group

```sh [go]
# Install with go command
go install github.com/kefniark/mango-sql/cmd/mangosql

# Use MangoSQL CLI
mangosql schema.sql
```

```sh [manual]
# Download the last release from https://github.com/kefniark/mango-sql/releases

# Use MangoSQL CLI
./mangosql schema.sql
```

```sh [docker]
# Download from docker registry: https://github.com/kefniark/mango-sql/pkgs/container/mango-sql

# Use MangoSQL CLI
docker run -t --rm -v .:/app/ \
  ghcr.io/kefniark/mango-sql:latest \
  -i /app/schema.sql > client.go

# -i/--inline: allow to output the generated file to the console
```

:::

## CLI Usage

::: code-group

```sh [postgres]
# Default command (output: ./database/client.go)
mangosql ./schema.sql

# Or: Output in a specific folder
mangosql --output ./myfolder --package myfolder ./schema.sql
```

```sh [sqlite]
# Default command (output: ./database/client.go)
mangosql --driver sqlite ./schema.sql

# Or: Output in a specific folder
mangosql --output ./myfolder --package myfolder --driver sqlite ./schema.sql
```

```sh [mariadb]
# Default command (output: ./database/client.go)
mangosql --driver mariadb ./schema.sql

# Or: Output in a specific folder
mangosql --output ./myfolder --package myfolder --driver mariadb ./schema.sql
```

```sh [mysql]
# Default command (output: ./database/client.go)
mangosql --driver mysql ./schema.sql

# Or: Output in a specific folder
mangosql --output ./myfolder --package myfolder --driver mysql ./schema.sql
```

:::

* `./schema.sql`: Input schema, accept a SQL file or a directory of migrations
* `--output`: Output where the golang code generated will be written. Accept a file path or a directory
* `--package`: Go Package to use in the generated code (by default `database`)
* `--driver`: Name of the golang package to use, by default `database`

::: tip
The Recommended folder structure is to have a dedicated Go package with both schema and generated code
```sh [folder structure]
database/
  * schema.sql # the schema
  * client.go # the generated client
  * main.go # code to instantiate database connection and create a db client
```
This will make later import easier and more readable
```go
import "{your_project_url}/database"

```
:::

::: tip
Mangosql can be combined with go:generate (ref: https://go.dev/blog/generate)

```go
//go:generate mangosql --output ./client.go --driver mysql ./schema.sql
```

This allow to keep code generation and usage really close to each other
:::