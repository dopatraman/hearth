package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dopatraman/hearth/server/internal/auth"
	"github.com/dopatraman/hearth/server/internal/dbtest"
)

func TestRelease_ClaimantCanRelease(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-release-ok"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		// Claim.
		req := httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("claim: %d", w.Code)
		}

		// Set a will_complete_by so we can confirm it's cleared on release.
		req = httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/will-complete-by",
			strings.NewReader(`{"will_complete_by":"2099-01-01T00:00:00.000Z"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("set wcb: %d", w.Code)
		}

		// Release.
		req = httptest.NewRequest(http.MethodDelete, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("release: %d body=%s", w.Code, w.Body.String())
		}
		var out map[string]any
		_ = json.NewDecoder(w.Body).Decode(&out)
		if out["claimed_by"] != nil {
			t.Errorf("claimed_by: want nil, got %v", out["claimed_by"])
		}
		if out["claimed_at"] != nil {
			t.Errorf("claimed_at: want nil, got %v", out["claimed_at"])
		}
		if out["will_complete_by"] != nil {
			t.Errorf("will_complete_by: want nil, got %v", out["will_complete_by"])
		}
		if out["state"] != "open" {
			t.Errorf("state: want open, got %v", out["state"])
		}
	})
}

func TestRelease_ReturnsToBacklogUnclaimed(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-release-backlog"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		req := httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("claim: %d", w.Code)
		}

		req = httptest.NewRequest(http.MethodDelete, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("release: %d", w.Code)
		}

		// Backlog still contains it, but unclaimed.
		req = httptest.NewRequest(http.MethodGet, "/chores", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("backlog: %d", w.Code)
		}
		var resp struct {
			Chores []map[string]any `json:"chores"`
		}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Chores) != 1 {
			t.Fatalf("want 1 chore, got %d", len(resp.Chores))
		}
		if resp.Chores[0]["claimed_by"] != nil {
			t.Errorf("backlog chore claimed_by: want nil, got %v", resp.Chores[0]["claimed_by"])
		}
	})
}

func TestRelease_NonClaimantReturns409(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-release-alice"})
		aliceToken, _ := signInHelper(t, aliceSrv)
		id := createChore(t, aliceSrv, aliceToken, "trash")
		code := createInvite(t, aliceSrv, aliceToken)

		bobSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-release-bob"})
		bobToken, _ := signInHelper(t, bobSrv)
		req := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("accept: %d", w.Code)
		}

		// Alice claims.
		req = httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w = httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("alice claim: %d", w.Code)
		}

		// Bob tries to release Alice's claim — 409.
		req = httptest.NewRequest(http.MethodDelete, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w = httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("bob release: want 409, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestRelease_UnclaimedReturns409(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-release-unclaimed"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		req := httptest.NewRequest(http.MethodDelete, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("want 409, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestRelease_AnotherUserCanReclaim(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reclaim-alice"})
		aliceToken, _ := signInHelper(t, aliceSrv)
		id := createChore(t, aliceSrv, aliceToken, "trash")
		code := createInvite(t, aliceSrv, aliceToken)

		bobSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-reclaim-bob"})
		bobToken, bobID := signInHelper(t, bobSrv)
		req := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("accept: %d", w.Code)
		}

		// Alice claims then releases.
		req = httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w = httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		req = httptest.NewRequest(http.MethodDelete, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w = httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("alice release: %d", w.Code)
		}

		// Bob claims successfully.
		req = httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w = httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("bob claim: %d body=%s", w.Code, w.Body.String())
		}
		var out map[string]any
		_ = json.NewDecoder(w.Body).Decode(&out)
		if cb, _ := out["claimed_by"].(string); cb != bobID {
			t.Errorf("claimed_by: want %q, got %v", bobID, out["claimed_by"])
		}
	})
}
