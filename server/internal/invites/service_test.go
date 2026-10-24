package invites_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/dbtest"
	"github.com/dopatraman/hearth/server/internal/invites"
	"github.com/dopatraman/hearth/server/internal/lists"
	"github.com/dopatraman/hearth/server/internal/users"
)

// setupUserWithList returns (userID, listID) for a freshly provisioned user.
func setupUserWithList(t *testing.T, ctx context.Context, q *db.Queries, sub string) (string, string) {
	t.Helper()
	u, err := users.NewService(q).UpsertByAppleSub(ctx, sub, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	listID, err := lists.NewService(q).EnsurePersonalMembership(ctx, u.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	return u.ID, listID
}

func TestInvites_Create_ReturnsSixDigitCode(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserWithList(t, ctx, q, "sub-inv-A")

		inv, err := invites.NewService(q).Create(ctx, listID, userID)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if len(inv.Code) != 6 {
			t.Errorf("code len: got %d want 6 (code=%q)", len(inv.Code), inv.Code)
		}
		for _, c := range inv.Code {
			if c < '0' || c > '9' {
				t.Errorf("code should be all digits, got %q", inv.Code)
				break
			}
		}
		if inv.ListID != listID {
			t.Errorf("list_id: got %q want %q", inv.ListID, listID)
		}
		if inv.CreatedBy != userID {
			t.Errorf("created_by: got %q want %q", inv.CreatedBy, userID)
		}
		if inv.ExpiresAt == "" {
			t.Error("expires_at empty")
		}
	})
}

func TestInvites_Create_ExpiresIn24h(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserWithList(t, ctx, q, "sub-inv-B")

		now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
		svc := invites.NewService(q).WithClock(func() time.Time { return now })

		inv, err := svc.Create(ctx, listID, userID)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		want := now.Add(24 * time.Hour).Format("2006-01-02T15:04:05.000Z")
		if inv.ExpiresAt != want {
			t.Errorf("expires_at: got %q want %q", inv.ExpiresAt, want)
		}
	})
}

func TestInvites_Lookup_ReturnsFreshInvite(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserWithList(t, ctx, q, "sub-inv-C")

		svc := invites.NewService(q)
		inv, err := svc.Create(ctx, listID, userID)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		found, err := svc.Lookup(ctx, inv.Code)
		if err != nil {
			t.Fatalf("lookup: %v", err)
		}
		if found.Code != inv.Code {
			t.Errorf("code mismatch: %q vs %q", found.Code, inv.Code)
		}
		if found.ListID != listID {
			t.Errorf("list_id mismatch")
		}
	})
}

func TestInvites_Lookup_NotFound(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		_, err := invites.NewService(q).Lookup(ctx, "000000")
		if !errors.Is(err, invites.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})
}

func TestInvites_Lookup_Expired(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserWithList(t, ctx, q, "sub-inv-D")

		// Create at T0 (short TTL window of 24h is baked into Service).
		t0 := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
		createSvc := invites.NewService(q).WithClock(func() time.Time { return t0 })
		inv, err := createSvc.Create(ctx, listID, userID)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		// Look up at T0 + 25h, past expiry.
		lookupSvc := invites.NewService(q).WithClock(func() time.Time {
			return t0.Add(25 * time.Hour)
		})
		_, err = lookupSvc.Lookup(ctx, inv.Code)
		if !errors.Is(err, invites.ErrExpired) {
			t.Fatalf("want ErrExpired, got %v", err)
		}
	})
}
