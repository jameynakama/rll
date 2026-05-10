# reallylonglink

Make links really, really long

## Stack

- **[chi](https://github.com/go-chi/chi)** -- HTTP router
- **[pgx/v5](https://github.com/jackc/pgx)** -- Postgres driver + connection pool
- **[sqlc](https://sqlc.dev)** -- generates type-safe Go from SQL queries
- **[golang-migrate](https://github.com/golang-migrate/migrate)** -- versioned migrations
- **[air](https://github.com/air-verse/air)** -- hot reloading for server
- **[gotestsum](https://github.com/gotestyourself/gotestsumr)** -- better go test output
- **docker-compose** -- local Postgres, no host install needed
- **[just](https://github.com/casey/just)** -- task runner

## Prerequisites

- go
- sqlc
- golang-migrate
- air (go install github.com/air-verse/air@latest)
- Docker Desktop (or equivalent) for Postgres
- (optional) just

## First Run

```bash
git clone https://github.com/jameynakama/go-crud-template my-actual-app-name
cd my-actual-app-name

# 1. Replace all placeholder names throughout with your app's actual name and GitHub username

grep -rli "appname\|GHUSER" . --include="*.go" --include="*.mod" --include="*.yml" --include="*.yaml" --include="*.example" --include="*.sql" | \
  xargs sed -i '' 's/GHUSER/yourgithubusername/g; s/APPNAME/yourappname/g; s/appname/yourappname/g'

# 2. Set up environment

cp .env.example .env # Edit .env for your needs

# 3. Start Postgres

docker compose up -d

# 4. Run migrations

just migrate-up

# 5. Start the server

just run
```

Server starts on `http://localhost:8080` (or whatever `PORT` is set to).

## Commands

| Command                 | Description                             |
| ----------------------- | --------------------------------------- |
| `just`                  | Run tests (default)                     |
| `just run`              | Start dev server                        |
| `just build`            | Build binary to `bin/APPNAME`           |
| `just migrate-up`       | Apply pending migrations                |
| `just migrate-down`     | Roll back one migration                 |
| `just generate`         | Regenerate sqlc types after SQL changes |
| `just migration name=X` | Create a new migration pair             |

## API

```
GET    /health
GET    /api/v1/links?limit=20&offset=0
GET    /api/v1/links/{id}
POST   /api/v1/links
PUT    /api/v1/links/{id}
DELETE /api/v1/links/{id}
```

## Adding a New Resource

1. `just migration name=add_widgets` -- creates `migrations/NNN_add_widgets.{up,down}.sql`
2. Write the schema in the `.up.sql` file; add the `set_update_time` trigger if needed
3. `just migrate-up`
4. Create `internal/store/queries/widgets.sql` with your queries
5. `just generate` -- sqlc emits the Go types and methods
6. Add handler methods to `internal/api/handlers.go`
7. Register routes in `internal/api/router.go`
8. Add tests to `internal/api/handlers_test.go`

## Tests

Integration tests run against a real ephemeral database created and destroyed
each run. Set `TEST_DATABASE_URL` in `.env` pointing at the same Postgres
instance as `DATABASE_URL` -- the test suite handles creating and dropping the
test database automatically.

```bash
just test
```
