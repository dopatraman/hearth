package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dopatraman/hearth/server/internal/api"
)

// createInvite POSTs /invites and returns the issued code.
func createInvite(t *testing.T, srv *api.Server, token string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/invites", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("createInvite: %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("createInvite decode: %v", err)
	}
	return resp.Code
}

// listChoreTitles GETs /chores and returns the titles in order.
func listChoreTitles(t *testing.T, srv *api.Server, token string) []string {
	t.Helper()
	rows := listChores(t, srv, token)
	titles := make([]string, 0, len(rows))
	for _, c := range rows {
		titles = append(titles, c.Title)
	}
	return titles
}

type backlogRow struct {
	Title        string  `json:"title"`
	SnoozedUntil *string `json:"snoozed_until"`
}

// listChores GETs /chores and returns the title + snoozed_until for each row.
func listChores(t *testing.T, srv *api.Server, token string) []backlogRow {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/chores", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("listChores: %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Chores []backlogRow `json:"chores"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("listChores decode: %v", err)
	}
	return resp.Chores
}

// signInHelper POSTs /auth/apple with the stub-recognized token and returns
// the (sessionToken, userID). Centralizes the pattern across chores/me tests.
func signInHelper(t *testing.T, srv *api.Server) (string, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/auth/apple",
		strings.NewReader(`{"id_token":"valid-apple-token"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("sign-in failed: %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		SessionToken string `json:"session_token"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("sign-in decode: %v", err)
	}
	return resp.SessionToken, resp.User.ID
}
