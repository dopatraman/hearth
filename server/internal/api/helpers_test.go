package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dopatraman/hearth/server/internal/api"
)

// signInHelper POSTs /auth/apple with the stub-recognized token and returns
// the (sessionToken, userID). Centralizes the pattern across items/me tests.
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
