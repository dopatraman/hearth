export DATABASE_URL := env_var_or_default("DATABASE_URL", "./hearth.dev.db")
export SQLITE_PATH := env_var_or_default("SQLITE_PATH", "./hearth.dev.db")
export SESSION_SECRET := env_var_or_default("SESSION_SECRET", "development-session-secret-NOT-FOR-PRODUCTION-ever!!")
export APPLE_AUDIENCE := env_var_or_default("APPLE_AUDIENCE", "com.dopatraman.hearth")
export PATH := env_var("HOME") / "go/bin" + ":" + env_var("PATH")

default:
    @just --list

# Apply migrations, then run the server.
dev: migrate run

# Run the server.
run:
    go -C server run ./cmd/hearth

# Run all Go tests.
test:
    @if ls server/migrations/*.sql >/dev/null 2>&1; then just migrate; fi
    go -C server test ./...

# Apply pending migrations.
migrate:
    @if ls server/migrations/*.sql >/dev/null 2>&1; then \
        goose -dir server/migrations sqlite3 "$DATABASE_URL" up; \
    else \
        echo "no migrations yet"; \
    fi

# Roll back all migrations.
migrate-reset:
    goose -dir server/migrations sqlite3 "$DATABASE_URL" reset

# Regenerate sqlc Go from queries/*.sql and migrations/*.sql.
sqlc:
    cd server && sqlc generate

# Delete the dev database file.
db-reset:
    rm -f server/hearth.dev.db server/hearth.dev.db-shm server/hearth.dev.db-wal hearth.dev.db hearth.dev.db-shm hearth.dev.db-wal
    just migrate
