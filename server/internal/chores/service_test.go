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
		chore, err := chores.NewService(q).Create(ctx, listID, userID, "trash", &notes, false, nil)
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

		_, err := chores.NewService(q).Create(ctx, listID, userID, "", nil, false, nil)
		if !errors.Is(err, chores.ErrEmptyTitle) {
			t.Fatalf("want ErrEmptyTitle, got %v", err)
		}
	})
}

func TestChores_Create_WithClaim_SetsClaimedBy(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-create-claim")

		chore, err := chores.NewService(q).Create(ctx, listID, userID, "trash", nil, true, nil)
		if err != nil {
			t.Fatalf("create+claim: %v", err)
		}
		if chore.ClaimedBy == nil || *chore.ClaimedBy != userID {
			t.Errorf("claimed_by: got %v want %q", chore.ClaimedBy, userID)
		}
		if chore.ClaimedAt == nil {
			t.Error("claimed_at: nil")
		}
	})
}

func TestChores_Create_WithClaim_EmitsCreatedThenClaimedEvents(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-create-claim-events")

		chore, err := chores.NewService(q).Create(ctx, listID, userID, "trash", nil, true, nil)
		if err != nil {
			t.Fatalf("create+claim: %v", err)
		}

		events, err := q.EventsForChore(ctx, chore.ID)
		if err != nil {
			t.Fatalf("events: %v", err)
		}
		if len(events) != 2 {
			t.Fatalf("want 2 events, got %d: %+v", len(events), events)
		}
		if events[0].Type != "created" {
			t.Errorf("event[0] type: got %q want created", events[0].Type)
		}
		if events[1].Type != "claimed" {
			t.Errorf("event[1] type: got %q want claimed", events[1].Type)
		}
		for i, e := range events {
			if e.ActorID != userID {
				t.Errorf("event[%d] actor: got %q want %q", i, e.ActorID, userID)
			}
		}
	})
}

func TestChores_Create_WithClaimAndWillCompleteBy_PersistsDateAndEmitsAllEvents(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-create-when")

		when := "2026-10-28T23:59:59.999Z"
		chore, err := chores.NewService(q).Create(ctx, listID, userID, "trash", nil, true, &when)
		if err != nil {
			t.Fatalf("create+claim+when: %v", err)
		}
		if chore.WillCompleteBy == nil || *chore.WillCompleteBy != when {
			t.Errorf("will_complete_by: got %v want %q", chore.WillCompleteBy, when)
		}
		if chore.ClaimedBy == nil || *chore.ClaimedBy != userID {
			t.Errorf("claimed_by: got %v want %q", chore.ClaimedBy, userID)
		}

		events, err := q.EventsForChore(ctx, chore.ID)
		if err != nil {
			t.Fatalf("events: %v", err)
		}
		if len(events) != 3 {
			t.Fatalf("want 3 events, got %d: %+v", len(events), events)
		}
		want := []string{"created", "claimed", "will_complete_by_set"}
		for i, w := range want {
			if events[i].Type != w {
				t.Errorf("event[%d] type: got %q want %q", i, events[i].Type, w)
			}
		}
	})
}

func TestChores_Create_WillCompleteByWithoutClaim_IsRejected(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-create-when-no-claim")

		when := "2026-10-28T23:59:59.999Z"
		_, err := chores.NewService(q).Create(ctx, listID, userID, "trash", nil, false, &when)
		if !errors.Is(err, chores.ErrWillCompleteByRequiresClaim) {
			t.Fatalf("want ErrWillCompleteByRequiresClaim, got %v", err)
		}
	})
}

func TestChores_Create_WithoutClaim_LeavesClaimedByNull(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-create-no-claim")

		chore, err := chores.NewService(q).Create(ctx, listID, userID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if chore.ClaimedBy != nil {
			t.Errorf("claimed_by: got %v want nil", chore.ClaimedBy)
		}

		events, err := q.EventsForChore(ctx, chore.ID)
		if err != nil {
			t.Fatalf("events: %v", err)
		}
		if len(events) != 1 || events[0].Type != "created" {
			t.Errorf("want single created event, got %+v", events)
		}
	})
}

func TestChores_Backlog_ReturnsOpenChoresNewestFirst(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-chore-C")
		svc := chores.NewService(q)

		for _, title := range []string{"first", "second", "third"} {
			if _, err := svc.Create(ctx, listID, userID, title, nil, false, nil); err != nil {
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

// --- claim ---

func TestChores_Claim_SetsClaimedBy(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-claim-A")

		svc := chores.NewService(q)
		created, err := svc.Create(ctx, listID, userID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		claimed, err := svc.Claim(ctx, created.ID, userID)
		if err != nil {
			t.Fatalf("claim: %v", err)
		}
		if claimed.ClaimedBy == nil || *claimed.ClaimedBy != userID {
			t.Errorf("claimed_by: got %v want %q", claimed.ClaimedBy, userID)
		}
		if claimed.ClaimedAt == nil {
			t.Error("claimed_at: nil")
		}
	})
}

func TestChores_Claim_SecondClaimReturnsErrNotClaimable(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, listID := setupUserInList(t, ctx, q, "sub-claim-race-A")
		bobID, _ := setupUserInList(t, ctx, q, "sub-claim-race-B")

		svc := chores.NewService(q)
		chore, err := svc.Create(ctx, listID, aliceID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		if _, err := svc.Claim(ctx, chore.ID, aliceID); err != nil {
			t.Fatalf("alice claim: %v", err)
		}

		_, err = svc.Claim(ctx, chore.ID, bobID)
		if !errors.Is(err, chores.ErrNotClaimable) {
			t.Fatalf("want ErrNotClaimable, got %v", err)
		}
	})
}

func TestChores_Claim_WritesClaimedEvent(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-claim-event")

		svc := chores.NewService(q)
		chore, err := svc.Create(ctx, listID, userID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		_, err = svc.Claim(ctx, chore.ID, userID)
		if err != nil {
			t.Fatalf("claim: %v", err)
		}

		events, err := q.EventsForChore(ctx, chore.ID)
		if err != nil {
			t.Fatalf("events: %v", err)
		}
		var seenClaimed bool
		for _, e := range events {
			if e.Type == "claimed" && e.ActorID == userID {
				seenClaimed = true
			}
		}
		if !seenClaimed {
			t.Errorf("no 'claimed' event with actor=%q in %+v", userID, events)
		}
	})
}

// --- complete ---

func TestChores_Complete_SetsStateAndCompletedBy(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-complete-A")
		svc := chores.NewService(q)

		chore, err := svc.Create(ctx, listID, userID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := svc.Claim(ctx, chore.ID, userID); err != nil {
			t.Fatalf("claim: %v", err)
		}

		completed, err := svc.Complete(ctx, chore.ID, userID)
		if err != nil {
			t.Fatalf("complete: %v", err)
		}
		if completed.State != "completed" {
			t.Errorf("state: got %q want completed", completed.State)
		}
		if completed.CompletedBy == nil || *completed.CompletedBy != userID {
			t.Errorf("completed_by: got %v want %q", completed.CompletedBy, userID)
		}
	})
}

func TestChores_Complete_RejectsNonClaimant(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, listID := setupUserInList(t, ctx, q, "sub-complete-alice")
		bobID, _ := setupUserInList(t, ctx, q, "sub-complete-bob")
		svc := chores.NewService(q)

		chore, err := svc.Create(ctx, listID, aliceID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := svc.Claim(ctx, chore.ID, aliceID); err != nil {
			t.Fatalf("claim: %v", err)
		}

		// Bob tries to complete Alice's claimed chore.
		_, err = svc.Complete(ctx, chore.ID, bobID)
		if !errors.Is(err, chores.ErrNotCompletable) {
			t.Fatalf("want ErrNotCompletable, got %v", err)
		}
	})
}

func TestChores_Complete_UnclaimedChoreReturnsErrNotCompletable(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-complete-unclaimed")
		svc := chores.NewService(q)

		chore, err := svc.Create(ctx, listID, userID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		// Skipping claim; try to complete directly.
		_, err = svc.Complete(ctx, chore.ID, userID)
		if !errors.Is(err, chores.ErrNotCompletable) {
			t.Fatalf("want ErrNotCompletable, got %v", err)
		}
	})
}

// --- graveyard ---

func TestChores_Graveyard_ListsCompletedChoresNewestFirst(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-grave-A")
		svc := chores.NewService(q)

		for _, title := range []string{"trash", "dishes", "mop"} {
			c, err := svc.Create(ctx, listID, userID, title, nil, false, nil)
			if err != nil {
				t.Fatalf("create %q: %v", title, err)
			}
			if _, err := svc.Claim(ctx, c.ID, userID); err != nil {
				t.Fatalf("claim %q: %v", title, err)
			}
			if _, err := svc.Complete(ctx, c.ID, userID); err != nil {
				t.Fatalf("complete %q: %v", title, err)
			}
		}

		grave, err := svc.Graveyard(ctx, userID)
		if err != nil {
			t.Fatalf("graveyard: %v", err)
		}
		if len(grave) != 3 {
			t.Fatalf("want 3 completed, got %d", len(grave))
		}
	})
}

func TestChores_SearchGraveyard_MatchesSubstringCaseInsensitive(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-search")
		svc := chores.NewService(q)

		for _, title := range []string{"Take out trash", "Dishes", "Mop"} {
			c, _ := svc.Create(ctx, listID, userID, title, nil, false, nil)
			_, _ = svc.Claim(ctx, c.ID, userID)
			_, _ = svc.Complete(ctx, c.ID, userID)
		}

		results, err := svc.SearchGraveyard(ctx, userID, "TRASH")
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(results) != 1 || results[0].Title != "Take out trash" {
			t.Errorf("search TRASH: got %+v", results)
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
		if _, err := svc.Create(ctx, aliceList, aliceID, "alice-only", nil, false, nil); err != nil {
			t.Fatalf("alice create: %v", err)
		}
		if _, err := svc.Create(ctx, bobList, bobID, "bob-only", nil, false, nil); err != nil {
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
