// Package auth handles Sign in with Apple JWT verification and the server's
// own session JWT issuance/verification.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

const (
	appleJWKSURL = "https://appleid.apple.com/auth/keys"
	appleIssuer  = "https://appleid.apple.com"
)

// AppleIdentity is what we extract from a verified Apple ID token.
type AppleIdentity struct {
	Sub   string
	Email string
}

// AppleVerifier verifies Apple-issued identity tokens.
type AppleVerifier interface {
	Verify(ctx context.Context, idToken string) (AppleIdentity, error)
}

// ErrInvalidToken is returned (wrapped) for any token that fails signature,
// claim, or format checks.
var ErrInvalidToken = errors.New("auth: invalid apple id token")

type jwtAppleVerifier struct {
	audience string
	keyFunc  jwt.Keyfunc
	now      func() time.Time
}

// NewAppleVerifier builds a production verifier that fetches Apple's JWKs with
// caching from appleid.apple.com. audience is the client_id Apple issued the
// token for (your iOS bundle id for the native Sign in with Apple flow).
func NewAppleVerifier(ctx context.Context, audience string) (AppleVerifier, error) {
	k, err := keyfunc.NewDefaultCtx(ctx, []string{appleJWKSURL})
	if err != nil {
		return nil, fmt.Errorf("init apple jwks: %w", err)
	}
	return &jwtAppleVerifier{
		audience: audience,
		keyFunc:  k.Keyfunc,
		now:      time.Now,
	}, nil
}

// NewAppleVerifierForTest wires a verifier against a custom key resolver and
// clock. Used by tests to sign fixtures with a known key instead of hitting
// Apple's live JWKs endpoint.
func NewAppleVerifierForTest(audience string, keyFunc jwt.Keyfunc, now func() time.Time) AppleVerifier {
	if now == nil {
		now = time.Now
	}
	return &jwtAppleVerifier{audience: audience, keyFunc: keyFunc, now: now}
}

func (v *jwtAppleVerifier) Verify(_ context.Context, idToken string) (AppleIdentity, error) {
	tok, err := jwt.Parse(idToken, v.keyFunc,
		jwt.WithIssuer(appleIssuer),
		jwt.WithAudience(v.audience),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithLeeway(60*time.Second),
		jwt.WithTimeFunc(v.now),
	)
	if err != nil {
		return AppleIdentity{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return AppleIdentity{}, fmt.Errorf("%w: claims not a map", ErrInvalidToken)
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return AppleIdentity{}, fmt.Errorf("%w: missing sub claim", ErrInvalidToken)
	}
	email, _ := claims["email"].(string)
	return AppleIdentity{Sub: sub, Email: email}, nil
}
