package api_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dopatraman/hearth/server/internal/auth"
	"github.com/dopatraman/hearth/server/internal/dbtest"
)

func TestRateLimit_AuthApple_ReturnsTooManyAfterBurst(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-ratelimit"})
		mux := srv.Router()
		seen429 := false
		for i := 0; i < 20; i++ {
			req := httptest.NewRequest(http.MethodPost, "/auth/apple",
				strings.NewReader(`{"id_token":"valid-apple-token"}`))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = "203.0.113.42:1234"
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			if w.Code == http.StatusTooManyRequests {
				seen429 = true
				break
			}
		}
		if !seen429 {
			t.Error("expected 429 after burst exhausted")
		}
	})
}

func TestRateLimit_AuthApple_DistinctIPsDontCollide(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-ratelimit-ips"})
		mux := srv.Router()

		for i := 0; i < 20; i++ {
			req := httptest.NewRequest(http.MethodPost, "/auth/apple",
				strings.NewReader(`{"id_token":"valid-apple-token"}`))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = "203.0.113.10:1234"
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
		}

		req := httptest.NewRequest(http.MethodPost, "/auth/apple",
			strings.NewReader(`{"id_token":"valid-apple-token"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.11:1234"
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Errorf("fresh IP got 429, want 2xx (got %d)", w.Code)
		}
	})
}
