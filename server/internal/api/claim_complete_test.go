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

// Shortcut: creates a chore via POST /chores on srv, returns its id.
func createChore(t *testing.T, srv *api.Server, token, title string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/chores",
		strings.NewReader(`{"title":"`+title+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("createChore %q: %d body=%s", title, w.Code, w.Body.String())
	}
	var out struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(w.Body).Decode(&out)
	return out.ID
}

// --- claim ---

func TestClaim_Succeeds(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-claim-ok"})
		token, userID := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		req := httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
		}
		var chore map[string]any
		_ = json.NewDecoder(w.Body).Decode(&chore)
		if claimedBy, _ := chore["claimed_by"].(string); claimedBy != userID {
			t.Errorf("claimed_by: got %v want %q", chore["claimed_by"], userID)
		}
	})
}

func TestClaim_SecondClaimReturns409(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		// Alice creates, invites, Bob accepts — both in same list now.
		aliceSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-claim-alice"})
		aliceToken, _ := signInHelper(t, aliceSrv)
		id := createChore(t, aliceSrv, aliceToken, "trash")

		code := createInvite(t, aliceSrv, aliceToken)

		bobSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-claim-bob"})
		bobToken, _ := signInHelper(t, bobSrv)

		// Bob accepts invite.
		req := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("accept: %d", w.Code)
		}

		// Alice claims first.
		req = httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w = httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("alice claim: %d body=%s", w.Code, w.Body.String())
		}

		// Bob tries — should 409.
		req = httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w = httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("want 409, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

// --- complete ---

func TestComplete_ClaimantCanComplete(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-complete-ok"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		// Claim first.
		req := httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("claim: %d", w.Code)
		}

		// Complete.
		req = httptest.NewRequest(http.MethodPost, "/chores/"+id+"/complete", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("complete: %d body=%s", w.Code, w.Body.String())
		}

		var chore map[string]any
		_ = json.NewDecoder(w.Body).Decode(&chore)
		if chore["state"] != "completed" {
			t.Errorf("state: got %v", chore["state"])
		}
	})
}

func TestComplete_UnclaimedReturns409(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-complete-unclaimed"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		req := httptest.NewRequest(http.MethodPost, "/chores/"+id+"/complete", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("want 409, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestComplete_RemovesFromBacklog_AppearsInGraveyard(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-grave-flow"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		// Claim + complete.
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
			t.Fatal("complete")
		}

		// Backlog is now empty.
		if titles := listChoreTitles(t, srv, token); len(titles) != 0 {
			t.Errorf("backlog should be empty after complete, got %v", titles)
		}

		// Graveyard contains it.
		req = httptest.NewRequest(http.MethodGet, "/chores/graveyard", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("graveyard: %d", w.Code)
		}
		var resp struct {
			Chores []struct{ Title string } `json:"chores"`
		}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Chores) != 1 || resp.Chores[0].Title != "trash" {
			t.Errorf("graveyard: %+v", resp.Chores)
		}
	})
}

func TestGraveyardSearch_FiltersByTitle(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-grave-search"})
		token, _ := signInHelper(t, srv)

		for _, title := range []string{"Take out trash", "Dishes", "Mop"} {
			id := createChore(t, srv, token, title)

			req := httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			srv.Router().ServeHTTP(w, req)

			req = httptest.NewRequest(http.MethodPost, "/chores/"+id+"/complete", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w = httptest.NewRecorder()
			srv.Router().ServeHTTP(w, req)
		}

		req := httptest.NewRequest(http.MethodGet, "/chores/graveyard?q=TRASH", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("search: %d", w.Code)
		}
		var resp struct {
			Chores []struct{ Title string } `json:"chores"`
		}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Chores) != 1 || resp.Chores[0].Title != "Take out trash" {
			t.Errorf("search TRASH: %+v", resp.Chores)
		}
	})
}
