package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dopatraman/hearth/server/internal/api"
	"github.com/dopatraman/hearth/server/internal/auth"
	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/dbtest"
	"github.com/dopatraman/hearth/server/internal/notifications"
)

type pushCall struct {
	tokens  []db.TokensForListRow
	payload notifications.Payload
}

type fakePusher struct {
	mu       sync.Mutex
	calls    []pushCall
	ch       chan struct{}
	goneSet  map[string]bool
}

func newFakePusher() *fakePusher {
	return &fakePusher{ch: make(chan struct{}, 32), goneSet: map[string]bool{}}
}

func (f *fakePusher) Push(_ context.Context, tokens []db.TokensForListRow, p notifications.Payload) []notifications.PushResult {
	f.mu.Lock()
	tokensCopy := append([]db.TokensForListRow(nil), tokens...)
	f.calls = append(f.calls, pushCall{tokens: tokensCopy, payload: p})
	results := make([]notifications.PushResult, 0, len(tokens))
	for _, t := range tokens {
		results = append(results, notifications.PushResult{
			Token:    t.Token,
			Platform: t.Platform,
			Gone:     f.goneSet[t.Token],
		})
	}
	f.mu.Unlock()
	f.ch <- struct{}{}
	return results
}

func (f *fakePusher) wait(t *testing.T) pushCall {
	t.Helper()
	select {
	case <-f.ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for push")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[len(f.calls)-1]
}

func (f *fakePusher) markGone(token string) {
	f.mu.Lock()
	f.goneSet[token] = true
	f.mu.Unlock()
}

func buildServerWithPusher(t *testing.T, tx *sql.Tx, appleID auth.AppleIdentity, p notifications.Pusher) *api.Server {
	t.Helper()
	uow := &db.TxUoW{Tx: tx}
	return &api.Server{
		UoW: uow,
		AppleVerifier: &stubApple{byToken: map[string]auth.AppleIdentity{
			"valid-apple-token": appleID,
		}},
		Session: testSessions(t),
		Dispatcher: &notifications.Dispatcher{
			UoW:    uow,
			Pusher: p,
		},
	}
}

// Register a device via POST /devices.
func registerDevice(t *testing.T, srv *api.Server, token, platform, deviceToken string) {
	t.Helper()
	body := strings.NewReader(`{"token":"` + deviceToken + `","platform":"` + platform + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/devices", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("registerDevice: %d body=%s", w.Code, w.Body.String())
	}
}

func setNotifications(t *testing.T, srv *api.Server, token string, enabled bool) {
	t.Helper()
	body := `{"enabled":false}`
	if enabled {
		body = `{"enabled":true}`
	}
	req := httptest.NewRequest(http.MethodPut, "/me/notifications", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("setNotifications: %d body=%s", w.Code, w.Body.String())
	}
}

func TestPush_CreateChoreNotifiesOtherListMembers(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		pusher := newFakePusher()
		aliceSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-push-alice"}, pusher)
		aliceToken, _ := signInHelper(t, aliceSrv)

		code := createInvite(t, aliceSrv, aliceToken)

		bobSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-push-bob"}, pusher)
		bobToken, _ := signInHelper(t, bobSrv)
		req := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("accept: %d", w.Code)
		}

		registerDevice(t, bobSrv, bobToken, "ios", "bob-ios-token")
		registerDevice(t, aliceSrv, aliceToken, "ios", "alice-ios-token")

		// Alice creates — Bob should be notified, Alice should NOT.
		req = httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"trash"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w = httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: %d", w.Code)
		}

		call := pusher.wait(t)
		if len(call.tokens) != 1 {
			t.Fatalf("recipients: got %d want 1", len(call.tokens))
		}
		if call.tokens[0].Token != "bob-ios-token" {
			t.Errorf("recipient: got %q want bob-ios-token", call.tokens[0].Token)
		}
		if call.payload.Alert != "" {
			t.Errorf("create should be silent, got alert %q", call.payload.Alert)
		}
	})
}

func TestPush_ClaimAndCompleteAlertsNameActorAndChore(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		pusher := newFakePusher()
		aliceSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-push-named-alice"}, pusher)
		aliceToken, _ := signInHelper(t, aliceSrv)
		code := createInvite(t, aliceSrv, aliceToken)

		bobSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-push-named-bob"}, pusher)
		req := httptest.NewRequest(http.MethodPost, "/auth/apple",
			strings.NewReader(`{"id_token":"valid-apple-token","display_name":"Bob Smith"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("bob sign-in: %d", w.Code)
		}
		var signIn struct {
			SessionToken string `json:"session_token"`
		}
		if err := json.NewDecoder(w.Body).Decode(&signIn); err != nil {
			t.Fatalf("decode sign-in: %v", err)
		}
		bobToken := signIn.SessionToken

		req = httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w = httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("accept: %d", w.Code)
		}
		registerDevice(t, aliceSrv, aliceToken, "ios", "alice-ios-token")

		req = httptest.NewRequest(http.MethodPost, "/chores", strings.NewReader(`{"title":"Look at flights"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w = httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: %d", w.Code)
		}
		var chore struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(w.Body).Decode(&chore); err != nil {
			t.Fatalf("decode chore: %v", err)
		}
		if call := pusher.wait(t); call.payload.Alert != "" {
			t.Errorf("create alert: got %q want silent", call.payload.Alert)
		}

		cases := []struct {
			method, path, want string
		}{
			{http.MethodPatch, "/chores/" + chore.ID + "/claim", `Bob claimed "Look at flights"`},
			{http.MethodPost, "/chores/" + chore.ID + "/complete", `Bob completed "Look at flights"`},
		}
		for _, c := range cases {
			req = httptest.NewRequest(c.method, c.path, nil)
			req.Header.Set("Authorization", "Bearer "+bobToken)
			w = httptest.NewRecorder()
			bobSrv.Router().ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("%s %s: %d", c.method, c.path, w.Code)
			}
			if call := pusher.wait(t); call.payload.Alert != c.want {
				t.Errorf("%s alert: got %q want %q", c.path, call.payload.Alert, c.want)
			}
		}
	})
}

func TestPush_NotificationsDisabledUserSkipped(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		pusher := newFakePusher()
		aliceSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-push-off-alice"}, pusher)
		aliceToken, _ := signInHelper(t, aliceSrv)
		code := createInvite(t, aliceSrv, aliceToken)

		bobSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-push-off-bob"}, pusher)
		bobToken, _ := signInHelper(t, bobSrv)
		req := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("accept: %d", w.Code)
		}

		registerDevice(t, bobSrv, bobToken, "ios", "bob-off-token")
		setNotifications(t, bobSrv, bobToken, false)

		req = httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"trash"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w = httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: %d", w.Code)
		}

		// Dispatcher runs with no recipients → no Push call expected.
		select {
		case <-pusher.ch:
			t.Fatal("expected no push for disabled user")
		case <-time.After(300 * time.Millisecond):
		}
	})
}

func TestPush_MultipleDevicesPerUser(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		pusher := newFakePusher()
		aliceSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-push-multi-alice"}, pusher)
		aliceToken, _ := signInHelper(t, aliceSrv)
		code := createInvite(t, aliceSrv, aliceToken)

		bobSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-push-multi-bob"}, pusher)
		bobToken, _ := signInHelper(t, bobSrv)
		req := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("accept: %d", w.Code)
		}

		registerDevice(t, bobSrv, bobToken, "ios", "bob-phone")
		registerDevice(t, bobSrv, bobToken, "watchos", "bob-watch")

		req = httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"trash"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w = httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: %d", w.Code)
		}

		call := pusher.wait(t)
		if len(call.tokens) != 2 {
			t.Fatalf("recipients: got %d want 2", len(call.tokens))
		}
	})
}

func TestPush_GoneTokenGetsDeleted(t *testing.T) {
	dbtest.WithTx(t, func(ctx context.Context, tx *sql.Tx) {
		pusher := newFakePusher()
		aliceSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-push-gone-alice"}, pusher)
		aliceToken, _ := signInHelper(t, aliceSrv)
		code := createInvite(t, aliceSrv, aliceToken)

		bobSrv := buildServerWithPusher(t, tx, auth.AppleIdentity{Sub: "sub-push-gone-bob"}, pusher)
		bobToken, bobID := signInHelper(t, bobSrv)
		req := httptest.NewRequest(http.MethodPost, "/invites/"+code+"/accept", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)
		w := httptest.NewRecorder()
		bobSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("accept: %d", w.Code)
		}

		registerDevice(t, bobSrv, bobToken, "ios", "bob-gone-token")
		pusher.markGone("bob-gone-token")

		req = httptest.NewRequest(http.MethodPost, "/chores",
			strings.NewReader(`{"title":"trash"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		w = httptest.NewRecorder()
		aliceSrv.Router().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create: %d", w.Code)
		}

		pusher.wait(t)

		// Give the goroutine a tick to run the delete after Push returns.
		deadline := time.Now().Add(1 * time.Second)
		for time.Now().Before(deadline) {
			var n int
			if err := tx.QueryRow(
				`SELECT COUNT(*) FROM device_tokens WHERE token = ?`,
				"bob-gone-token",
			).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n == 0 {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Errorf("gone token still present for user %s", bobID)

		_ = json.RawMessage{}
	})
}
