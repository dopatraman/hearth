package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dopatraman/hearth/server/internal/api"
	"github.com/dopatraman/hearth/server/internal/auth"
	"github.com/dopatraman/hearth/server/internal/dbtest"
)

// createRecurringChore POSTs /chores with is_recurring:true and returns the id.
func createRecurringChore(t *testing.T, srv *api.Server, token, title string) string {
	t.Helper()
	body := `{"title":"` + title + `","is_recurring":true}`
	req := httptest.NewRequest(http.MethodPost, "/chores", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("createRecurringChore %q: %d body=%s", title, w.Code, w.Body.String())
	}
	var out struct {
		ID          string `json:"id"`
		IsRecurring bool   `json:"is_recurring"`
		ClaimedBy   any    `json:"claimed_by"`
		State       string `json:"state"`
	}
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !out.IsRecurring {
		t.Fatalf("createRecurring returned is_recurring=false for %q", title)
	}
	if out.ClaimedBy != nil {
		t.Fatalf("createRecurring returned pre-claimed row for %q", title)
	}
	if out.State != "open" {
		t.Fatalf("createRecurring state: got %q want open", out.State)
	}
	return out.ID
}

// eventsOfType returns the number of events of the given type for chore id.
func eventsOfType(t *testing.T, tx *sql.Tx, choreID, typ string) int {
	t.Helper()
	var n int
	if err := tx.QueryRow(
		`SELECT COUNT(*) FROM events WHERE chore_id = ? AND type = ?`,
		choreID, typ,
	).Scan(&n); err != nil {
		t.Fatalf("eventsOfType: %v", err)
	}
	return n
}

// --- Create ---

func TestCreateRecurring_Succeeds(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-rec-create"})
		token, _ := signInHelper(t, srv)
		id := createRecurringChore(t, srv, token, "Make the bed")

		// Row is open, unclaimed, flagged recurring.
		var state string
		var claimedBy *string
		var isRecurring int
		if err := tx.QueryRow(
			`SELECT state, claimed_by, is_recurring FROM chores WHERE id = ?`,
			id,
		).Scan(&state, &claimedBy, &isRecurring); err != nil {
			t.Fatal(err)
		}
		if state != "open" || claimedBy != nil || isRecurring != 1 {
			t.Errorf("row shape: state=%q claimedBy=%v is_recurring=%d", state, claimedBy, isRecurring)
		}
	})
}

func TestCreateRecurring_RejectsClaim(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-rec-reject-claim"})
		token, _ := signInHelper(t, srv)

		body := `{"title":"bed","is_recurring":true,"claim":true}`
		req := httptest.NewRequest(http.MethodPost, "/chores", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestCreateRecurring_RejectsWillCompleteBy(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-rec-reject-wcb"})
		token, _ := signInHelper(t, srv)

		body := `{"title":"bed","is_recurring":true,"will_complete_by":"2027-01-01T00:00:00.000Z"}`
		req := httptest.NewRequest(http.MethodPost, "/chores", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

// --- Backlog ---

func TestBacklog_ExposesIsRecurring(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-rec-backlog"})
		token, _ := signInHelper(t, srv)
		recID := createRecurringChore(t, srv, token, "Water plants")
		oneOffID := createChore(t, srv, token, "Fix the door")

		req := httptest.NewRequest(http.MethodGet, "/chores", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("list: %d", w.Code)
		}
		var resp struct {
			Chores []struct {
				ID          string `json:"id"`
				IsRecurring bool   `json:"is_recurring"`
				Icon        *string `json:"icon"`
			} `json:"chores"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		var recSeen, oneOffSeen bool
		for _, c := range resp.Chores {
			if c.ID == recID {
				recSeen = true
				if !c.IsRecurring {
					t.Errorf("recurring row returned is_recurring=false")
				}
			}
			if c.ID == oneOffID {
				oneOffSeen = true
				if c.IsRecurring {
					t.Errorf("one-off row returned is_recurring=true")
				}
			}
		}
		if !recSeen || !oneOffSeen {
			t.Errorf("missing rows: rec=%v oneOff=%v", recSeen, oneOffSeen)
		}
	})
}

// --- Claim + complete flow ---

func TestCompleteRecurring_ReplenishesInsteadOfGraveyard(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-rec-replenish"})
		token, _ := signInHelper(t, srv)
		id := createRecurringChore(t, srv, token, "Unload dishwasher")

		// Claim.
		req := httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("claim: %d body=%s", w.Code, w.Body.String())
		}

		// Complete.
		req = httptest.NewRequest(http.MethodPost, "/chores/"+id+"/complete", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("complete: %d body=%s", w.Code, w.Body.String())
		}

		var body struct {
			State        string  `json:"state"`
			ClaimedBy    *string `json:"claimed_by"`
			ClaimedAt    *string `json:"claimed_at"`
			SectionOrder *int64  `json:"section_order"`
			IsRecurring  bool    `json:"is_recurring"`
		}
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.State != "open" {
			t.Errorf("state: got %q want open", body.State)
		}
		if body.ClaimedBy != nil || body.ClaimedAt != nil {
			t.Errorf("claim not cleared: claimedBy=%v claimedAt=%v", body.ClaimedBy, body.ClaimedAt)
		}
		if body.SectionOrder != nil {
			t.Errorf("section_order not cleared: got %v", *body.SectionOrder)
		}
		if !body.IsRecurring {
			t.Errorf("is_recurring dropped on replenish")
		}

		// Backlog still contains the row.
		rows := listChores(t, srv, token)
		if len(rows) != 1 || rows[0].Title != "Unload dishwasher" {
			t.Errorf("backlog after replenish: %+v", rows)
		}

		// Graveyard is empty.
		req = httptest.NewRequest(http.MethodGet, "/chores/graveyard", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		var grave struct {
			Chores []json.RawMessage `json:"chores"`
		}
		_ = json.NewDecoder(w.Body).Decode(&grave)
		if len(grave.Chores) != 0 {
			t.Errorf("graveyard should be empty after replenish, got %d rows", len(grave.Chores))
		}

		// 'replenished' event recorded, no 'completed' event.
		if n := eventsOfType(t, tx, id, "replenished"); n != 1 {
			t.Errorf("replenished events: got %d want 1", n)
		}
		if n := eventsOfType(t, tx, id, "completed"); n != 0 {
			t.Errorf("completed events on recurring chore: got %d want 0", n)
		}
	})
}

func TestCompleteRecurring_CanBeReclaimed(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-rec-recycle"})
		token, _ := signInHelper(t, srv)
		id := createRecurringChore(t, srv, token, "Water plants")

		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			srv.Router().ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("claim iter %d: %d body=%s", i, w.Code, w.Body.String())
			}

			req = httptest.NewRequest(http.MethodPost, "/chores/"+id+"/complete", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w = httptest.NewRecorder()
			srv.Router().ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("complete iter %d: %d body=%s", i, w.Code, w.Body.String())
			}
		}

		if n := eventsOfType(t, tx, id, "replenished"); n != 2 {
			t.Errorf("replenished events after two cycles: got %d want 2", n)
		}
	})
}

// Regression: non-recurring complete still goes to the graveyard.
func TestCompleteOneOff_StillGravestones(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-oneoff-still-grave"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "Fix the door")

		req := httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatal("claim")
		}

		req = httptest.NewRequest(http.MethodPost, "/chores/"+id+"/complete", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("complete: %d body=%s", w.Code, w.Body.String())
		}

		// Backlog empty.
		if rows := listChores(t, srv, token); len(rows) != 0 {
			t.Errorf("backlog after one-off complete: %+v", rows)
		}
		// Graveyard has it.
		req = httptest.NewRequest(http.MethodGet, "/chores/graveyard", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		var grave struct {
			Chores []struct{ Title string } `json:"chores"`
		}
		_ = json.NewDecoder(w.Body).Decode(&grave)
		if len(grave.Chores) != 1 || grave.Chores[0].Title != "Fix the door" {
			t.Errorf("graveyard: %+v", grave.Chores)
		}
		if n := eventsOfType(t, tx, id, "completed"); n != 1 {
			t.Errorf("completed events: got %d want 1", n)
		}
		if n := eventsOfType(t, tx, id, "replenished"); n != 0 {
			t.Errorf("replenished events on one-off: got %d want 0", n)
		}
	})
}

// Visible push alert still fires when a recurring chore is completed — the
// chore lives on but the household still wants the "<Actor> completed '<X>'"
// notification. Verifies the alert string is populated, not empty.
func TestCompleteRecurring_VisiblePushAlertFires(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		pusher := newFakePusher()
		aliceSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-rec-push-alice"}, pusher)
		aliceToken, _ := signInHelper(t, aliceSrv)
		code := createInvite(t, aliceSrv, aliceToken)

		bobSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-rec-push-bob"}, pusher)
		bobToken, _ := signInHelper(t, bobSrv)
		req := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("accept: %d", w.Code)
		}
		registerDevice(t, bobSrv, bobToken, "ios", "bob-phone-rec")

		id := createRecurringChore(t, aliceSrv, aliceToken, "Trash")
		pusher.wait(t) // created push (silent)

		// Alice claims.
		req = httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w = httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("claim: %d body=%s", w.Code, w.Body.String())
		}
		pusher.wait(t) // claimed push (visible)

		// Alice completes → replenishes. The alert text should still look like a
		// completion ("<Actor> completed \"Trash\"") — verifies that the handler
		// branches to replenish WITHOUT dropping the visible push.
		req = httptest.NewRequest(http.MethodPost, "/chores/"+id+"/complete", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w = httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("complete: %d body=%s", w.Code, w.Body.String())
		}
		call := pusher.wait(t)
		if !strings.Contains(call.payload.Alert, "completed") || !strings.Contains(call.payload.Alert, "Trash") {
			t.Errorf("complete push alert: got %q want completed+Trash", call.payload.Alert)
		}
	})
}
