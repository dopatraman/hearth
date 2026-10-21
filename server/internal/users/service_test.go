package users_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/dbtest"
	"github.com/dopatraman/hearth/server/internal/users"
)

func TestUpsertByAppleSub_CreatesNewUser(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		svc := users.NewService(db.New(tx))

		name := "Alice"
		u, err := svc.UpsertByAppleSub(ctx, "apple-sub-001", &name)
		if err != nil {
			t.Fatalf("upsert: %v", err)
		}
		if u.ID == "" {
			t.Error("id: empty")
		}
		if u.AppleSub != "apple-sub-001" {
			t.Errorf("apple_sub: got %q, want %q", u.AppleSub, "apple-sub-001")
		}
		if u.DisplayName == nil || *u.DisplayName != "Alice" {
			t.Errorf("display_name: got %v, want \"Alice\"", u.DisplayName)
		}
		if u.NotificationsEnabled != 1 {
			t.Errorf("notifications_enabled default: got %d, want 1", u.NotificationsEnabled)
		}
	})
}

func TestUpsertByAppleSub_SecondCallIsIdempotent(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		svc := users.NewService(db.New(tx))

		name := "Alice"
		first, err := svc.UpsertByAppleSub(ctx, "apple-sub-002", &name)
		if err != nil {
			t.Fatalf("first upsert: %v", err)
		}

		second, err := svc.UpsertByAppleSub(ctx, "apple-sub-002", nil)
		if err != nil {
			t.Fatalf("second upsert: %v", err)
		}

		if first.ID != second.ID {
			t.Errorf("ids differ across calls: %q vs %q", first.ID, second.ID)
		}
		if second.DisplayName == nil || *second.DisplayName != "Alice" {
			t.Errorf("display_name should be preserved when passing nil, got %v", second.DisplayName)
		}
	})
}

func TestUpsertByAppleSub_DifferentSubsAreDifferentUsers(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		svc := users.NewService(db.New(tx))

		a, err := svc.UpsertByAppleSub(ctx, "apple-sub-A", nil)
		if err != nil {
			t.Fatalf("A: %v", err)
		}
		b, err := svc.UpsertByAppleSub(ctx, "apple-sub-B", nil)
		if err != nil {
			t.Fatalf("B: %v", err)
		}
		if a.ID == b.ID {
			t.Error("expected different ids for different apple_subs")
		}
	})
}
