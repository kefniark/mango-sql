build:
    CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/mangosql ./cmd/mangosql

format:
    golangci-lint run --fix ./...

lint:
    golangci-lint run ./...

docs:
    npm run docs:dev

generate:
    go mod download
    go generate ./tests/...

bench:
    CGO_ENABLED=0 go test -bench=. -benchmem ./tests/bench | tee bench.log
    go run ./cmd/bench/

test:
    go test -race ./...
    go test --cover --coverprofile=coverage.txt ./tests/queries/...
    go tool cover -html=coverage.txt -o coverage.html
    gocover-cobertura < coverage.txt > coverage.xml

update:
    devenv update
    go get -u ./...
    go mod tidy
    npm update
