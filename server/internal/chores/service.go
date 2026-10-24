// Package chores holds business logic for household chores: create, backlog
// read, and (later) claim/complete/snooze.
package chores

import (
	"context"
	"database/sql"
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

var (
	ErrEmptyTitle   = errors.New("chores: title must not be empty")
	ErrNotClaimable = errors.New("chores: cannot claim (not found, already claimed, or completed)")
	ErrNotCompletable = errors.New("chores: cannot complete (not found, not claimed by you, or already completed)")
	ErrNotModifiable  = errors.New("chores: cannot modify (not found, not claimed by you, or already completed)")
	ErrNotReleasable  = errors.New("chores: cannot release (not found, not claimed by you, or already completed)")
)

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

// Claim atomically sets claimed_by to userID iff the chore is open and
// unclaimed. On success, writes a 'claimed' event. ErrNotClaimable if the
// chore doesn't exist, is already claimed, or is completed — the caller can
// surface a generic "can't claim right now" to the user.
func (s *Service) Claim(ctx context.Context, choreID, userID string) (db.Chore, error) {
	user := userID
	chore, err := s.q.ClaimChore(ctx, db.ClaimChoreParams{
		ClaimedBy: &user,
		ID:        choreID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Chore{}, ErrNotClaimable
		}
		return db.Chore{}, err
	}

	eventID, err := uuid.NewV7()
	if err != nil {
		return db.Chore{}, err
	}
	if err := s.q.InsertEvent(ctx, db.InsertEventParams{
		ID:      eventID.String(),
		ListID:  chore.ListID,
		ChoreID: chore.ID,
		ActorID: userID,
		Type:    "claimed",
	}); err != nil {
		return db.Chore{}, err
	}
	return chore, nil
}

// Complete atomically marks the chore completed iff it's open AND claimed by
// userID. On success, writes a 'completed' event. ErrNotCompletable otherwise.
func (s *Service) Complete(ctx context.Context, choreID, userID string) (db.Chore, error) {
	user := userID
	chore, err := s.q.CompleteChore(ctx, db.CompleteChoreParams{
		CompletedBy: &user,
		ID:          choreID,
		ClaimedBy:   &user,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Chore{}, ErrNotCompletable
		}
		return db.Chore{}, err
	}

	eventID, err := uuid.NewV7()
	if err != nil {
		return db.Chore{}, err
	}
	if err := s.q.InsertEvent(ctx, db.InsertEventParams{
		ID:      eventID.String(),
		ListID:  chore.ListID,
		ChoreID: chore.ID,
		ActorID: userID,
		Type:    "completed",
	}); err != nil {
		return db.Chore{}, err
	}
	return chore, nil
}

// SetWillCompleteBy sets (or clears, if when == nil) the claimant's committed
// completion date. Only the current claimant can set it; unclaimed or
// someone-else's-claim returns ErrNotModifiable. Writes a
// 'will_complete_by_set' event on success.
func (s *Service) SetWillCompleteBy(ctx context.Context, choreID, userID string, when *string) (db.Chore, error) {
	user := userID
	chore, err := s.q.SetWillCompleteBy(ctx, db.SetWillCompleteByParams{
		WillCompleteBy: when,
		ID:             choreID,
		ClaimedBy:      &user,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Chore{}, ErrNotModifiable
		}
		return db.Chore{}, err
	}

	eventID, err := uuid.NewV7()
	if err != nil {
		return db.Chore{}, err
	}
	if err := s.q.InsertEvent(ctx, db.InsertEventParams{
		ID:      eventID.String(),
		ListID:  chore.ListID,
		ChoreID: chore.ID,
		ActorID: userID,
		Type:    "will_complete_by_set",
	}); err != nil {
		return db.Chore{}, err
	}
	return chore, nil
}

// Release atomically clears claimed_by, claimed_at, and will_complete_by iff
// the chore is open AND claimed by userID. On success, writes a 'released'
// event. ErrNotReleasable otherwise.
func (s *Service) Release(ctx context.Context, choreID, userID string) (db.Chore, error) {
	user := userID
	chore, err := s.q.ReleaseClaim(ctx, db.ReleaseClaimParams{
		ID:        choreID,
		ClaimedBy: &user,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Chore{}, ErrNotReleasable
		}
		return db.Chore{}, err
	}

	eventID, err := uuid.NewV7()
	if err != nil {
		return db.Chore{}, err
	}
	if err := s.q.InsertEvent(ctx, db.InsertEventParams{
		ID:      eventID.String(),
		ListID:  chore.ListID,
		ChoreID: chore.ID,
		ActorID: userID,
		Type:    "unclaimed",
	}); err != nil {
		return db.Chore{}, err
	}
	return chore, nil
}

// Graveyard returns completed chores across the user's lists, newest-completed
// first.
func (s *Service) Graveyard(ctx context.Context, userID string) ([]db.Chore, error) {
	return s.q.GraveyardForUser(ctx, userID)
}

// SearchGraveyard returns completed chores whose title contains q (case-
// insensitive), newest-completed first. Empty q returns everything.
func (s *Service) SearchGraveyard(ctx context.Context, userID, q string) ([]db.Chore, error) {
	if q == "" {
		return s.Graveyard(ctx, userID)
	}
	return s.q.SearchGraveyard(ctx, db.SearchGraveyardParams{
		UserID: userID,
		Query:  q,
	})
}
