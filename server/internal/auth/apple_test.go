package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/dopatraman/hearth/server/internal/auth"
	"github.com/golang-jwt/jwt/v5"
)

const testAudience = "com.dopatraman.hearth"

func newRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gen rsa: %v", err)
	}
	return key
}

func keyFuncFor(key *rsa.PrivateKey) jwt.Keyfunc {
	return func(*jwt.Token) (any, error) { return &key.PublicKey, nil }
}

func signRS256(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func fixedNow() func() time.Time {
	frozen := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return frozen }
}

func TestAppleVerifier_AcceptsValidToken(t *testing.T) {
	key := newRSAKey(t)
	now := fixedNow()
	v := auth.NewAppleVerifierForTest(testAudience, keyFuncFor(key), now)

	tok := signRS256(t, key, jwt.MapClaims{
		"iss":   "https://appleid.apple.com",
		"aud":   testAudience,
		"sub":   "001234.56781234abcd",
		"email": "alice@example.com",
		"iat":   now().Unix(),
		"exp":   now().Add(10 * time.Minute).Unix(),
	})

	id, err := v.Verify(context.Background(), tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if id.Sub != "001234.56781234abcd" {
		t.Errorf("sub: got %q, want %q", id.Sub, "001234.56781234abcd")
	}
	if id.Email != "alice@example.com" {
		t.Errorf("email: got %q, want %q", id.Email, "alice@example.com")
	}
}

func TestAppleVerifier_RejectsExpired(t *testing.T) {
	key := newRSAKey(t)
	now := fixedNow()
	v := auth.NewAppleVerifierForTest(testAudience, keyFuncFor(key), now)

	tok := signRS256(t, key, jwt.MapClaims{
		"iss": "https://appleid.apple.com",
		"aud": testAudience,
		"sub": "x",
		"iat": now().Add(-20 * time.Minute).Unix(),
		"exp": now().Add(-5 * time.Minute).Unix(), // 5 min ago, past 60s leeway
	})

	_, err := v.Verify(context.Background(), tok)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("want ErrInvalidToken, got %v", err)
	}
}

func TestAppleVerifier_RejectsWrongAudience(t *testing.T) {
	key := newRSAKey(t)
	now := fixedNow()
	v := auth.NewAppleVerifierForTest(testAudience, keyFuncFor(key), now)

	tok := signRS256(t, key, jwt.MapClaims{
		"iss": "https://appleid.apple.com",
		"aud": "com.someone.else",
		"sub": "x",
		"iat": now().Unix(),
		"exp": now().Add(10 * time.Minute).Unix(),
	})

	_, err := v.Verify(context.Background(), tok)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("want ErrInvalidToken, got %v", err)
	}
}

func TestAppleVerifier_RejectsWrongIssuer(t *testing.T) {
	key := newRSAKey(t)
	now := fixedNow()
	v := auth.NewAppleVerifierForTest(testAudience, keyFuncFor(key), now)

	tok := signRS256(t, key, jwt.MapClaims{
		"iss": "https://evil.example.com",
		"aud": testAudience,
		"sub": "x",
		"iat": now().Unix(),
		"exp": now().Add(10 * time.Minute).Unix(),
	})

	_, err := v.Verify(context.Background(), tok)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("want ErrInvalidToken, got %v", err)
	}
}

func TestAppleVerifier_RejectsBadSignature(t *testing.T) {
	actualKey := newRSAKey(t)
	otherKey := newRSAKey(t) // signer != resolver
	now := fixedNow()
	v := auth.NewAppleVerifierForTest(testAudience, keyFuncFor(actualKey), now)

	tok := signRS256(t, otherKey, jwt.MapClaims{
		"iss": "https://appleid.apple.com",
		"aud": testAudience,
		"sub": "x",
		"iat": now().Unix(),
		"exp": now().Add(10 * time.Minute).Unix(),
	})

	_, err := v.Verify(context.Background(), tok)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("want ErrInvalidToken, got %v", err)
	}
}

func TestAppleVerifier_RejectsMissingSub(t *testing.T) {
	key := newRSAKey(t)
	now := fixedNow()
	v := auth.NewAppleVerifierForTest(testAudience, keyFuncFor(key), now)

	tok := signRS256(t, key, jwt.MapClaims{
		"iss": "https://appleid.apple.com",
		"aud": testAudience,
		// no sub
		"iat": now().Unix(),
		"exp": now().Add(10 * time.Minute).Unix(),
	})

	_, err := v.Verify(context.Background(), tok)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("want ErrInvalidToken, got %v", err)
	}
}
