package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dopatraman/hearth/server/internal/api"
	"github.com/dopatraman/hearth/server/internal/auth"
	"github.com/dopatraman/hearth/server/internal/dbtest"
)

func TestRequestLogger_EmitsOneLinePerRequest(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(api.NewContextSlogHandler(slog.NewJSONHandler(&buf, nil))))
	defer slog.SetDefault(prev)

	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-logtest"})
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
	})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// At minimum the http line; may also include migration/startup logs but
	// those don't emit through the test request path.
	var httpLine map[string]any
	for _, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			continue
		}
		if m["msg"] == "http" {
			httpLine = m
		}
	}
	if httpLine == nil {
		t.Fatalf("no http log line found: %s", buf.String())
	}
	for _, key := range []string{"method", "path", "status", "latency_ms", "request_id", "remote_ip"} {
		if _, ok := httpLine[key]; !ok {
			t.Errorf("http log missing %q: %v", key, httpLine)
		}
	}
	if httpLine["path"] != "/healthz" {
		t.Errorf("path: got %v want /healthz", httpLine["path"])
	}
}

func TestContextSlogHandler_AttachesUserIDFromAuthedRequest(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(api.NewContextSlogHandler(slog.NewJSONHandler(&buf, nil))))
	defer slog.SetDefault(prev)

	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-logtest-user"})
		token, userID := signInHelper(t, srv)
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("me: %d", w.Code)
		}

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		var sawUser bool
		for _, line := range lines {
			var m map[string]any
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				continue
			}
			if m["msg"] == "http" {
				if m["user_id"] == userID {
					sawUser = true
				}
			}
		}
		if !sawUser {
			t.Errorf("expected user_id=%s on authed /me log line; got: %s", userID, buf.String())
		}
	})
}
