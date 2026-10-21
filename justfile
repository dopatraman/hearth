export DATABASE_URL := env_var_or_default("DATABASE_URL", "postgres://hearth@localhost:5432/hearth?sslmode=disable")
export PATH := env_var("HOME") / "go/bin" + ":" + "/opt/homebrew/opt/postgresql@17/bin" + ":" + env_var("PATH")

default:
    @just --list

# Start Postgres, apply migrations, run the server.
dev: db-up migrate run

# Run the server (expects Postgres already up and migrated).
run:
    go -C server run ./cmd/hearth

# Run all Go tests.
test:
    go -C server test ./...

# Apply pending migrations.
migrate:
    goose -dir server/migrations postgres "$DATABASE_URL" up

# Roll back all migrations.
migrate-reset:
    goose -dir server/migrations postgres "$DATABASE_URL" reset

# Regenerate sqlc Go from queries/*.sql and migrations/*.sql.
sqlc:
    cd server && sqlc generate

# Start Postgres (via Homebrew services), create role and db if missing.
db-up:
    brew services start postgresql@17
    @echo "waiting for postgres..."
    @until pg_isready -h localhost >/dev/null 2>&1; do sleep 0.5; done
    @createuser -h localhost -s hearth 2>/dev/null || true
    @createdb -h localhost -O hearth hearth 2>/dev/null || true
    @echo "postgres ready"

# Stop Postgres.
db-down:
    brew services stop postgresql@17

# Drop and recreate the hearth database, then re-apply migrations.
db-reset:
    dropdb -h localhost --if-exists hearth
    createdb -h localhost -O hearth hearth
    just migrate
