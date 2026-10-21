// Package users holds business logic for user accounts and list memberships.
// Thin wrapper around sqlc-generated queries today; grows as auth, invites,
// and notification preferences land.
package users

import (
	"context"

	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/google/uuid"
)

type Service struct {
	q *db.Queries
}

func NewService(q *db.Queries) *Service {
	return &Service{q: q}
}

// UpsertByAppleSub returns the user with the given apple_sub, creating one if
// none exists. displayName is applied on first-sign-in only — once a user has
// a display_name, subsequent nil values do not clear it.
func (s *Service) UpsertByAppleSub(ctx context.Context, appleSub string, displayName *string) (db.User, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return db.User{}, err
	}
	return s.q.UpsertUserByAppleSub(ctx, db.UpsertUserByAppleSubParams{
		ID:          id.String(),
		AppleSub:    appleSub,
		DisplayName: displayName,
	})
}

// ByID returns the user with the given id, or sql.ErrNoRows.
func (s *Service) ByID(ctx context.Context, id string) (db.User, error) {
	return s.q.GetUserByID(ctx, id)
}
