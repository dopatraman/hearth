package api_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dopatraman/hearth/server/internal/auth"
	"github.com/dopatraman/hearth/server/internal/dbtest"
)

func TestSnooze_HidesFromBacklog(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-snooze-hide"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		req := httptest.NewRequest(http.MethodPost, "/chores/"+id+"/snooze", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("snooze: %d body=%s", w.Code, w.Body.String())
		}

		if titles := listChoreTitles(t, srv, token); len(titles) != 0 {
			t.Errorf("backlog should be empty after snooze, got %v", titles)
		}
	})
}

func TestSnooze_IsPerUser(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-snooze-alice"})
		aliceToken, _ := signInHelper(t, aliceSrv)
		id := createChore(t, aliceSrv, aliceToken, "trash")

		code := createInvite(t, aliceSrv, aliceToken)
		bobSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-snooze-bob"})
		bobToken, _ := signInHelper(t, bobSrv)
		req := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("accept: %d", w.Code)
		}

		// Alice snoozes.
		req = httptest.NewRequest(http.MethodPost, "/chores/"+id+"/snooze", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w = httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("alice snooze: %d", w.Code)
		}

		// Alice's backlog is empty, Bob's still shows the chore.
		if titles := listChoreTitles(t, aliceSrv, aliceToken); len(titles) != 0 {
			t.Errorf("alice backlog: want empty, got %v", titles)
		}
		if titles := listChoreTitles(t, bobSrv, bobToken); len(titles) != 1 || titles[0] != "trash" {
			t.Errorf("bob backlog: want [trash], got %v", titles)
		}
	})
}

func TestSnooze_UpsertReplaces(t *testing.T) {
	// Snoozing twice must leave exactly one row (PK upsert), not two.
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-snooze-upsert"})
		token, userID := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodPost, "/chores/"+id+"/snooze", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			srv.Router().ServeHTTP(w, req)
			if w.Code != http.StatusNoContent {
				t.Fatalf("snooze %d: %d", i, w.Code)
			}
		}

		var n int
		if err := tx.QueryRow(
			`SELECT COUNT(*) FROM snoozes WHERE user_id = ? AND chore_id = ?`,
			userID, id,
		).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("snoozes rows: want 1, got %d", n)
		}
	})
}

func TestSnooze_ExpiredReappears(t *testing.T) {
	// Simulate an expired snooze by writing one directly with a past
	// timestamp — the backlog query filters by snoozed_until > now().
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-snooze-expired"})
		token, userID := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		past := time.Now().UTC().Add(-time.Hour).Format("2006-01-02T15:04:05.000Z")
		if _, err := tx.Exec(
			`INSERT INTO snoozes (user_id, chore_id, snoozed_until) VALUES (?, ?, ?)`,
			userID, id, past,
		); err != nil {
			t.Fatal(err)
		}

		if titles := listChoreTitles(t, srv, token); len(titles) != 1 || titles[0] != "trash" {
			t.Errorf("want [trash], got %v", titles)
		}
	})
}

func TestSnooze_WritesEvent(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-snooze-event"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		req := httptest.NewRequest(http.MethodPost, "/chores/"+id+"/snooze", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("snooze: %d", w.Code)
		}

		var n int
		if err := tx.QueryRow(
			`SELECT COUNT(*) FROM events WHERE chore_id = ? AND type = 'snoozed'`,
			id,
		).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("snoozed events: want 1, got %d", n)
		}
	})
}

