build:
    go build -ldflags="-s -w" -o bin/mangosql ./cmd/mangosql

bench:
    go test -bench=. -benchmem ./tests/bench

format:
    golangci-lint run --fix ./...

lint:
    golangci-lint run ./...

generate:
    go run ./cmd/mangosql/ --output ./tests/postgres/overview/client.go --package overview ./tests/postgres/overview/schema.sql

    go run ./cmd/mangosql/ --output ./tests/postgres/auto-increment/client.go --package autoincrement ./tests/postgres/auto-increment/schema.sql
    go run ./cmd/mangosql/ --output ./tests/postgres/composite/client.go --package composite ./tests/postgres/composite/schema.sql
    go run ./cmd/mangosql/ --output ./tests/postgres/types/client.go --package types ./tests/postgres/types/schema.sql

    go run ./cmd/mangosql/ --output ./tests/bench/client.go --package bench ./tests/bench/schema.sql
    #go run ./cmd/mangosql/ --output ./tests/postgres/enum/client.go --package enum ./tests/postgres/enum/schema.sql

test: generate
    go test --cover --coverprofile=coverage.txt ./...
    go tool cover -html=coverage.txt -o coverage.html
    gocover-cobertura < coverage.txt > coverage.xml

