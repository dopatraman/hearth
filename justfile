export DATABASE_URL := env_var_or_default("DATABASE_URL", "postgres://hearth:hearth_dev@localhost:5433/hearth?sslmode=disable")
export PATH := env_var("HOME") / "go/bin" + ":" + env_var("PATH")

default:
    @just --list

# Bring up Postgres, apply migrations, run the server.
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

# Bring up Postgres in the background and wait for it to be ready.
db-up:
    docker compose -f deploy/docker-compose.yml up -d
    @echo "waiting for postgres..."
    @until docker exec hearth-postgres pg_isready -U hearth >/dev/null 2>&1; do sleep 0.5; done
    @echo "postgres ready"

# Stop Postgres (data persists in the named volume).
db-down:
    docker compose -f deploy/docker-compose.yml down

# Nuke Postgres data and re-create.
db-reset:
    docker compose -f deploy/docker-compose.yml down -v
    just db-up
    just migrate
