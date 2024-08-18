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

