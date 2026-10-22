package lists_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/dbtest"
	"github.com/dopatraman/hearth/server/internal/lists"
	"github.com/dopatraman/hearth/server/internal/users"
)

func TestEnsurePersonalMembership_CreatesListForNewUser(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)

		u, err := users.NewService(q).UpsertByAppleSub(ctx, "sub-A", nil)
		if err != nil {
			t.Fatalf("upsert user: %v", err)
		}

		listID, err := lists.NewService(q).EnsurePersonalMembership(ctx, u.ID)
		if err != nil {
			t.Fatalf("ensure membership: %v", err)
		}
		if listID == "" {
			t.Fatal("listID empty")
		}

		memberships, err := q.ListMembershipsForUser(ctx, u.ID)
		if err != nil {
			t.Fatalf("list memberships: %v", err)
		}
		if len(memberships) != 1 {
			t.Fatalf("want 1 membership, got %d", len(memberships))
		}
		if memberships[0].ListID != listID {
			t.Errorf("membership list mismatch: %q vs %q", memberships[0].ListID, listID)
		}
	})
}

func TestEnsurePersonalMembership_IsIdempotent(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)

		u, err := users.NewService(q).UpsertByAppleSub(ctx, "sub-B", nil)
		if err != nil {
			t.Fatalf("upsert user: %v", err)
		}

		s := lists.NewService(q)
		first, err := s.EnsurePersonalMembership(ctx, u.ID)
		if err != nil {
			t.Fatalf("first: %v", err)
		}
		second, err := s.EnsurePersonalMembership(ctx, u.ID)
		if err != nil {
			t.Fatalf("second: %v", err)
		}

		if first != second {
			t.Errorf("expected same list_id on repeat call, got %q then %q", first, second)
		}
	})
}
