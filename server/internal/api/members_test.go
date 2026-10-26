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

type memberDTO struct {
	ID          string  `json:"id"`
	DisplayName *string `json:"display_name"`
}

type membersResponse struct {
	Members []memberDTO `json:"members"`
}

// signInWithName posts /auth/apple with a display_name and returns (token, userID).
func signInWithName(t *testing.T, srv *api.Server, name string) (string, string) {
	t.Helper()
	body := `{"id_token":"valid-apple-token","display_name":"` + name + `"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/apple", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("sign-in with name: %d body=%s", w.Code, w.Body.String())
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

func getMembers(t *testing.T, srv *api.Server, token string) membersResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/lists/mine/members", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("getMembers: %d body=%s", w.Code, w.Body.String())
	}
	var resp membersResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("members decode: %v", err)
	}
	return resp
}

func TestListMembers_ReturnsAllListMembers(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		aliceSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-m-alice"})
		aliceToken, aliceID := signInWithName(t, aliceSrv, "Alice")

		// Alice creates a chore to force list provisioning.
		req := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"something"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w := httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("alice create chore: %d", w.Code)
		}

		// Alice invites Bob.
		code := createInvite(t, aliceSrv, aliceToken)

		bobSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-m-bob"})
		bobToken, bobID := signInWithName(t, bobSrv, "Bob")

		acceptReq := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		acceptReq.Header.Set("Authorization", "Bearer "+bobToken)
		acceptW := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(acceptW, acceptReq)
		if acceptW.Code != http.StatusOK {
			t.Fatalf("bob accept: %d body=%s", acceptW.Code, acceptW.Body.String())
		}

		// Both Alice and Bob should see each other.
		resp := getMembers(t, aliceSrv, aliceToken)
		if len(resp.Members) != 2 {
			t.Fatalf("alice members: got %d want 2 (%+v)", len(resp.Members), resp.Members)
		}
		gotIDs := map[string]*string{}
		for _, m := range resp.Members {
			gotIDs[m.ID] = m.DisplayName
		}
		if dn, ok := gotIDs[aliceID]; !ok || dn == nil || *dn != "Alice" {
			t.Errorf("alice member missing/incorrect: %+v", gotIDs)
		}
		if dn, ok := gotIDs[bobID]; !ok || dn == nil || *dn != "Bob" {
			t.Errorf("bob member missing/incorrect: %+v", gotIDs)
		}
	})
}

func TestListMembers_RequiresAuth(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-m-noauth"})

		req := httptest.NewRequest(http.MethodGet, "/lists/mine/members", nil)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestListMembers_IsolatedByList(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		// Alice provisions her own list.
		aliceSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-m-iso-alice"})
		aliceToken, aliceID := signInWithName(t, aliceSrv, "Alice")
		req := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"alice chore"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w := httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("alice create chore: %d", w.Code)
		}

		// Carol provisions her own, separate list (never accepts Alice's invite).
		carolSrv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-m-iso-carol"})
		carolToken, carolID := signInWithName(t, carolSrv, "Carol")
		carolReq := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"carol chore"}`))
		carolReq.Header.Set("Content-Type", "application/json")
		carolReq.Header.Set("Authorization", "Bearer "+carolToken)
		carolW := httptest.NewRecorder()
		carolSrv.Router().ServeHTTP(carolW, carolReq)
		if carolW.Code != http.StatusCreated {
			t.Fatalf("carol create chore: %d", carolW.Code)
		}

		resp := getMembers(t, aliceSrv, aliceToken)
		for _, m := range resp.Members {
			if m.ID == carolID {
				t.Errorf("alice members should NOT include carol (%s); got %+v", carolID, resp.Members)
			}
		}
		if len(resp.Members) != 1 {
			t.Errorf("alice members: got %d want 1 (self only)", len(resp.Members))
		}
		if len(resp.Members) == 1 && resp.Members[0].ID != aliceID {
			t.Errorf("alice sole member should be self (%s); got %s", aliceID, resp.Members[0].ID)
		}
	})
}

func TestListMembers_IncludesSelf(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-m-self"})
		token, userID := signInWithName(t, srv, "Solo")

		// Force list provisioning.
		req := httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"only mine"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create chore: %d", w.Code)
		}

		resp := getMembers(t, srv, token)
		if len(resp.Members) != 1 || resp.Members[0].ID != userID {
			t.Fatalf("expected sole member = self (%s); got %+v", userID, resp.Members)
		}
		if resp.Members[0].DisplayName == nil || *resp.Members[0].DisplayName != "Solo" {
			t.Errorf("self display name: got %+v want Solo", resp.Members[0].DisplayName)
		}
	})
}

func TestListMembers_EmptyForUserWithNoMemberships(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		srv := buildServer(t, tx, auth.AppleIdentity{Sub: "sub-m-empty"})
		token, _ := signInWithName(t, srv, "Nomad")

		// Nomad signs in but never creates a chore or accepts an invite — no
		// list membership. Endpoint should return an empty array, not 404.
		resp := getMembers(t, srv, token)
		if resp.Members == nil {
			t.Fatal("members should be [] not nil")
		}
		if len(resp.Members) != 0 {
			t.Errorf("want 0 members, got %+v", resp.Members)
		}
	})
}
