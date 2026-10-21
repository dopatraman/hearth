// Package migrations ships the SQL migration files to the running binary
// via go:embed and runs them via goose on startup.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var FS embed.FS

// Up applies all pending migrations against db.
func Up(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(FS)
	defer goose.SetBaseFS(nil)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("up: %w", err)
	}
	return nil
}
