package api_test

import (
	"bytes"
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

type reorderChore struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	ClaimedOrder *int   `json:"claimed_order"`
}

func createAndClaim(t *testing.T, srv *api.Server, token, title string) reorderChore {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/chores",
		strings.NewReader(`{"title":"`+title+`","claim":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create+claim %q: %d body=%s", title, w.Code, w.Body.String())
	}
	var out reorderChore
	if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
		t.Fatalf("create decode: %v", err)
	}
	return out
}

func postReorder(t *testing.T, srv *api.Server, token string, ids []string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"order": ids})
	req := httptest.NewRequest(http.MethodPost, "/chores/reorder", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	return w
}

func backlogByID(t *testing.T, srv *api.Server, token string) map[string]reorderChore {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/chores", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("backlog: %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Chores []reorderChore `json:"chores"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("backlog decode: %v", err)
	}
	out := map[string]reorderChore{}
	for _, c := range resp.Chores {
		out[c.ID] = c
	}
	return out
}

func TestClaim_AssignsNextClaimedOrder(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-order-claim"})
		token, _ := signInHelper(t, srv)

		a := createAndClaim(t, srv, token, "a")
		b := createAndClaim(t, srv, token, "b")
		c := createAndClaim(t, srv, token, "c")

		rows := backlogByID(t, srv, token)
		if rows[a.ID].ClaimedOrder == nil || *rows[a.ID].ClaimedOrder != 1 {
			t.Errorf("a.claimed_order: got %v want 1", rows[a.ID].ClaimedOrder)
		}
		if rows[b.ID].ClaimedOrder == nil || *rows[b.ID].ClaimedOrder != 2 {
			t.Errorf("b.claimed_order: got %v want 2", rows[b.ID].ClaimedOrder)
		}
		if rows[c.ID].ClaimedOrder == nil || *rows[c.ID].ClaimedOrder != 3 {
			t.Errorf("c.claimed_order: got %v want 3", rows[c.ID].ClaimedOrder)
		}
	})
}

func TestRelease_ClearsClaimedOrder(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-order-release"})
		token, _ := signInHelper(t, srv)
		c := createAndClaim(t, srv, token, "a")

		req := httptest.NewRequest(http.MethodDelete, "/chores/"+c.ID+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("release: %d body=%s", w.Code, w.Body.String())
		}

		var out reorderChore
		if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if out.ClaimedOrder != nil {
			t.Errorf("claimed_order after release: got %v want nil", *out.ClaimedOrder)
		}
	})
}

func TestComplete_ClearsClaimedOrder(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-order-complete"})
		token, _ := signInHelper(t, srv)
		c := createAndClaim(t, srv, token, "a")

		req := httptest.NewRequest(http.MethodPost, "/chores/"+c.ID+"/complete", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("complete: %d body=%s", w.Code, w.Body.String())
		}

		var out reorderChore
		if err := json.NewDecoder(w.Body).Decode(&out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if out.ClaimedOrder != nil {
			t.Errorf("claimed_order after complete: got %v want nil", *out.ClaimedOrder)
		}
	})
}

func TestReorder_RewritesOrderOneToN(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reorder-1n"})
		token, _ := signInHelper(t, srv)

		a := createAndClaim(t, srv, token, "a")
		b := createAndClaim(t, srv, token, "b")
		c := createAndClaim(t, srv, token, "c")

		w := postReorder(t, srv, token, []string{c.ID, a.ID, b.ID})
		if w.Code != http.StatusOK {
			t.Fatalf("reorder: %d body=%s", w.Code, w.Body.String())
		}

		rows := backlogByID(t, srv, token)
		want := map[string]int{c.ID: 1, a.ID: 2, b.ID: 3}
		for id, wantOrder := range want {
			got := rows[id].ClaimedOrder
			if got == nil || *got != wantOrder {
				t.Errorf("%s claimed_order: got %v want %d", id, got, wantOrder)
			}
		}
	})
}

func TestReorder_LenientAppendForOmittedClaimedChore(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reorder-append"})
		token, _ := signInHelper(t, srv)

		a := createAndClaim(t, srv, token, "a")
		b := createAndClaim(t, srv, token, "b")
		c := createAndClaim(t, srv, token, "c")

		// Caller's body does not include c (simulating a concurrent claim).
		w := postReorder(t, srv, token, []string{b.ID, a.ID})
		if w.Code != http.StatusOK {
			t.Fatalf("reorder: %d body=%s", w.Code, w.Body.String())
		}

		rows := backlogByID(t, srv, token)
		if got := rows[b.ID].ClaimedOrder; got == nil || *got != 1 {
			t.Errorf("b: got %v want 1", got)
		}
		if got := rows[a.ID].ClaimedOrder; got == nil || *got != 2 {
			t.Errorf("a: got %v want 2", got)
		}
		if got := rows[c.ID].ClaimedOrder; got == nil || *got != 3 {
			t.Errorf("c (omitted): got %v want 3", got)
		}
	})
}

func TestReorder_RejectsUnclaimedID(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reorder-unclaimed"})
		token, _ := signInHelper(t, srv)

		c := createChore(t, srv, token, "unclaimed")

		w := postReorder(t, srv, token, []string{c})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400 for unclaimed id, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestReorder_RejectsUnknownID(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reorder-unknown"})
		token, _ := signInHelper(t, srv)
		_ = createAndClaim(t, srv, token, "a")

		w := postReorder(t, srv, token, []string{"00000000-0000-0000-0000-000000000000"})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400 for unknown id, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestReorder_RejectsEmptyOrder(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reorder-empty"})
		token, _ := signInHelper(t, srv)

		w := postReorder(t, srv, token, []string{})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400 for empty order, got %d", w.Code)
		}
	})
}

func TestReorder_RejectsDuplicateIDs(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reorder-dup"})
		token, _ := signInHelper(t, srv)
		a := createAndClaim(t, srv, token, "a")

		w := postReorder(t, srv, token, []string{a.ID, a.ID})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400 for duplicate ids, got %d", w.Code)
		}
	})
}

func TestReorder_RejectsNonMember(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reorder-alice"})
		aliceToken, _ := signInHelper(t, aliceSrv)
		a := createAndClaim(t, aliceSrv, aliceToken, "a")

		bobSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reorder-bob"})
		bobToken, _ := signInHelper(t, bobSrv)

		w := postReorder(t, bobSrv, bobToken, []string{a.ID})
		if w.Code != http.StatusForbidden {
			t.Fatalf("want 403 for non-member, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestReorder_SilentPushToOtherMembers(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		pusher := newFakePusher()
		aliceSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-reorder-push-alice"}, pusher)
		aliceToken, _ := signInHelper(t, aliceSrv)
		code := createInvite(t, aliceSrv, aliceToken)

		bobSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-reorder-push-bob"}, pusher)
		bobToken, _ := signInHelper(t, bobSrv)
		req := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("accept: %d", w.Code)
		}

		registerDevice(t, bobSrv, bobToken, "ios", "bob-reorder-token")

		a := createAndClaim(t, aliceSrv, aliceToken, "a")
		pusher.wait(t) // create
		b := createAndClaim(t, aliceSrv, aliceToken, "b")
		pusher.wait(t) // create

		w = postReorder(t, aliceSrv, aliceToken, []string{b.ID, a.ID})
		if w.Code != http.StatusOK {
			t.Fatalf("reorder: %d body=%s", w.Code, w.Body.String())
		}

		call := pusher.wait(t)
		if len(call.tokens) != 1 || call.tokens[0].Token != "bob-reorder-token" {
			t.Errorf("recipients: %+v", call.tokens)
		}
		if call.payload.Alert != "" {
			t.Errorf("reorder push should be silent, got alert %q", call.payload.Alert)
		}
	})
}
