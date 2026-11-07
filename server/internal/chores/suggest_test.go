package chores_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/dopatraman/hearth/server/internal/chores"
	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/dbtest"
)

// seedTwoMembers provisions Alice (sender) + Bob (recipient) and puts them in
// the same list. Returns (aliceID, bobID, listID).
func seedTwoMembers(t *testing.T, ctx context.Context, q *db.Queries, tag string) (string, string, string) {
	t.Helper()
	aliceID, listID := setupUserInList(t, ctx, q, "sub-"+tag+"-A")
	bobID, _ := setupUserInList(t, ctx, q, "sub-"+tag+"-B")
	if err := q.AddMembership(ctx, db.AddMembershipParams{ListID: listID, UserID: bobID}); err != nil {
		t.Fatalf("add bob to alice list: %v", err)
	}
	return aliceID, bobID, listID
}

func TestChores_Suggest_WritesSuggestionAndEvent(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, bobID, listID := seedTwoMembers(t, ctx, q, "suggest-ok")
		svc := chores.NewService(q)

		chore, err := svc.Create(ctx, listID, aliceID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create chore: %v", err)
		}

		note := "when you have a sec"
		sug, err := svc.Suggest(ctx, chore.ID, aliceID, bobID, &note)
		if err != nil {
			t.Fatalf("suggest: %v", err)
		}
		if sug.ID == "" {
			t.Error("suggestion id empty")
		}
		if sug.ChoreID != chore.ID {
			t.Errorf("suggestion chore_id: got %q want %q", sug.ChoreID, chore.ID)
		}
		if sug.FromUserID != aliceID {
			t.Errorf("suggestion from_user_id: got %q want %q", sug.FromUserID, aliceID)
		}
		if sug.ToUserID != bobID {
			t.Errorf("suggestion to_user_id: got %q want %q", sug.ToUserID, bobID)
		}
		if sug.Note == nil || *sug.Note != note {
			t.Errorf("suggestion note: got %v want %q", sug.Note, note)
		}
		if sug.DeclinedAt != nil {
			t.Errorf("suggestion declined_at: got %v want nil", sug.DeclinedAt)
		}

		// Paired 'suggested' event lives in the same tx.
		events, err := q.EventsForChore(ctx, chore.ID)
		if err != nil {
			t.Fatalf("events: %v", err)
		}
		var suggestedCount int
		for _, e := range events {
			if e.Type == "suggested" {
				suggestedCount++
				if e.ActorID != aliceID {
					t.Errorf("event actor_id: got %q want %q", e.ActorID, aliceID)
				}
				if e.ChoreID != chore.ID {
					t.Errorf("event chore_id: got %q want %q", e.ChoreID, chore.ID)
				}
				if e.ListID != listID {
					t.Errorf("event list_id: got %q want %q", e.ListID, listID)
				}
			}
		}
		if suggestedCount != 1 {
			t.Fatalf("want 1 'suggested' event, got %d", suggestedCount)
		}
	})
}

func TestChores_Suggest_NilNoteAccepted(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, bobID, listID := seedTwoMembers(t, ctx, q, "suggest-nil-note")
		svc := chores.NewService(q)
		chore, err := svc.Create(ctx, listID, aliceID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		sug, err := svc.Suggest(ctx, chore.ID, aliceID, bobID, nil)
		if err != nil {
			t.Fatalf("suggest with nil note: %v", err)
		}
		if sug.Note != nil {
			t.Errorf("note: got %v want nil", sug.Note)
		}
	})
}

func TestChores_Suggest_RejectsDuplicatePending(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, bobID, listID := seedTwoMembers(t, ctx, q, "suggest-dup")
		svc := chores.NewService(q)
		chore, err := svc.Create(ctx, listID, aliceID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}

		if _, err := svc.Suggest(ctx, chore.ID, aliceID, bobID, nil); err != nil {
			t.Fatalf("first suggest: %v", err)
		}
		_, err = svc.Suggest(ctx, chore.ID, aliceID, bobID, nil)
		if !errors.Is(err, chores.ErrSuggestionAlreadyPending) {
			t.Fatalf("second suggest: got %v want ErrSuggestionAlreadyPending", err)
		}

		// Still exactly one 'suggested' event.
		events, err := q.EventsForChore(ctx, chore.ID)
		if err != nil {
			t.Fatalf("events: %v", err)
		}
		var count int
		for _, e := range events {
			if e.Type == "suggested" {
				count++
			}
		}
		if count != 1 {
			t.Errorf("want 1 'suggested' event after duplicate, got %d", count)
		}
	})
}

func TestChores_Suggest_RejectsSelfSuggest(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, listID := setupUserInList(t, ctx, q, "sub-suggest-self")
		svc := chores.NewService(q)
		chore, err := svc.Create(ctx, listID, aliceID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		_, err = svc.Suggest(ctx, chore.ID, aliceID, aliceID, nil)
		if !errors.Is(err, chores.ErrSuggestionSelf) {
			t.Fatalf("got %v want ErrSuggestionSelf", err)
		}
	})
}

func TestChores_Suggest_RejectsRecipientNotInList(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, listID := setupUserInList(t, ctx, q, "sub-suggest-outsider-A")
		// Bob exists but is in his own list, not Alice's.
		bobID, _ := setupUserInList(t, ctx, q, "sub-suggest-outsider-B")
		svc := chores.NewService(q)
		chore, err := svc.Create(ctx, listID, aliceID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		_, err = svc.Suggest(ctx, chore.ID, aliceID, bobID, nil)
		if !errors.Is(err, chores.ErrSuggestionRecipientNotInList) {
			t.Fatalf("got %v want ErrSuggestionRecipientNotInList", err)
		}
	})
}

func TestChores_Suggest_RejectsClaimedChore(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, bobID, listID := seedTwoMembers(t, ctx, q, "suggest-claimed")
		svc := chores.NewService(q)
		chore, err := svc.Create(ctx, listID, aliceID, "trash", nil, true, nil)
		if err != nil {
			t.Fatalf("create+claim: %v", err)
		}
		_, err = svc.Suggest(ctx, chore.ID, aliceID, bobID, nil)
		if !errors.Is(err, chores.ErrSuggestionNotSuggestable) {
			t.Fatalf("got %v want ErrSuggestionNotSuggestable", err)
		}
	})
}

func TestChores_Suggest_RejectsNonexistentChore(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, bobID, _ := seedTwoMembers(t, ctx, q, "suggest-404")
		svc := chores.NewService(q)
		_, err := svc.Suggest(ctx, "nope-does-not-exist", aliceID, bobID, nil)
		if !errors.Is(err, chores.ErrSuggestionNotSuggestable) {
			t.Fatalf("got %v want ErrSuggestionNotSuggestable", err)
		}
	})
}

func TestChores_Claim_ByRecipient_StampsAcceptedAt(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, bobID, listID := seedTwoMembers(t, ctx, q, "claim-accepts")
		svc := chores.NewService(q)
		chore, err := svc.Create(ctx, listID, aliceID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		sug, err := svc.Suggest(ctx, chore.ID, aliceID, bobID, nil)
		if err != nil {
			t.Fatalf("suggest: %v", err)
		}

		// Bob (the recipient) claims.
		if _, err := svc.Claim(ctx, chore.ID, bobID); err != nil {
			t.Fatalf("claim: %v", err)
		}

		// Suggestion is no longer pending (dropped from the pending index).
		if _, err := q.GetPendingSuggestionForChore(ctx, chore.ID); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("pending guard: want sql.ErrNoRows, got %v", err)
		}

		// Row still exists — accepted_at stamped, invalidated_at + declined_at still nil.
		after, err := q.GetSuggestionByID(ctx, sug.ID)
		if err != nil {
			t.Fatalf("get suggestion: %v", err)
		}
		if after.AcceptedAt == nil {
			t.Error("accepted_at: expected non-nil")
		}
		if after.InvalidatedAt != nil {
			t.Errorf("invalidated_at: want nil, got %v", after.InvalidatedAt)
		}
		if after.DeclinedAt != nil {
			t.Errorf("declined_at: want nil, got %v", after.DeclinedAt)
		}
	})
}

func TestChores_Claim_ByThirdParty_StampsInvalidatedAt(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, bobID, listID := seedTwoMembers(t, ctx, q, "claim-invalidates")
		charlieID, _ := setupUserInList(t, ctx, q, "sub-claim-invalidates-C")
		if err := q.AddMembership(ctx, db.AddMembershipParams{ListID: listID, UserID: charlieID}); err != nil {
			t.Fatalf("add charlie: %v", err)
		}
		svc := chores.NewService(q)
		chore, err := svc.Create(ctx, listID, aliceID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		sug, err := svc.Suggest(ctx, chore.ID, aliceID, bobID, nil)
		if err != nil {
			t.Fatalf("suggest: %v", err)
		}

		// Charlie (not the recipient) grabs it first.
		if _, err := svc.Claim(ctx, chore.ID, charlieID); err != nil {
			t.Fatalf("claim: %v", err)
		}

		if _, err := q.GetPendingSuggestionForChore(ctx, chore.ID); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("pending guard: want sql.ErrNoRows, got %v", err)
		}

		// Row still exists — invalidated_at stamped, accepted_at + declined_at still nil.
		after, err := q.GetSuggestionByID(ctx, sug.ID)
		if err != nil {
			t.Fatalf("get suggestion: %v", err)
		}
		if after.InvalidatedAt == nil {
			t.Error("invalidated_at: expected non-nil")
		}
		if after.AcceptedAt != nil {
			t.Errorf("accepted_at: want nil, got %v", after.AcceptedAt)
		}
		if after.DeclinedAt != nil {
			t.Errorf("declined_at: want nil, got %v", after.DeclinedAt)
		}
	})
}

func TestChores_Claim_WithNoPendingSuggestion_IsNoOp(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		userID, listID := setupUserInList(t, ctx, q, "sub-claim-no-sug")
		svc := chores.NewService(q)
		chore, err := svc.Create(ctx, listID, userID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		// No suggestion. Claim must still succeed.
		if _, err := svc.Claim(ctx, chore.ID, userID); err != nil {
			t.Fatalf("claim without suggestion failed: %v", err)
		}
	})
}

func TestChores_Suggest_AllowedAfterPriorInvalidation(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		q := db.New(tx)
		aliceID, bobID, listID := seedTwoMembers(t, ctx, q, "resuggest-after-invalid")
		charlieID, _ := setupUserInList(t, ctx, q, "sub-resuggest-after-invalid-C")
		if err := q.AddMembership(ctx, db.AddMembershipParams{ListID: listID, UserID: charlieID}); err != nil {
			t.Fatalf("add charlie: %v", err)
		}
		svc := chores.NewService(q)
		chore, err := svc.Create(ctx, listID, aliceID, "trash", nil, false, nil)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := svc.Suggest(ctx, chore.ID, aliceID, bobID, nil); err != nil {
			t.Fatalf("first suggest: %v", err)
		}
		// Charlie claims → prior suggestion invalidated.
		if _, err := svc.Claim(ctx, chore.ID, charlieID); err != nil {
			t.Fatalf("claim: %v", err)
		}
		// Charlie releases — now the chore is suggestable again.
		if _, err := svc.Release(ctx, chore.ID, charlieID); err != nil {
			t.Fatalf("release: %v", err)
		}
		// Alice can re-suggest; the invalidated row doesn't block it.
		if _, err := svc.Suggest(ctx, chore.ID, aliceID, bobID, nil); err != nil {
			t.Fatalf("re-suggest after invalidation: %v", err)
		}
	})
}
