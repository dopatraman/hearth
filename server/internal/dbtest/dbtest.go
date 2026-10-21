// Package dbtest is the per-test-transaction harness used across the server's
// test suite. Any test that touches the DB gets a transaction that rolls back
// on cleanup — no mocks, no fixtures to tear down.
//
// The underlying database is SQLite via modernc.org/sqlite (pure Go, no CGo).
// SQLite serializes writers; tests that mutate state should not run in
// parallel.
package dbtest

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dopatraman/hearth/server/migrations"

	_ "modernc.org/sqlite"
)

var (
	db     *sql.DB
	dbOnce sync.Once
	dbErr  error
)

func initDB() {
	// Default to an in-memory DB that is shared across connections within
	// this test binary but completely isolated from other test binaries
	// (different packages). Prevents SQLITE_BUSY contention when `go test`
	// runs packages in parallel.
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "file::memory:?cache=shared"
	}
	d, err := sql.Open("sqlite", url)
	if err != nil {
		dbErr = err
		return
	}
	// Enforce foreign keys and use WAL for better concurrency.
	for _, pragma := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	} {
		if _, err := d.Exec(pragma); err != nil {
			dbErr = err
			return
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.PingContext(ctx); err != nil {
		dbErr = err
		return
	}

	// Apply all migrations once at pool init. Each test then runs in a
	// transaction that rolls back, so the migrated schema persists but
	// test-written rows do not.
	migCtx, cancelMig := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelMig()
	if err := migrations.Up(migCtx, d); err != nil {
		dbErr = err
		return
	}

	db = d
}

// WithTx opens a transaction, hands it to fn, and rolls back on completion.
// Any state fn writes inside the transaction is gone when the test ends.
//
// Tests MUST route all database interaction through tx. Opening a second
// connection would bypass the rollback and leak state.
func WithTx(t testing.TB, fn func(ctx context.Context, tx *sql.Tx)) {
	t.Helper()
	dbOnce.Do(initDB)
	if dbErr != nil {
		t.Fatalf("dbtest: init failed: %v", dbErr)
	}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("dbtest: begin: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	fn(ctx, tx)
}
