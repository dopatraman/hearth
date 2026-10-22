// Package items holds business logic for chore items: create, backlog read,
// and (later) claim/complete/snooze.
package items

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

var ErrEmptyTitle = errors.New("items: title must not be empty")

// Create inserts a new item + a corresponding 'created' event. Call with a
// tx-bound Queries (UoW.InTx) if atomicity across both writes matters —
// which it should: a client that sees an item exists should also see its
// creation event.
func (s *Service) Create(ctx context.Context, listID, creatorID, title string, notes *string) (db.Item, error) {
	if title == "" {
		return db.Item{}, ErrEmptyTitle
	}

	itemID, err := uuid.NewV7()
	if err != nil {
		return db.Item{}, err
	}
	item, err := s.q.CreateItem(ctx, db.CreateItemParams{
		ID:        itemID.String(),
		ListID:    listID,
		CreatorID: creatorID,
		Title:     title,
		Notes:     notes,
	})
	if err != nil {
		return db.Item{}, err
	}

	eventID, err := uuid.NewV7()
	if err != nil {
		return db.Item{}, err
	}
	if err := s.q.InsertEvent(ctx, db.InsertEventParams{
		ID:           eventID.String(),
		ListID:       listID,
		ItemID:       item.ID,
		ActorID:      creatorID,
		TargetUserID: nil,
		Type:         "created",
		Payload:      nil,
	}); err != nil {
		return db.Item{}, err
	}

	return item, nil
}

// Backlog returns open items in the given list, newest first.
func (s *Service) Backlog(ctx context.Context, listID string) ([]db.Item, error) {
	return s.q.BacklogForList(ctx, listID)
}
