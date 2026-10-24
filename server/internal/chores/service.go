// Package chores holds business logic for household chores: create, backlog
// read, and (later) claim/complete/snooze.
package chores

import (
	"context"
	"errors"

	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/google/uuid"
)

type Service struct {
	q *db.Queries
}

func NewService(q *db.Queries) *Service {
	return &Service{q: q}
}

var ErrEmptyTitle = errors.New("chores: title must not be empty")

// Create inserts a new chore + a corresponding 'created' event. Pass tx-bound
// Queries (UoW.InTx) so the two writes are atomic — a client that sees a
// chore exists should also see its creation event.
func (s *Service) Create(ctx context.Context, listID, creatorID, title string, notes *string) (db.Chore, error) {
	if title == "" {
		return db.Chore{}, ErrEmptyTitle
	}

	choreID, err := uuid.NewV7()
	if err != nil {
		return db.Chore{}, err
	}
	chore, err := s.q.CreateChore(ctx, db.CreateChoreParams{
		ID:        choreID.String(),
		ListID:    listID,
		CreatorID: creatorID,
		Title:     title,
		Notes:     notes,
	})
	if err != nil {
		return db.Chore{}, err
	}

	eventID, err := uuid.NewV7()
	if err != nil {
		return db.Chore{}, err
	}
	if err := s.q.InsertEvent(ctx, db.InsertEventParams{
		ID:      eventID.String(),
		ListID:  listID,
		ChoreID: chore.ID,
		ActorID: creatorID,
		Type:    "created",
	}); err != nil {
		return db.Chore{}, err
	}

	return chore, nil
}

// Backlog returns open chores in the given list, newest first.
func (s *Service) Backlog(ctx context.Context, listID string) ([]db.Chore, error) {
	return s.q.BacklogForList(ctx, listID)
}

// BacklogForUser returns the union of open chores across every list the user
// is a member of, newest first. Preferred over Backlog(listID) when the user
// may belong to multiple lists (invite flow).
func (s *Service) BacklogForUser(ctx context.Context, userID string) ([]db.Chore, error) {
	return s.q.BacklogForUser(ctx, userID)
}
