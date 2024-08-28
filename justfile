build:
    CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/mangosql ./cmd/mangosql

format:
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3
    golangci-lint run --fix ./...

lint:
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3
    golangci-lint run ./...

docs:
    npm run docs:dev

generate:
    go mod download
    go generate ./tests/...

generate_docs:
    go run ./cmd/mangosql/ diagram --output ./docs/public/blog.svg ./tests/diagram/blog.sql
    go run ./cmd/mangosql/ diagram --output ./docs/public/blog_dark.svg -s -d -t "My wonderful Blog" -m "Version: 1.0.2" ./tests/diagram/blog.sql
    go run ./cmd/mangosql/ diagram --output ./docs/public/blog_simple.svg -s -t "" ./tests/diagram/blog.sql

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
