set dotenv-load := true

alias mu := migrate-up
alias md := migrate-down

default: test

# Run tests with nicer output
test args="":
    gotestsum ./... -- {{ args }}

# Run regular tests with certain args that gotestsum seems to ignore
gotest args="":
    go test ./... {{ args }}

# Run tests with coverage report
cover:
    go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

# Start the dev server with hot reload
run:
    air

# Build binary
build:
    go build -o bin/reallylonglink ./cmd/server

# Create a new migration pair
migration name:
    migrate create -ext sql -dir migrations -seq -digits 3 {{ name }}

# Run pending migrations
migrate-up:
    migrate -path migrations -database "$DATABASE_URL" up

# Roll back one migration
migrate-down num="1":
    migrate -path migrations -database "$DATABASE_URL" down {{ num }}

# Regenerate sqlc types after query changes
generate:
    rm -f internal/store/*.sql.go
    sqlc generate
