build:
    CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/mangosql ./cmd/mangosql

format:
    golangci-lint run --fix ./...

lint:
    golangci-lint run ./...

generate:
    # tests
    go run ./cmd/mangosql/ --output ./tests/postgres/pgx_driver/client.go --package main ./tests/postgres/pgx_driver/schema.sql
    go run ./cmd/mangosql/ --output ./tests/postgres/overview/client.go --package overview ./tests/postgres/overview/schema.sql
    go run ./cmd/mangosql/ --output ./tests/postgres/auto-increment/client.go --package autoincrement ./tests/postgres/auto-increment/schema.sql
    go run ./cmd/mangosql/ --output ./tests/postgres/composite/client.go --package composite ./tests/postgres/composite/schema.sql

    go run ./cmd/mangosql/ --output ./tests/sqlite/client.go --package sqlite --driver pq ./tests/sqlite/schema.sql
    #go run ./cmd/mangosql/ --output ./tests/postgres/enum/client.go --package enum ./tests/postgres/enum/schema.sql
    go run ./cmd/mangosql/ --output ./tests/postgres/types/client.go --package types ./tests/postgres/types/schema.sql

    # bench
    mkdir -p ./tests/bench/pq
    mkdir -p ./tests/bench/pgx
    go run ./cmd/mangosql/ --output ./tests/bench/pq/client.go --package pq --driver pq ./tests/bench/schema.sql
    go run ./cmd/mangosql/ --output ./tests/bench/pgx/client.go --package pgx ./tests/bench/schema.sql
    go run ./cmd/mangosql/ --output ./tests/bench/sqlite/client.go --package pq --driver pq ./tests/bench/schema.sqlite.sql

bench:
    CGO_ENABLED=0 go test -c -bench=. -benchtime=1s -benchmem ./tests/bench

test: generate
    go test -race --cover --coverprofile=coverage.txt ./...
    go tool cover -html=coverage.txt -o coverage.html
    gocover-cobertura < coverage.txt > coverage.xml

