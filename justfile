build:
    CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/mangosql ./cmd/mangosql

format:
    golangci-lint run --fix ./...

lint:
    golangci-lint run ./...

docs:
    npm run docs:dev

generate:
    # tests
    go run ./cmd/mangosql/ --output ./tests/postgres/auto-increment/client.go --package autoincrement --logger console ./tests/postgres/auto-increment/schema.sql
    go run ./cmd/mangosql/ --output ./tests/postgres/composite/client.go --package composite --logger console ./tests/postgres/composite/schema.sql
    go run ./cmd/mangosql/ --output ./tests/postgres/types/client.go --package types --logger console ./tests/postgres/types/schema.sql

    # test loggers
    go run ./cmd/mangosql/ --output ./tests/logger/zap-logger/client.go --package zaplogger --logger zap ./tests/logger/zap-logger/schema.sql
    go run ./cmd/mangosql/ --output ./tests/logger/logrus-logger/client.go --package logruslogger --logger logrus ./tests/logger/logrus-logger/schema.sql
    go run ./cmd/mangosql/ --output ./tests/logger/zerolog-logger/client.go --package zerologlogger --logger zerolog ./tests/logger/zerolog-logger/schema.sql

    # test queries
    go run ./cmd/mangosql/ --output ./tests/queries/pq/client.go --package pq --driver pq --logger console ./tests/queries/sqlited/schema.sql
    go run ./cmd/mangosql/ --output ./tests/queries/pgx/client.go --package pgx --logger console ./tests/queries/sqlited/schema.sql
    go run ./cmd/mangosql/ --output ./tests/queries/sqlited/client.go --package sqlited --driver sqlite --logger console ./tests/queries/sqlited/schema.sql

    # bench
    mkdir -p ./tests/bench/pq
    mkdir -p ./tests/bench/pgx
    mkdir -p ./tests/bench/sqlite
    go run ./cmd/mangosql/ --output ./tests/bench/pq/client.go --package pq --driver pq ./tests/bench/schema.sql
    go run ./cmd/mangosql/ --output ./tests/bench/pgx/client.go --package pgx ./tests/bench/schema.sql
    go run ./cmd/mangosql/ --output ./tests/bench/sqlite/client.go --package pq --driver sqlite ./tests/bench/schema.sqlite.sql

bench:
    CGO_ENABLED=0 go test -bench=. -benchmem ./tests/bench | tee bench.log
    go run ./cmd/bench/

test: generate
    go test -race --cover --coverprofile=coverage.txt ./...
    go tool cover -html=coverage.txt -o coverage.html
    gocover-cobertura < coverage.txt > coverage.xml

update:
    devenv update
    go get -u ./...
    go mod tidy
