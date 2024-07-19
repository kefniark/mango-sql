build:
    go build -ldflags="-s -w" -o bin/mangosql ./cmd/mangosql

format:
    golangci-lint run --fix

lint:
    golangci-lint run

generate:
    go run ./cmd/mangosql/ --output codegen/postgres/client.go --package postgres ./codegen/postgres/schema.sql

test: generate
    go test --cover --coverprofile=coverage.txt ./...
    go tool cover -html=coverage.txt -o coverage.html
    gocover-cobertura < coverage.txt > coverage.xml

