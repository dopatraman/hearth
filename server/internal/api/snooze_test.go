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

func TestSnooze_MarksChoreSnoozedInBacklog(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-snooze-marks"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		req := httptest.NewRequest(http.MethodPost, "/chores/"+id+"/snooze", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("snooze: %d body=%s", w.Code, w.Body.String())
		}

		rows := listChores(t, srv, token)
		if len(rows) != 1 || rows[0].Title != "trash" {
			t.Fatalf("backlog: want [trash], got %+v", rows)
		}
		if rows[0].SnoozedUntil == nil {
			t.Error("snoozed_until: nil, want non-nil for the snoozing user")
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

		// Alice sees the chore with snoozed_until; Bob sees it without.
		aliceRows := listChores(t, aliceSrv, aliceToken)
		if len(aliceRows) != 1 || aliceRows[0].SnoozedUntil == nil {
			t.Errorf("alice backlog: want 1 chore with snoozed_until, got %+v", aliceRows)
		}
		bobRows := listChores(t, bobSrv, bobToken)
		if len(bobRows) != 1 || bobRows[0].SnoozedUntil != nil {
			t.Errorf("bob backlog: want 1 chore with snoozed_until=nil, got %+v", bobRows)
		}
	})
}

func TestWake_RemovesSnooze(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-wake"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		// Snooze then wake.
		req := httptest.NewRequest(http.MethodPost, "/chores/"+id+"/snooze", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("snooze: %d", w.Code)
		}

		req = httptest.NewRequest(http.MethodDelete, "/chores/"+id+"/snooze", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("wake: %d body=%s", w.Code, w.Body.String())
		}

		rows := listChores(t, srv, token)
		if len(rows) != 1 || rows[0].SnoozedUntil != nil {
			t.Errorf("after wake: want snoozed_until=nil, got %+v", rows)
		}
	})
}

func TestWake_IsIdempotent(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-wake-idem"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		// Wake with no prior snooze — still 204.
		req := httptest.NewRequest(http.MethodDelete, "/chores/"+id+"/snooze", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("wake on non-snoozed: %d body=%s", w.Code, w.Body.String())
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

func TestSnooze_ExpiredSurfacesWithNilSnoozedUntil(t *testing.T) {
	// An expired snooze row doesn't surface as a snooze on the backlog —
	// the LEFT JOIN's `s.snoozed_until > now()` clause fails to match, so
	// the returned row's SnoozedUntil is NULL.
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

		rows := listChores(t, srv, token)
		if len(rows) != 1 || rows[0].Title != "trash" {
			t.Fatalf("want [trash], got %+v", rows)
		}
		if rows[0].SnoozedUntil != nil {
			t.Errorf("snoozed_until: want nil for expired snooze, got %v", *rows[0].SnoozedUntil)
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

