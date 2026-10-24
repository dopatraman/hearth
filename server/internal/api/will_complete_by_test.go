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

func TestWillCompleteBy_ClaimantCanSet(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-wcb-ok"})
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

		// Set will_complete_by.
		req = httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/will-complete-by",
			strings.NewReader(`{"will_complete_by":"2027-01-01T00:00:00.000Z"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("set: %d body=%s", w.Code, w.Body.String())
		}
		var out map[string]any
		_ = json.NewDecoder(w.Body).Decode(&out)
		if got, _ := out["will_complete_by"].(string); got != "2027-01-01T00:00:00.000Z" {
			t.Errorf("will_complete_by: %v", out["will_complete_by"])
		}
	})
}

func TestWillCompleteBy_UnclaimedReturns409(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-wcb-unclaimed"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		req := httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/will-complete-by",
			strings.NewReader(`{"will_complete_by":"2027-01-01T00:00:00.000Z"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("want 409, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestWillCompleteBy_OverdueFlagDerivedCorrectly(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-wcb-overdue"})
		token, _ := signInHelper(t, srv)
		id := createChore(t, srv, token, "trash")

		req := httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/claim", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		// Past date.
		req = httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/will-complete-by",
			strings.NewReader(`{"will_complete_by":"2020-01-01T00:00:00.000Z"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("set: %d", w.Code)
		}
		var out map[string]any
		_ = json.NewDecoder(w.Body).Decode(&out)
		if ov, _ := out["overdue"].(bool); !ov {
			t.Errorf("expected overdue=true, got %v", out["overdue"])
		}

		// Future date.
		req = httptest.NewRequest(http.MethodPatch, "/chores/"+id+"/will-complete-by",
			strings.NewReader(`{"will_complete_by":"2099-01-01T00:00:00.000Z"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		_ = json.NewDecoder(w.Body).Decode(&out)
		if ov, _ := out["overdue"].(bool); ov {
			t.Errorf("expected overdue=false for future date, got %v", out["overdue"])
		}
	})
}
