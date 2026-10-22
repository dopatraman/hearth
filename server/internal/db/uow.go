package db

import (
	"context"
	"database/sql"
)

// UoW ("unit of work") lets handlers run multiple sqlc queries atomically
// without having to pass *sql.DB vs *sql.Tx around. In production, InTx opens
// a real transaction; in tests, the test harness supplies a UoW that reuses
// the outer per-test transaction (SQLite doesn't nest cleanly, and dbtest
// already provides rollback-at-end-of-test).
type UoW interface {
	// InTx runs fn against transactional Queries. Returned error from fn
	// triggers a rollback; otherwise commit.
	InTx(ctx context.Context, fn func(*Queries) error) error

	// Queries returns non-transactional Queries, useful for single-statement
	// handlers that don't need multi-statement atomicity.
	Queries() *Queries
}

// PoolUoW is the production impl. Each InTx opens a fresh real transaction
// on the pool.
type PoolUoW struct {
	Pool *sql.DB
}

func (p *PoolUoW) InTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := p.Pool.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(New(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (p *PoolUoW) Queries() *Queries {
	return New(p.Pool)
}

// TxUoW is the test impl. Both InTx and Queries resolve to the single wrapped
// *sql.Tx (no sub-transactions). Rollback is the outer test harness's job.
type TxUoW struct {
	Tx *sql.Tx
}

func (t *TxUoW) InTx(ctx context.Context, fn func(*Queries) error) error {
	return fn(New(t.Tx))
}

func (t *TxUoW) Queries() *Queries {
	return New(t.Tx)
}
