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

// --- POST /items ---

func TestItems_Create_ReturnsCreatedItem(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-items-A"})

		token, userID := signInHelper(t, srv)

		req := httptest.NewRequest(http.MethodPost, "/items",
			strings.NewReader(`{"title":"take out trash","notes":"both bins"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status: got %d body=%s", w.Code, w.Body.String())
		}

		var item map[string]any
		if err := json.NewDecoder(w.Body).Decode(&item); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if item["title"] != "take out trash" {
			t.Errorf("title: %v", item["title"])
		}
		if item["notes"] != "both bins" {
			t.Errorf("notes: %v", item["notes"])
		}
		if item["state"] != "open" {
			t.Errorf("state: %v", item["state"])
		}
		if item["creator_id"] != userID {
			t.Errorf("creator_id: got %v want %q", item["creator_id"], userID)
		}
		if item["id"] == "" || item["id"] == nil {
			t.Error("id empty")
		}
	})
}

func TestItems_Create_RejectsEmptyTitle(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-items-B"})
		token, _ := signInHelper(t, srv)

		req := httptest.NewRequest(http.MethodPost, "/items",
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

func TestItems_Create_RequiresAuth(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-items-C"})

		req := httptest.NewRequest(http.MethodPost, "/items",
			strings.NewReader(`{"title":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", w.Code)
		}
	})
}

// --- GET /items ---

func TestItems_List_EmptyForFreshUser(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-items-D"})
		token, _ := signInHelper(t, srv)

		req := httptest.NewRequest(http.MethodGet, "/items", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status: %d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Items) != 0 {
			t.Errorf("expected empty, got %d items", len(resp.Items))
		}
	})
}

func TestItems_List_ReturnsCreatedItems(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-items-E"})
		token, _ := signInHelper(t, srv)

		titles := []string{"trash", "dishes", "mop"}
		for _, title := range titles {
			req := httptest.NewRequest(http.MethodPost, "/items",
				strings.NewReader(`{"title":"`+title+`"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			srv.Router().ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("create %q: %d body=%s", title, w.Code, w.Body.String())
			}
		}

		req := httptest.NewRequest(http.MethodGet, "/items", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("list: %d body=%s", w.Code, w.Body.String())
		}

		var resp struct {
			Items []map[string]any `json:"items"`
		}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Items) != len(titles) {
			t.Fatalf("count: got %d want %d", len(resp.Items), len(titles))
		}
		got := map[string]bool{}
		for _, i := range resp.Items {
			got[i["title"].(string)] = true
		}
		for _, want := range titles {
			if !got[want] {
				t.Errorf("missing title %q", want)
			}
		}
	})
}

func TestItems_List_IsScopedToCaller(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		// Alice gets her own server + sign-in.
		aliceSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-items-alice"})
		aliceToken, _ := signInHelper(t, aliceSrv)

		// Alice creates an item.
		req := httptest.NewRequest(http.MethodPost, "/items",
			strings.NewReader(`{"title":"alice-only"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w := httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("alice create: %d", w.Code)
		}

		// Bob signs in on a server wired to his apple sub.
		bobSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-items-bob"})
		bobToken, _ := signInHelper(t, bobSrv)

		req = httptest.NewRequest(http.MethodGet, "/items", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w = httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("bob list: %d body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Items []map[string]any `json:"items"`
		}
		_ = json.NewDecoder(w.Body).Decode(&resp)
		if len(resp.Items) != 0 {
			t.Errorf("bob should see no items, got %d: %+v", len(resp.Items), resp.Items)
		}
	})
}
