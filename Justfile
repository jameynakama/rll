set dotenv-load := true

alias mu := migrate-up
alias md := migrate-down

default: test

# Run all tests
test args="":
    gotestsum ./... -- {{ args }}

cover:
    go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

# Start the dev server with hot reload
run:
    air

# Build binary
build:
    go build -o bin/reallylonglink ./cmd/server

# Run pending migrations
migrate-up:
    migrate -path migrations -database "$DATABASE_URL" up

# Roll back one migration
migrate-down:
    migrate -path migrations -database "$DATABASE_URL" down 1

# Regenerate sqlc types after query changes
generate:
    rm -f internal/store/*.sql.go
    sqlc generate
