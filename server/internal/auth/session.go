package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidSession is returned (wrapped) for any session token that fails
// signature, claim, or format checks.
var ErrInvalidSession = errors.New("auth: invalid session token")

// SessionTokens issues and verifies the server's own HS256-signed session JWTs.
// After we verify an Apple identity token in POST /auth/apple, we issue one of
// these; clients send it back as `Authorization: Bearer <token>`.
type SessionTokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewSessionTokens requires at least 32 bytes of secret. ttl defaults to 30d
// if zero.
func NewSessionTokens(secret []byte, ttl time.Duration) (*SessionTokens, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("auth: session secret must be at least 32 bytes, got %d", len(secret))
	}
	if ttl == 0 {
		ttl = 30 * 24 * time.Hour
	}
	return &SessionTokens{secret: secret, ttl: ttl, now: time.Now}, nil
}

// WithClock overrides the clock, for tests.
func (s *SessionTokens) WithClock(now func() time.Time) *SessionTokens {
	cp := *s
	cp.now = now
	return &cp
}

// Issue returns a signed JWT whose `sub` claim is userID.
func (s *SessionTokens) Issue(userID string) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"iat": s.now().Unix(),
		"exp": s.now().Add(s.ttl).Unix(),
	})
	return tok.SignedString(s.secret)
}

// Verify parses token, checks signature / expiration / method, and returns the
// userID (`sub` claim).
func (s *SessionTokens) Verify(token string) (string, error) {
	tok, err := jwt.Parse(token, func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(60*time.Second),
		jwt.WithTimeFunc(s.now),
	)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidSession, err)
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("%w: claims not a map", ErrInvalidSession)
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", fmt.Errorf("%w: missing sub", ErrInvalidSession)
	}
	return sub, nil
}
