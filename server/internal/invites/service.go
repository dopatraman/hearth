// Package invites holds business logic for list invitations: generate a short
// code someone can share, redeem a code to join a list.
package invites

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/dopatraman/hearth/server/internal/db"
)

const (
	codeLength = 6
	// How long an invite is valid for.
	TTL = 24 * time.Hour
	// How many times to retry on PK collision when generating a code.
	// 6 digits = 1M space, so collision probability is tiny; 10 retries is
	// paranoia.
	maxCollisionRetries = 10
)

var (
	// ErrNotFound  — no invite with that code (never issued or already pruned).
	ErrNotFound = errors.New("invites: not found")
	// ErrExpired   — invite exists but is past its expires_at.
	ErrExpired = errors.New("invites: expired")
	// ErrGeneration — exhausted retries trying to generate a unique code.
	ErrGeneration = errors.New("invites: failed to generate unique code")
)

type Service struct {
	q   *db.Queries
	now func() time.Time
}

func NewService(q *db.Queries) *Service {
	return &Service{q: q, now: time.Now}
}

// WithClock overrides the clock, for tests.
func (s *Service) WithClock(now func() time.Time) *Service {
	cp := *s
	cp.now = now
	return &cp
}

// Create generates a 6-digit code, inserts an invite tied to listID + createdBy,
// and returns the stored invite (including expires_at). Retries on code
// collision up to maxCollisionRetries.
func (s *Service) Create(ctx context.Context, listID, createdBy string) (db.Invite, error) {
	expiresAt := s.now().UTC().Add(TTL).Format("2006-01-02T15:04:05.000Z")

	for attempt := 0; attempt < maxCollisionRetries; attempt++ {
		code, err := generateCode()
		if err != nil {
			return db.Invite{}, err
		}
		invite, err := s.q.CreateInvite(ctx, db.CreateInviteParams{
			Code:      code,
			ListID:    listID,
			CreatedBy: createdBy,
			ExpiresAt: expiresAt,
		})
		if err == nil {
			return invite, nil
		}
		// Collision: retry with a fresh code. Any other error bubbles up.
		if !isPKCollision(err) {
			return db.Invite{}, err
		}
	}
	return db.Invite{}, ErrGeneration
}

// Lookup fetches an invite by code and validates it's not expired. Returns
// ErrNotFound or ErrExpired for the two sad paths; the invite otherwise.
func (s *Service) Lookup(ctx context.Context, code string) (db.Invite, error) {
	inv, err := s.q.GetInviteByCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Invite{}, ErrNotFound
		}
		return db.Invite{}, err
	}
	expires, err := time.Parse("2006-01-02T15:04:05.000Z", inv.ExpiresAt)
	if err != nil {
		// Older sqlite outputs — try RFC3339.
		expires, err = time.Parse(time.RFC3339Nano, inv.ExpiresAt)
		if err != nil {
			return db.Invite{}, fmt.Errorf("invites: parse expires_at %q: %w", inv.ExpiresAt, err)
		}
	}
	if !expires.After(s.now()) {
		return db.Invite{}, ErrExpired
	}
	return inv, nil
}

// generateCode returns a cryptographically random 6-digit, zero-padded string.
func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// isPKCollision returns true if err looks like a UNIQUE/PK conflict on invites.
// modernc.org/sqlite surfaces constraint violations in the error message.
func isPKCollision(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "PRIMARY KEY must be unique") ||
		strings.Contains(msg, "constraint failed: invites")
}
