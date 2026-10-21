package dbtest_test

import (
	"context"
	"testing"

	"github.com/dopatraman/hearth/server/internal/dbtest"
	"github.com/jackc/pgx/v5"
)

// Writes made inside the WithTx closure are visible to subsequent statements
// within the same transaction.
func TestWithTx_WritesVisibleWithinTx(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx pgx.Tx) {
		if _, err := tx.Exec(ctx, `CREATE TEMPORARY TABLE t (x int) ON COMMIT DROP`); err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO t (x) VALUES (1), (2), (3)`); err != nil {
			t.Fatalf("insert: %v", err)
		}
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM t`).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		if n != 3 {
			t.Errorf("count: got %d, want 3", n)
		}
	})
}

// Writes do not persist across WithTx calls — rollback happens in t.Cleanup.
// We prove it by creating a non-temp table in one subtest and expecting the
// next subtest to be able to create the same name; only possible if the first
// was rolled back before the second runs.
func TestWithTx_RollsBackBetweenRuns(t *testing.T) {
	t.Run("creates_table", func(t *testing.T) {
		dbtest.WithTx(t, func(ctx context.Context, tx pgx.Tx) {
			if _, err := tx.Exec(ctx, `CREATE TABLE harness_rollback_check (x int)`); err != nil {
				t.Fatalf("first create: %v", err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO harness_rollback_check (x) VALUES (1)`); err != nil {
				t.Fatalf("insert: %v", err)
			}
		})
	})

	t.Run("table_gone_after_rollback", func(t *testing.T) {
		dbtest.WithTx(t, func(ctx context.Context, tx pgx.Tx) {
			if _, err := tx.Exec(ctx, `CREATE TABLE harness_rollback_check (x int)`); err != nil {
				t.Fatalf("second create should succeed (first was rolled back), got: %v", err)
			}
		})
	})
}
