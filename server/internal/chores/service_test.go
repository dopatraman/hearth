package chores_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/dopatraman/hearth/server/internal/chores"
	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/dbtest"
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

func TestChores_Create_WritesChoreAndEvent(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-chore-A")

		notes := "take out both bins"
		chore, err := chores.NewService(q).Create(ctx, listID, userID, "trash", &notes)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if chore.ID == "" {
			t.Error("id empty")
		}
		if chore.Title != "trash" {
			t.Errorf("title: got %q want %q", chore.Title, "trash")
		}
		if chore.State != "open" {
			t.Errorf("state default: got %q want %q", chore.State, "open")
		}
		if chore.CreatorID != userID {
			t.Errorf("creator_id: got %q want %q", chore.CreatorID, userID)
		}

		events, err := q.EventsForChore(ctx, chore.ID)
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
		if ev.ChoreID != chore.ID {
			t.Errorf("event chore_id mismatch")
		}
	})
}

func TestChores_Create_RejectsEmptyTitle(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-chore-B")

		_, err := chores.NewService(q).Create(ctx, listID, userID, "", nil)
		if !errors.Is(err, chores.ErrEmptyTitle) {
			t.Fatalf("want ErrEmptyTitle, got %v", err)
		}
	})
}

func TestChores_Backlog_ReturnsOpenChoresNewestFirst(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-chore-C")
		svc := chores.NewService(q)

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
			t.Fatalf("want 3 chores, got %d", len(backlog))
		}
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

func TestChores_Backlog_IsScopedToList(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, aliceList := setupUserInList(t, ctx, q, "sub-alice")
		bobID, bobList := setupUserInList(t, ctx, q, "sub-bob")

		if aliceList == bobList {
			t.Fatal("expected distinct lists for distinct users")
		}

		svc := chores.NewService(q)
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
