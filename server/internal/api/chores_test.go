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

// --- POST /chores ---

func TestChores_Create_ReturnsCreatedChore(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-chores-A"})

		token, userID := signInHelper(t, srv)

		req := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"take out trash","notes":"both bins"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status: got %d body=%s", w.Code, w.Body.String())
		}

		var chore map[string]any
		if err := json.NewDecoder(w.Body).Decode(&chore); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if chore["title"] != "take out trash" {
			t.Errorf("title: %v", chore["title"])
		}
		if chore["notes"] != "both bins" {
			t.Errorf("notes: %v", chore["notes"])
		}
		if chore["state"] != "open" {
			t.Errorf("state: %v", chore["state"])
		}
		if chore["creator_id"] != userID {
			t.Errorf("creator_id: got %v want %q", chore["creator_id"], userID)
		}
		if chore["id"] == "" || chore["id"] == nil {
			t.Error("id empty")
		}
	})
}

func TestChores_Create_WithClaim_ReturnsClaimedChore(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-chores-create-claim"})
		token, userID := signInHelper(t, srv)

		req := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"descale the kettle","claim":true}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status: got %d body=%s", w.Code, w.Body.String())
		}

		var chore map[string]any
		if err := json.NewDecoder(w.Body).Decode(&chore); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if chore["claimed_by"] != userID {
			t.Errorf("claimed_by: got %v want %q", chore["claimed_by"], userID)
		}
		if chore["claimed_at"] == nil {
			t.Error("claimed_at: nil")
		}
	})
}

func TestChores_Create_WithClaimAndWhen_ReturnsScheduledChore(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-chores-when"})
		token, userID := signInHelper(t, srv)

		req := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"vacuum","claim":true,"will_complete_by":"2026-10-28T23:59:59.999Z"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status: got %d body=%s", w.Code, w.Body.String())
		}

		var chore map[string]any
		if err := json.NewDecoder(w.Body).Decode(&chore); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if chore["claimed_by"] != userID {
			t.Errorf("claimed_by: got %v want %q", chore["claimed_by"], userID)
		}
		if chore["will_complete_by"] != "2026-10-28T23:59:59.999Z" {
			t.Errorf("will_complete_by: got %v", chore["will_complete_by"])
		}
	})
}

func TestChores_Create_WhenWithoutClaim_Returns400(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-chores-when-no-claim"})
		token, _ := signInHelper(t, srv)

		req := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"vacuum","will_complete_by":"2026-10-28T23:59:59.999Z"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestChores_Create_WithoutClaim_ReturnsUnclaimedChore(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-chores-create-no-claim"})
		token, _ := signInHelper(t, srv)

		req := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"vacuum"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status: got %d body=%s", w.Code, w.Body.String())
		}

		var chore map[string]any
		if err := json.NewDecoder(w.Body).Decode(&chore); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if chore["claimed_by"] != nil {
			t.Errorf("claimed_by: got %v want nil", chore["claimed_by"])
		}
	})
}

func TestChores_Create_RejectsEmptyTitle(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-chores-B"})
		token, _ := signInHelper(t, srv)

		req := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":""}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestChores_Create_RequiresAuth(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-chores-C"})

		req := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", w.Code)
		}
	})
}

// --- GET /chores ---

func TestChores_List_EmptyForFreshUser(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-chores-D"})
		token, _ := signInHelper(t, srv)

		req := httptest.NewRequest(http.MethodGet, "/chores", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			Chores []map[string]any `json:"chores"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Chores) != 0 {
			t.Errorf("expected empty, got %d chores", len(resp.Chores))
		}
	})
}

func TestChores_List_ReturnsCreatedChores(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-chores-E"})
		token, _ := signInHelper(t, srv)

		titles := []string{"trash", "dishes", "mop"}
		for _, title := range titles {
			req := httptest.NewRequest(http.MethodPost, "/chores",
				strings.NewReader(`{"title":"`+title+`"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			srv.Router().ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("create %q: %d body=%s", title, w.Code, w.Body.String())
			}
		}

		req := httptest.NewRequest(http.MethodGet, "/chores", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("list: %d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			Chores []map[string]any `json:"chores"`
		}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Chores) != len(titles) {
			t.Fatalf("count: got %d want %d", len(resp.Chores), len(titles))
		}
		got := map[string]bool{}
		for _, c := range resp.Chores {
			got[c["title"].(string)] = true
		}
		for _, want := range titles {
			if !got[want] {
				t.Errorf("missing title %q", want)
			}
		}
	})
}

func TestChores_List_IsScopedToCaller(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		// Alice gets her own server + sign-in.
		aliceSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-chores-alice"})
		aliceToken, _ := signInHelper(t, aliceSrv)

		// Alice creates a chore.
		req := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"alice-only"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w := httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("alice create: %d", w.Code)
		}

		// Bob signs in on a server wired to his apple sub.
		bobSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-chores-bob"})
		bobToken, _ := signInHelper(t, bobSrv)

		req = httptest.NewRequest(http.MethodGet, "/chores", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w = httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("bob list: %d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Chores []map[string]any `json:"chores"`
		}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Chores) != 0 {
			t.Errorf("bob should see no chores, got %d: %+v", len(resp.Chores), resp.Chores)
		}
	})
}
