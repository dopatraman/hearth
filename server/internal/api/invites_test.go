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

// --- POST /invites ---

func TestInvites_Create_ReturnsSixDigitCode(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-invite-A"})
		token, _ := signInHelper(t, srv)

		// Need to force personal list provisioning first by creating a chore
		// (createInvite calls EnsurePersonalMembership, which also provisions).
		req := httptest.NewRequest(http.MethodPost, "/invites", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status: got %d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			Code      string `json:"code"`
			ListID    string `json:"list_id"`
			ExpiresAt string `json:"expires_at"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Code) != 6 {
			t.Errorf("code len: got %d want 6 (code=%q)", len(resp.Code), resp.Code)
		}
		if resp.ListID == "" {
			t.Error("list_id empty")
		}
		if resp.ExpiresAt == "" {
			t.Error("expires_at empty")
		}
	})
}

func TestInvites_Create_RequiresAuth(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-invite-B"})

		req := httptest.NewRequest(http.MethodPost, "/invites", nil)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", w.Code)
		}
	})
}

// --- POST /invites/{code}/accept ---

func TestInvites_Accept_NonexistentReturns404(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-accept-A"})
		token, _ := signInHelper(t, srv)

		req := httptest.NewRequest(http.MethodPost, "/invites/999999/accept", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("want 404, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestInvites_Accept_IsIdempotent(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-alice-idem"})
		aliceToken, _ := signInHelper(t, aliceSrv)

		code := createInvite(t, aliceSrv, aliceToken)

		bobSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-bob-idem"})
		bobToken, _ := signInHelper(t, bobSrv)

		// First accept — succeeds.
		req := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("first accept: %d body=%s", w.Code, w.Body.String())
		}

		// Second accept — same request, should also succeed (no duplicate membership error).
		req = httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w = httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("second accept: %d body=%s", w.Code, w.Body.String())
		}
	})
}

// The money shot: Alice creates a chore, invites Bob, Bob accepts, Bob's
// GET /chores returns Alice's chore.
func TestInvites_AfterAccept_BobSeesAlicesChores(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		// Alice signs in, creates a chore, generates an invite.
		aliceSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-alice-share"})
		aliceToken, _ := signInHelper(t, aliceSrv)

		req := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"take out trash"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w := httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("alice create chore: %d", w.Code)
		}

		code := createInvite(t, aliceSrv, aliceToken)

		// Bob signs in and accepts.
		bobSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-bob-share"})
		bobToken, _ := signInHelper(t, bobSrv)

		// Pre-accept: Bob's backlog is empty.
		if gotTitles := listChoreTitles(t, bobSrv, bobToken); len(gotTitles) != 0 {
			t.Errorf("bob pre-accept backlog should be empty, got %v", gotTitles)
		}

		req = httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w = httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("bob accept: %d body=%s", w.Code, w.Body.String())
		}

		// Post-accept: Bob sees Alice's chore.
		titles := listChoreTitles(t, bobSrv, bobToken)
		if len(titles) != 1 || titles[0] != "take out trash" {
			t.Errorf("bob post-accept backlog: got %v, want [take out trash]", titles)
		}
	})
}

