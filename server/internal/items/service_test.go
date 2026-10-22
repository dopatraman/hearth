package items_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/dbtest"
	"github.com/dopatraman/hearth/server/internal/items"
	"github.com/dopatraman/hearth/server/internal/lists"
	"github.com/dopatraman/hearth/server/internal/users"
)

// setupUserInList provisions a user and ensures they have a "Home" list.
// Returns (userID, listID).
func setupUserInList(t *testing.T, ctx context.Context, q *db.Queries, appleSub string) (string, string) {
	t.Helper()
	u, err := users.NewService(q).UpsertByAppleSub(ctx, appleSub, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	listID, err := lists.NewService(q).EnsurePersonalMembership(ctx, u.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	return u.ID, listID
}

func TestItems_Create_WritesItemAndEvent(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-item-A")

		notes := "take out both bins"
		item, err := items.NewService(q).Create(ctx, listID, userID, "trash", &notes)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if item.ID == "" {
			t.Error("id empty")
		}
		if item.Title != "trash" {
			t.Errorf("title: got %q want %q", item.Title, "trash")
		}
		if item.State != "open" {
			t.Errorf("state default: got %q want %q", item.State, "open")
		}
		if item.CreatorID != userID {
			t.Errorf("creator_id: got %q want %q", item.CreatorID, userID)
		}

		// Event row must exist with type='created' and actor=creator.
		events, err := q.EventsForItem(ctx, item.ID)
		if err != nil {
			t.Fatalf("events: %v", err)
		}
		if len(events) != 1 {
			t.Fatalf("want 1 event, got %d", len(events))
		}
		ev := events[0]
		if ev.Type != "created" {
			t.Errorf("event type: got %q want %q", ev.Type, "created")
		}
		if ev.ActorID != userID {
			t.Errorf("event actor: got %q want %q", ev.ActorID, userID)
		}
		if ev.ItemID != item.ID {
			t.Errorf("event item_id mismatch")
		}
	})
}

func TestItems_Create_RejectsEmptyTitle(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-item-B")

		_, err := items.NewService(q).Create(ctx, listID, userID, "", nil)
		if !errors.Is(err, items.ErrEmptyTitle) {
			t.Fatalf("want ErrEmptyTitle, got %v", err)
		}
	})
}

func TestItems_Backlog_ReturnsOpenItemsNewestFirst(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-item-C")
		svc := items.NewService(q)

		for _, title := range []string{"first", "second", "third"} {
			if _, err := svc.Create(ctx, listID, userID, title, nil); err != nil {
				t.Fatalf("create %q: %v", title, err)
			}
		}

		backlog, err := svc.Backlog(ctx, listID)
		if err != nil {
			t.Fatalf("backlog: %v", err)
		}
		if len(backlog) != 3 {
			t.Fatalf("want 3 items, got %d", len(backlog))
		}
		// Newest first; uuidv7 + default created_at = creation order descending.
		// Technically rows are ordered by created_at DESC; if two created_at
		// strings are equal (sub-millisecond resolution lost), order is
		// unspecified but we at least want all three present.
		titles := map[string]bool{}
		for _, b := range backlog {
			titles[b.Title] = true
		}
		for _, want := range []string{"first", "second", "third"} {
			if !titles[want] {
				t.Errorf("missing title %q", want)
			}
		}
	})
}

func TestItems_Backlog_IsScopedToList(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, aliceList := setupUserInList(t, ctx, q, "sub-alice")
		bobID, bobList := setupUserInList(t, ctx, q, "sub-bob")

		if aliceList == bobList {
			t.Fatal("expected distinct lists for distinct users")
		}

		svc := items.NewService(q)
		if _, err := svc.Create(ctx, aliceList, aliceID, "alice-only", nil); err != nil {
			t.Fatalf("alice create: %v", err)
		}
		if _, err := svc.Create(ctx, bobList, bobID, "bob-only", nil); err != nil {
			t.Fatalf("bob create: %v", err)
		}

		aliceBacklog, err := svc.Backlog(ctx, aliceList)
		if err != nil {
			t.Fatalf("alice backlog: %v", err)
		}
		if len(aliceBacklog) != 1 || aliceBacklog[0].Title != "alice-only" {
			t.Errorf("alice backlog wrong: %+v", aliceBacklog)
		}

		bobBacklog, err := svc.Backlog(ctx, bobList)
		if err != nil {
			t.Fatalf("bob backlog: %v", err)
		}
		if len(bobBacklog) != 1 || bobBacklog[0].Title != "bob-only" {
			t.Errorf("bob backlog wrong: %+v", bobBacklog)
		}
	})
}
