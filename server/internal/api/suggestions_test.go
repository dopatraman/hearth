package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dopatraman/hearth/server/internal/api"
	"github.com/dopatraman/hearth/server/internal/auth"
	"github.com/dopatraman/hearth/server/internal/dbtest"
)

// --- helpers ---

// acceptInvite POSTs /invites/{code}/accept on srv.
func acceptInvite(t *testing.T, srv *api.Server, token, code string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("accept invite: %d body=%s", w.Code, w.Body.String())
	}
}

// seedPair returns (aliceSrv, aliceToken, aliceID, bobSrv, bobToken, bobID)
// with both users in the same list. Each test gets its own apple-sub tag so
// the shared tx doesn't trip on duplicate subs.
func seedPair(t *testing.T, tx *sql.Tx, tag string) (
	aliceSrv *api.Server, aliceToken, aliceID string,
	bobSrv *api.Server, bobToken, bobID string,
) {
	t.Helper()
	aliceSrv = buildServer(t, tx, auth.AppleIdentity{Sub: "sub-" + tag + "-alice"})
	aliceToken, aliceID = signInHelper(t, aliceSrv)
	bobSrv = buildServer(t, tx, auth.AppleIdentity{Sub: "sub-" + tag + "-bob"})
	bobToken, bobID = signInHelper(t, bobSrv)
	code := createInvite(t, aliceSrv, aliceToken)
	acceptInvite(t, bobSrv, bobToken, code)
	return
}

// suggestionResp mirrors the server's suggestionDTO.
type suggestionResp struct {
	ID         string  `json:"id"`
	ChoreID    string  `json:"chore_id"`
	FromUserID string  `json:"from_user_id"`
	ToUserID   string  `json:"to_user_id"`
	Note       *string `json:"note"`
	CreatedAt  string  `json:"created_at"`
}

// postSuggestion POSTs /suggestions, returns (status, body). The caller
// decodes the body itself so a test can both inspect error payloads and
// success DTOs with the same helper.
func postSuggestion(t *testing.T, srv *api.Server, token, choreID, toUserID string, note *string) (int, []byte) {
	t.Helper()
	body := map[string]any{"chore_id": choreID, "to_user_id": toUserID}
	if note != nil {
		body["note"] = *note
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/suggestions", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	return w.Code, w.Body.Bytes()
}

// listChoresWithSuggestion GETs /chores and decodes rows with an extra
// `suggestion` sub-object.
type backlogRowWithSuggestion struct {
	ID         string          `json:"id"`
	Title      string          `json:"title"`
	ClaimedBy  *string         `json:"claimed_by"`
	Suggestion *suggestionResp `json:"suggestion"`
}

func listChoresWithSuggestion(t *testing.T, srv *api.Server, token string) []backlogRowWithSuggestion {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/chores", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list chores: %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Chores []backlogRowWithSuggestion `json:"chores"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.Chores
}

// --- POST /suggestions ---

func TestSuggest_Succeeds(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv, aliceToken, aliceID, _, _, bobID := seedPair(t, tx, "suggest-ok")
		choreID := createChore(t, aliceSrv, aliceToken, "trash")

		note := "when you have a sec"
		code, body := postSuggestion(t, aliceSrv, aliceToken, choreID, bobID, &note)
		if code != http.StatusCreated {
			t.Fatalf("want 201, got %d body=%s", code, string(body))
		}
		var sug suggestionResp
		if err := json.Unmarshal(body, &sug); err != nil {
			t.Fatalf("decode: %v body=%s", err, string(body))
		}
		if sug.ID == "" {
			t.Error("id empty")
		}
		if sug.ChoreID != choreID {
			t.Errorf("chore_id: got %q want %q", sug.ChoreID, choreID)
		}
		if sug.FromUserID != aliceID {
			t.Errorf("from_user_id: got %q want %q", sug.FromUserID, aliceID)
		}
		if sug.ToUserID != bobID {
			t.Errorf("to_user_id: got %q want %q", sug.ToUserID, bobID)
		}
		if sug.Note == nil || *sug.Note != note {
			t.Errorf("note: got %v want %q", sug.Note, note)
		}
	})
}

func TestSuggest_DuplicatePending_Returns409(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv, aliceToken, _, _, _, bobID := seedPair(t, tx, "suggest-dup")
		choreID := createChore(t, aliceSrv, aliceToken, "trash")
		code, _ := postSuggestion(t, aliceSrv, aliceToken, choreID, bobID, nil)
		if code != http.StatusCreated {
			t.Fatalf("first: %d", code)
		}
		code2, body := postSuggestion(t, aliceSrv, aliceToken, choreID, bobID, nil)
		if code2 != http.StatusConflict {
			t.Fatalf("second: want 409, got %d body=%s", code2, string(body))
		}
	})
}

func TestSuggest_SelfSuggest_Returns400(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-suggest-self"})
		token, userID := signInHelper(t, srv)
		choreID := createChore(t, srv, token, "trash")
		code, _ := postSuggestion(t, srv, token, choreID, userID, nil)
		if code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", code)
		}
	})
}

func TestSuggest_RecipientNotInList_Returns400(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-suggest-outsider-A"})
		aliceToken, _ := signInHelper(t, aliceSrv)
		// Bob exists but is not in Alice's list.
		bobSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-suggest-outsider-B"})
		_, bobID := signInHelper(t, bobSrv)
		choreID := createChore(t, aliceSrv, aliceToken, "trash")
		code, _ := postSuggestion(t, aliceSrv, aliceToken, choreID, bobID, nil)
		if code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", code)
		}
	})
}

func TestSuggest_AlreadyClaimedChore_Returns409(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv, aliceToken, _, _, _, bobID := seedPair(t, tx, "suggest-claimed")
		choreID := createChore(t, aliceSrv, aliceToken, "trash")
		// Alice claims it first.
		req := httptest.NewRequest(http.MethodPatch, "/chores/"+choreID+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w := httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("claim: %d", w.Code)
		}
		code, _ := postSuggestion(t, aliceSrv, aliceToken, choreID, bobID, nil)
		if code != http.StatusConflict {
			t.Fatalf("want 409, got %d", code)
		}
	})
}

func TestSuggest_NonexistentChore_Returns404(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv, aliceToken, _, _, _, bobID := seedPair(t, tx, "suggest-404")
		code, _ := postSuggestion(t, aliceSrv, aliceToken, "nope-does-not-exist", bobID, nil)
		if code != http.StatusNotFound {
			t.Fatalf("want 404, got %d", code)
		}
	})
}

// --- GET /chores DTO extension ---

func TestListChores_RecipientSeesSuggestion(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv, aliceToken, aliceID, bobSrv, bobToken, bobID := seedPair(t, tx, "dto-recipient")
		choreID := createChore(t, aliceSrv, aliceToken, "trash")
		if code, _ := postSuggestion(t, aliceSrv, aliceToken, choreID, bobID, nil); code != http.StatusCreated {
			t.Fatalf("suggest: %d", code)
		}

		rows := listChoresWithSuggestion(t, bobSrv, bobToken)
		var found *backlogRowWithSuggestion
		for i := range rows {
			if rows[i].ID == choreID {
				found = &rows[i]
			}
		}
		if found == nil {
			t.Fatalf("chore %q not in bob's backlog", choreID)
		}
		if found.Suggestion == nil {
			t.Fatalf("suggestion field is nil for recipient")
		}
		if found.Suggestion.FromUserID != aliceID {
			t.Errorf("from_user_id: got %q want %q", found.Suggestion.FromUserID, aliceID)
		}
		if found.Suggestion.ToUserID != bobID {
			t.Errorf("to_user_id: got %q want %q", found.Suggestion.ToUserID, bobID)
		}
	})
}

func TestListChores_SenderSeesSuggestion(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv, aliceToken, _, _, _, bobID := seedPair(t, tx, "dto-sender")
		choreID := createChore(t, aliceSrv, aliceToken, "trash")
		if code, _ := postSuggestion(t, aliceSrv, aliceToken, choreID, bobID, nil); code != http.StatusCreated {
			t.Fatalf("suggest: %d", code)
		}
		rows := listChoresWithSuggestion(t, aliceSrv, aliceToken)
		var found *backlogRowWithSuggestion
		for i := range rows {
			if rows[i].ID == choreID {
				found = &rows[i]
			}
		}
		if found == nil {
			t.Fatalf("chore not in alice's backlog")
		}
		if found.Suggestion == nil {
			t.Fatalf("sender does not see suggestion field (needed for 'Suggested to Bob' subtitle)")
		}
	})
}

func TestListChores_UnsuggestedChoreHasNullSuggestion(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-dto-null"})
		token, _ := signInHelper(t, srv)
		choreID := createChore(t, srv, token, "trash")
		rows := listChoresWithSuggestion(t, srv, token)
		for _, r := range rows {
			if r.ID == choreID && r.Suggestion != nil {
				t.Errorf("expected suggestion: null, got %+v", r.Suggestion)
			}
		}
	})
}

// --- Claim path ---

func TestClaim_RemovesSuggestionFromSubsequentDTO(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv, aliceToken, _, bobSrv, bobToken, bobID := seedPair(t, tx, "claim-removes-sug")
		choreID := createChore(t, aliceSrv, aliceToken, "trash")
		if code, _ := postSuggestion(t, aliceSrv, aliceToken, choreID, bobID, nil); code != http.StatusCreated {
			t.Fatalf("suggest: %d", code)
		}

		// Bob claims.
		req := httptest.NewRequest(http.MethodPatch, "/chores/"+choreID+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("claim: %d body=%s", w.Code, w.Body.String())
		}

		// Both backlog views see suggestion: null.
		for _, view := range []struct {
			name  string
			srv   *api.Server
			token string
		}{
			{"bob", bobSrv, bobToken},
			{"alice", aliceSrv, aliceToken},
		} {
			rows := listChoresWithSuggestion(t, view.srv, view.token)
			for _, r := range rows {
				if r.ID == choreID && r.Suggestion != nil {
					t.Errorf("%s view still has suggestion after claim: %+v", view.name, r.Suggestion)
				}
			}
		}
	})
}

// Compile-time guard — fail loudly if the test file is accidentally built
// with the wrong fmt import absent (makes the helper usable from debug prints).
var _ = fmt.Sprint
