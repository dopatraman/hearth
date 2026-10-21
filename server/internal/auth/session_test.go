package auth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/dopatraman/hearth/server/internal/auth"
)

func newSessions(t *testing.T, ttl time.Duration) *auth.SessionTokens {
	t.Helper()
	s, err := auth.NewSessionTokens([]byte("a-secret-at-least-thirty-two-bytes-long!!"), ttl)
	if err != nil {
		t.Fatalf("new sessions: %v", err)
	}
	return s
}

func TestNewSessionTokens_RequiresMinSecret(t *testing.T) {
	if _, err := auth.NewSessionTokens([]byte("too-short"), time.Hour); err == nil {
		t.Fatal("expected error for short secret, got nil")
	}
}

func TestSession_IssueThenVerifyRoundtrips(t *testing.T) {
	s := newSessions(t, 1*time.Hour)

	tok, err := s.Issue("user-id-123")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	sub, err := s.Verify(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if sub != "user-id-123" {
		t.Errorf("sub: got %q, want %q", sub, "user-id-123")
	}
}

func TestSession_RejectsExpired(t *testing.T) {
	s := newSessions(t, 1*time.Hour)

	// Issue at T0.
	t0 := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	issuedAtT0 := s.WithClock(func() time.Time { return t0 })
	tok, err := issuedAtT0.Issue("u")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	// Verify at T0 + 2h (past 1h ttl + 60s leeway).
	verifierAtT2h := s.WithClock(func() time.Time { return t0.Add(2 * time.Hour) })
	if _, err := verifierAtT2h.Verify(tok); !errors.Is(err, auth.ErrInvalidSession) {
		t.Fatalf("want ErrInvalidSession, got %v", err)
	}
}

func TestSession_RejectsDifferentSecret(t *testing.T) {
	issuer, _ := auth.NewSessionTokens([]byte("secret-one-thirty-two-bytes-plus-aaaaa"), time.Hour)
	verifier, _ := auth.NewSessionTokens([]byte("secret-two-thirty-two-bytes-plus-bbbbb"), time.Hour)

	tok, err := issuer.Issue("u")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := verifier.Verify(tok); !errors.Is(err, auth.ErrInvalidSession) {
		t.Fatalf("want ErrInvalidSession, got %v", err)
	}
}

func TestSession_RejectsMalformed(t *testing.T) {
	s := newSessions(t, time.Hour)
	if _, err := s.Verify("not-a-jwt-at-all"); !errors.Is(err, auth.ErrInvalidSession) {
		t.Fatalf("want ErrInvalidSession, got %v", err)
	}
}
