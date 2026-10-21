// Package dbtest is the per-test-transaction harness used across the server's
// test suite. Any test that touches Postgres gets a transaction that rolls back
// on cleanup — no mocks, no fixtures to tear down.
package dbtest

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pool     *pgxpool.Pool
	poolOnce sync.Once
	poolErr  error
)

func initPool() {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		poolErr = errors.New("DATABASE_URL is not set (use `just db-up && just migrate` for local dev)")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, err := pgxpool.New(ctx, url)
	if err != nil {
		poolErr = err
		return
	}
	if err := p.Ping(ctx); err != nil {
		poolErr = err
		return
	}
	pool = p
}

// WithTx opens a transaction, hands it to fn, and rolls back on completion.
// Any state fn writes inside the transaction is gone when the test ends.
//
// Tests MUST route all database interaction through tx. Opening a second
// connection (via the pool directly) would bypass the rollback and leak state.
func WithTx(t testing.TB, fn func(ctx context.Context, tx pgx.Tx)) {
	t.Helper()
	poolOnce.Do(initPool)
	if poolErr != nil {
		t.Fatalf("dbtest: pool init failed: %v", poolErr)
	}

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("dbtest: begin: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })

	fn(ctx, tx)
}
