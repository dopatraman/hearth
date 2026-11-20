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
	ClaimedAt    *string `json:"claimed_at"`
	SectionOrder *int   `json:"section_order"`
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

func createUnclaimed(t *testing.T, srv *api.Server, token, title string) reorderChore {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/chores",
		strings.NewReader(`{"title":"`+title+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %q: %d body=%s", title, w.Code, w.Body.String())
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

// --- Section-order lifecycle ---

func TestCreateWithClaim_AssignsNextClaimedSectionOrder(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-order-claim"})
		token, _ := signInHelper(t, srv)

		a := createAndClaim(t, srv, token, "a")
		b := createAndClaim(t, srv, token, "b")
		c := createAndClaim(t, srv, token, "c")

		rows := backlogByID(t, srv, token)
		if rows[a.ID].SectionOrder == nil || *rows[a.ID].SectionOrder != 1 {
			t.Errorf("a.section_order: got %v want 1", rows[a.ID].SectionOrder)
		}
		if rows[b.ID].SectionOrder == nil || *rows[b.ID].SectionOrder != 2 {
			t.Errorf("b.section_order: got %v want 2", rows[b.ID].SectionOrder)
		}
		if rows[c.ID].SectionOrder == nil || *rows[c.ID].SectionOrder != 3 {
			t.Errorf("c.section_order: got %v want 3", rows[c.ID].SectionOrder)
		}
	})
}

func TestCreateWithoutClaim_AssignsNextUpForGrabsSectionOrder(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-order-unclaimed-create"})
		token, _ := signInHelper(t, srv)

		a := createUnclaimed(t, srv, token, "a")
		b := createUnclaimed(t, srv, token, "b")
		c := createUnclaimed(t, srv, token, "c")

		rows := backlogByID(t, srv, token)
		if got := rows[a.ID].SectionOrder; got == nil || *got != 1 {
			t.Errorf("a.section_order: got %v want 1", got)
		}
		if got := rows[b.ID].SectionOrder; got == nil || *got != 2 {
			t.Errorf("b.section_order: got %v want 2", got)
		}
		if got := rows[c.ID].SectionOrder; got == nil || *got != 3 {
			t.Errorf("c.section_order: got %v want 3", got)
		}
	})
}

func TestRelease_MovesToUpForGrabsBottom(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-order-release"})
		token, _ := signInHelper(t, srv)

		// Two existing unclaimed chores occupy positions 1 and 2 in Up-for-grabs.
		_ = createUnclaimed(t, srv, token, "x")
		_ = createUnclaimed(t, srv, token, "y")
		// Then claim and release a third chore; it should land at Up-for-grabs 3.
		z := createAndClaim(t, srv, token, "z")

		req := httptest.NewRequest(http.MethodDelete, "/chores/"+z.ID+"/claim", nil)
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
		if out.SectionOrder == nil || *out.SectionOrder != 3 {
			t.Errorf("section_order after release: got %v want 3", out.SectionOrder)
		}
	})
}

func TestComplete_ClearsSectionOrder(t *testing.T) {
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
		if out.SectionOrder != nil {
			t.Errorf("section_order after complete: got %v want nil", *out.SectionOrder)
		}
	})
}

// --- Reorder: Claimed section ---

func TestReorderClaimed_RewritesOrderOneToN(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reorder-claimed-1n"})
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
			got := rows[id].SectionOrder
			if got == nil || *got != wantOrder {
				t.Errorf("%s section_order: got %v want %d", id, got, wantOrder)
			}
		}
	})
}

// --- Reorder: Up-for-grabs section ---

func TestReorderUnclaimed_RewritesOrderOneToN(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reorder-unclaimed-1n"})
		token, _ := signInHelper(t, srv)

		a := createUnclaimed(t, srv, token, "a")
		b := createUnclaimed(t, srv, token, "b")
		c := createUnclaimed(t, srv, token, "c")

		w := postReorder(t, srv, token, []string{c.ID, a.ID, b.ID})
		if w.Code != http.StatusOK {
			t.Fatalf("reorder: %d body=%s", w.Code, w.Body.String())
		}

		rows := backlogByID(t, srv, token)
		want := map[string]int{c.ID: 1, a.ID: 2, b.ID: 3}
		for id, wantOrder := range want {
			got := rows[id].SectionOrder
			if got == nil || *got != wantOrder {
				t.Errorf("%s section_order: got %v want %d", id, got, wantOrder)
			}
		}
	})
}

// --- Reorder: validation ---

func TestReorder_RejectsMixedSections(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reorder-mixed"})
		token, _ := signInHelper(t, srv)

		claimed := createAndClaim(t, srv, token, "claimed")
		unclaimed := createUnclaimed(t, srv, token, "unclaimed")

		w := postReorder(t, srv, token, []string{claimed.ID, unclaimed.ID})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400 for mixed-section body, got %d body=%s", w.Code, w.Body.String())
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
