// Package lists holds business logic for lists and list_memberships.
package lists

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

// EnsurePersonalMembership returns a list_id the user is a member of. If the
// user has no memberships yet, it creates a fresh "Home" list and adds them.
// Idempotent: subsequent calls return a membership's list_id.
func (s *Service) EnsurePersonalMembership(ctx context.Context, userID string) (string, error) {
	memberships, err := s.q.ListMembershipsForUser(ctx, userID)
	if err != nil {
		return "", err
	}
	if len(memberships) > 0 {
		return memberships[0].ListID, nil
	}

	listID, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	list, err := s.q.CreateList(ctx, db.CreateListParams{
		ID:   listID.String(),
		Name: "Home",
	})
	if err != nil {
		return "", err
	}
	if err := s.q.AddMembership(ctx, db.AddMembershipParams{
		ListID: list.ID,
		UserID: userID,
	}); err != nil {
		return "", err
	}
	return list.ID, nil
}
