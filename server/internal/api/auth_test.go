package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dopatraman/hearth/server/internal/api"
	"github.com/dopatraman/hearth/server/internal/auth"
	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/dbtest"
)

// stubApple returns a canned identity for a canned id_token, errors otherwise.
type stubApple struct {
	byToken map[string]auth.AppleIdentity
}

func (s *stubApple) Verify(_ context.Context, token string) (auth.AppleIdentity, error) {
	id, ok := s.byToken[token]
	if !ok {
		return auth.AppleIdentity{}, auth.ErrInvalidToken
	}
	return id, nil
}

func testSessions(t *testing.T) *auth.SessionTokens {
	t.Helper()
	s, err := auth.NewSessionTokens([]byte("test-secret-at-least-thirty-two-bytes!!"), time.Hour)
	if err != nil {
		t.Fatalf("new sessions: %v", err)
	}
	return s
}

// buildServer wires an api.Server where all handlers run against the given tx.
func buildServer(t *testing.T, tx *sql.Tx, appleID auth.AppleIdentity) *api.Server {
	t.Helper()
	return &api.Server{
		UoW: &db.TxUoW{Tx: tx},
		AppleVerifier: &stubApple{byToken: map[string]auth.AppleIdentity{
			"valid-apple-token": appleID,
		}},
		Session: testSessions(t),
	}
}

// --- POST /auth/apple ---

func TestAuthApple_CreatesUserAndReturnsSession(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		name := "Alice"
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "001234.alice", Email: "alice@example.com"})

		body := strings.NewReader(`{"id_token":"valid-apple-token","display_name":"Alice"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/apple", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status: got %d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			SessionToken string `json:"session_token"`
			User         struct {
				ID                   string  `json:"id"`
				DisplayName          *string `json:"display_name"`
				NotificationsEnabled bool    `json:"notifications_enabled"`
			} `json:"user"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.SessionToken == "" {
			t.Error("session_token missing")
		}
		if resp.User.ID == "" {
			t.Error("user.id missing")
		}
		if resp.User.DisplayName == nil || *resp.User.DisplayName != name {
			t.Errorf("display_name: got %v want %q", resp.User.DisplayName, name)
		}
		if !resp.User.NotificationsEnabled {
			t.Error("notifications_enabled should default to true")
		}

		// The returned session token should verify back to the same user id.
		verifiedID, err := testSessions(t).Verify(resp.SessionToken) // same secret
		if err != nil {
			t.Fatalf("session.Verify: %v", err)
		}
		if verifiedID != resp.User.ID {
			t.Errorf("session sub: got %q want %q", verifiedID, resp.User.ID)
		}
	})
}

func TestAuthApple_IsIdempotentForSameSub(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "001234.alice"})

		call := func() string {
			req := httptest.NewRequest(http.MethodPost, "/auth/apple",
				strings.NewReader(`{"id_token":"valid-apple-token"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			srv.Router().ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
			}
			var resp struct {
				User struct{ ID string } `json:"user"`
			}
			_ = json.NewDecoder(w.Body).Decode(&resp)
			return resp.User.ID
		}

		first := call()
		second := call()
		if first != second {
			t.Errorf("expected same user id on repeated sign-in, got %q then %q", first, second)
		}
	})
}

func TestAuthApple_RejectsInvalidAppleToken(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "whatever"})

		req := httptest.NewRequest(http.MethodPost, "/auth/apple",
			strings.NewReader(`{"id_token":"forged-token"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestAuthApple_RejectsMissingIDToken(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "whatever"})

		req := httptest.NewRequest(http.MethodPost, "/auth/apple",
			strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

// --- GET /me + middleware ---

func TestMe_ReturnsAuthenticatedUser(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		name := "Alice"
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "001234.alice"})

		// Sign in to get a token.
		signinReq := httptest.NewRequest(http.MethodPost, "/auth/apple",
			strings.NewReader(`{"id_token":"valid-apple-token","display_name":"Alice"}`))
		signinReq.Header.Set("Content-Type", "application/json")
		signinW := httptest.NewRecorder()
		srv.Router().ServeHTTP(signinW, signinReq)
		if signinW.Code != http.StatusOK {
			t.Fatalf("sign-in: %d", signinW.Code)
		}
		var signinResp struct {
			SessionToken string `json:"session_token"`
			User         struct {
				ID string `json:"id"`
			} `json:"user"`
		}
		_ = json.NewDecoder(signinW.Body).Decode(&signinResp)

		// Now call /me with the token.
		meReq := httptest.NewRequest(http.MethodGet, "/me", nil)
		meReq.Header.Set("Authorization", "Bearer "+signinResp.SessionToken)
		meW := httptest.NewRecorder()
		srv.Router().ServeHTTP(meW, meReq)

		if meW.Code != http.StatusOK {
			t.Fatalf("me: %d body=%s", meW.Code, meW.Body.String())
		}
		var meResp struct {
			ID          string  `json:"id"`
			DisplayName *string `json:"display_name"`
		}
		_ = json.NewDecoder(meW.Body).Decode(&meResp)
		if meResp.ID != signinResp.User.ID {
			t.Errorf("id mismatch: got %q want %q", meResp.ID, signinResp.User.ID)
		}
		if meResp.DisplayName == nil || *meResp.DisplayName != name {
			t.Errorf("display_name: got %v want %q", meResp.DisplayName, name)
		}
	})
}

func TestMe_RejectsMissingAuthHeader(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "whatever"})

		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", w.Code)
		}
	})
}

func TestMe_RejectsBadBearerToken(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "whatever"})

		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("Authorization", "Bearer not-a-real-session")
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestMe_RejectsSessionForUnknownUser(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "whatever"})

		// Issue a token for a user id that doesn't exist in the DB.
		s := testSessions(t)
		tok, err := s.Issue("not-a-real-user-id")
		if err != nil {
			t.Fatalf("issue: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

// prevents "errors imported and not used" if any sql.ErrNoRows path drifts.
var _ = errors.Is
