build:
    CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/mangosql ./cmd/mangosql

format:
    golangci-lint run --fix ./...

lint:
    golangci-lint run ./...

generate:
    # tests
    go run ./cmd/mangosql/ --output ./tests/postgres/auto-increment/client.go --package autoincrement ./tests/postgres/auto-increment/schema.sql
    go run ./cmd/mangosql/ --output ./tests/postgres/composite/client.go --package composite ./tests/postgres/composite/schema.sql
    go run ./cmd/mangosql/ --output ./tests/postgres/types/client.go --package types ./tests/postgres/types/schema.sql

    # queries
    go run ./cmd/mangosql/ --output ./tests/queries/pq/client.go --package pq --driver pq ./tests/queries/sqlited/schema.sql
    go run ./cmd/mangosql/ --output ./tests/queries/pgx/client.go --package pgx ./tests/queries/sqlited/schema.sql
    go run ./cmd/mangosql/ --output ./tests/queries/sqlited/client.go --package sqlited --driver sqlite ./tests/queries/sqlited/schema.sql

    # bench
    mkdir -p ./tests/bench/pq
    mkdir -p ./tests/bench/pgx
    mkdir -p ./tests/bench/sqlite
    go run ./cmd/mangosql/ --output ./tests/bench/pq/client.go --package pq --driver pq ./tests/bench/schema.sql
    go run ./cmd/mangosql/ --output ./tests/bench/pgx/client.go --package pgx ./tests/bench/schema.sql
    go run ./cmd/mangosql/ --output ./tests/bench/sqlite/client.go --package pq --driver sqlite ./tests/bench/schema.sqlite.sql

bench:
    CGO_ENABLED=0 go test -bench=. -benchmem ./tests/bench | tee bench.log

test: generate
    go test -race --cover --coverprofile=coverage.txt ./...
    go tool cover -html=coverage.txt -o coverage.html
    gocover-cobertura < coverage.txt > coverage.xml

update:
    devenv update
    go get -u ./...
    go mod tidy
