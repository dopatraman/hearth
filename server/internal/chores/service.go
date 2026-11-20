// Package chores is the business logic for household chores.
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
	ErrWillCompleteByRequiresClaim = errors.New("chores: will_complete_by requires claim")
	ErrReorderEmpty     = errors.New("chores: reorder requires at least one id")
	ErrReorderDuplicate = errors.New("chores: reorder ids must be unique")
	ErrReorderUnknown   = errors.New("chores: reorder references an unknown chore")
	ErrReorderNotOpen   = errors.New("chores: reorder references a completed chore")
	ErrReorderNotClaimed = errors.New("chores: reorder references an unclaimed chore")
	ErrReorderMixedLists = errors.New("chores: reorder ids span multiple lists")
	ErrReorderNotMember  = errors.New("chores: reorder user is not a member of the list")
)

// Create inserts a chore and, if claim is true, claims it for the creator and
// optionally stamps a will_complete_by date — all in one transaction. A
// willCompleteBy without claim returns ErrWillCompleteByRequiresClaim.
func (s *Service) Create(
	ctx context.Context,
	listID, creatorID, title string,
	notes *string,
	claim bool,
	willCompleteBy *string,
) (db.Chore, error) {
	if title == "" {
		return db.Chore{}, ErrEmptyTitle
	}
	if willCompleteBy != nil && !claim {
		return db.Chore{}, ErrWillCompleteByRequiresClaim
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

	if !claim {
		return chore, nil
	}

	actor := creatorID
	claimed, err := s.q.ClaimChore(ctx, db.ClaimChoreParams{
		ClaimedBy: &actor,
		ID:        chore.ID,
	})
	if err != nil {
		return db.Chore{}, err
	}
	claimed, err = s.q.AssignNextClaimedOrder(ctx, chore.ID)
	if err != nil {
		return db.Chore{}, err
	}

	claimEventID, err := uuid.NewV7()
	if err != nil {
		return db.Chore{}, err
	}
	if err := s.q.InsertEvent(ctx, db.InsertEventParams{
		ID:      claimEventID.String(),
		ListID:  listID,
		ChoreID: chore.ID,
		ActorID: creatorID,
		Type:    "claimed",
	}); err != nil {
		return db.Chore{}, err
	}

	if willCompleteBy == nil {
		return claimed, nil
	}

	scheduled, err := s.q.SetWillCompleteBy(ctx, db.SetWillCompleteByParams{
		WillCompleteBy: willCompleteBy,
		ID:             chore.ID,
		ClaimedBy:      &actor,
	})
	if err != nil {
		return db.Chore{}, err
	}

	scheduleEventID, err := uuid.NewV7()
	if err != nil {
		return db.Chore{}, err
	}
	if err := s.q.InsertEvent(ctx, db.InsertEventParams{
		ID:      scheduleEventID.String(),
		ListID:  listID,
		ChoreID: chore.ID,
		ActorID: creatorID,
		Type:    "will_complete_by_set",
	}); err != nil {
		return db.Chore{}, err
	}

	return scheduled, nil
}

// Backlog returns the list's open chores, newest first.
func (s *Service) Backlog(ctx context.Context, listID string) ([]db.Chore, error) {
	return s.q.BacklogForList(ctx, listID)
}

// BacklogForUser returns every open chore across the user's lists, newest
// first. Snoozed chores are included with snoozed_until set so the client can
// render them in a Snoozed section.
func (s *Service) BacklogForUser(ctx context.Context, userID string) ([]db.BacklogForUserRow, error) {
	return s.q.BacklogForUser(ctx, userID)
}

// Wake removes the user's snooze on the chore. Idempotent. No event is
// written because the events.type CHECK doesn't have an "unsnoozed" value;
// the original "snoozed" event remains the audit trail.
func (s *Service) Wake(ctx context.Context, choreID, userID string) error {
	return s.q.DeleteSnooze(ctx, db.DeleteSnoozeParams{
		UserID:  userID,
		ChoreID: choreID,
	})
}

// Claim claims the chore for userID and stamps it with the next claimed_order
// in its list. Returns ErrNotClaimable if the chore is missing, already
// claimed, or completed.
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
	chore, err = s.q.AssignNextClaimedOrder(ctx, choreID)
	if err != nil {
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

// Complete marks the chore completed and clears its claimed_order. Returns
// ErrNotCompletable if the chore isn't claimed by userID or is already
// completed.
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
	if err := s.q.ClearClaimedOrder(ctx, choreID); err != nil {
		return db.Chore{}, err
	}
	chore.ClaimedOrder = nil

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

// SetWillCompleteBy sets or clears (when==nil) the claimant's committed
// completion date. Returns ErrNotModifiable unless the chore is claimed by
// userID and still open.
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

// Snooze snoozes the chore for userID for 8 hours. Per-user, so another
// household member's view is unaffected. No claim or membership check —
// anyone who can see the chore can snooze it from their own view.
func (s *Service) Snooze(ctx context.Context, choreID, userID string) error {
	chore, err := s.q.GetChoreByID(ctx, choreID)
	if err != nil {
		return err
	}
	if err := s.q.UpsertSnooze(ctx, db.UpsertSnoozeParams{
		UserID:  userID,
		ChoreID: choreID,
	}); err != nil {
		return err
	}
	eventID, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return s.q.InsertEvent(ctx, db.InsertEventParams{
		ID:      eventID.String(),
		ListID:  chore.ListID,
		ChoreID: choreID,
		ActorID: userID,
		Type:    "snoozed",
	})
}

// Release clears the user's claim on the chore (claimed_by, claimed_at,
// will_complete_by, claimed_order). Returns ErrNotReleasable if the chore
// isn't claimed by userID or is already completed.
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
	if err := s.q.ClearClaimedOrder(ctx, choreID); err != nil {
		return db.Chore{}, err
	}
	chore.ClaimedOrder = nil

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

// Reorder renumbers the Claimed section of userID's list to match `order`.
// All submitted ids must be currently claimed in one list userID belongs to.
// Any other still-claimed chore in that list gets appended at the end,
// preserving its existing relative order (last-write-wins, lenient).
// Returns the list_id so the caller can send the silent push.
func (s *Service) Reorder(ctx context.Context, userID string, order []string) (string, error) {
	orderSet, err := validateReorderShape(order)
	if err != nil {
		return "", err
	}

	listID, err := s.validateReorderTargets(ctx, order)
	if err != nil {
		return "", err
	}

	if err := s.requireListMembership(ctx, userID, listID); err != nil {
		return "", err
	}

	if err := s.rewriteClaimedOrder(ctx, listID, order, orderSet); err != nil {
		return "", err
	}

	return listID, nil
}

// validateReorderShape rejects an empty body and duplicate ids. Returns the
// same ids in set form so later steps can check membership in O(1).
func validateReorderShape(order []string) (map[string]struct{}, error) {
	if len(order) == 0 {
		return nil, ErrReorderEmpty
	}
	orderSet := make(map[string]struct{}, len(order))
	for _, id := range order {
		if _, ok := orderSet[id]; ok {
			return nil, ErrReorderDuplicate
		}
		orderSet[id] = struct{}{}
	}
	return orderSet, nil
}

// validateReorderTargets checks every submitted chore is open and claimed and
// returns the single list_id they all belong to. The list_id is only
// returned once uniqueness is proven.
func (s *Service) validateReorderTargets(ctx context.Context, order []string) (string, error) {
	listIDs := map[string]struct{}{}
	for _, id := range order {
		c, err := s.q.GetChoreByID(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", ErrReorderUnknown
			}
			return "", err
		}
		if c.State != "open" {
			return "", ErrReorderNotOpen
		}
		if c.ClaimedAt == nil {
			return "", ErrReorderNotClaimed
		}
		listIDs[c.ListID] = struct{}{}
		if len(listIDs) > 1 {
			return "", ErrReorderMixedLists
		}
	}
	for id := range listIDs {
		return id, nil
	}
	return "", ErrReorderEmpty
}

// requireListMembership returns ErrReorderNotMember if userID isn't in
// list_memberships for listID.
func (s *Service) requireListMembership(ctx context.Context, userID, listID string) error {
	memberships, err := s.q.ListMembershipsForUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, m := range memberships {
		if m.ListID == listID {
			return nil
		}
	}
	return ErrReorderNotMember
}

// rewriteClaimedOrder renumbers the list's Claimed section. The ids in
// `order` get ranks 1..N; every other still-claimed chore in the list gets
// N+1..K, keeping its existing relative order. Both halves run together so
// every claimed chore ends up with a distinct rank — renumbering only
// `order` could leave old ranks colliding with the fresh 1..N.
func (s *Service) rewriteClaimedOrder(
	ctx context.Context,
	listID string,
	order []string,
	orderSet map[string]struct{},
) error {
	claimed, err := s.q.ClaimedChoresForList(ctx, listID)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(claimed))
	ids = append(ids, order...)
	for _, c := range claimed {
		if _, ok := orderSet[c.ID]; !ok {
			ids = append(ids, c.ID)
		}
	}
	return s.assignRanks(ctx, ids, 1)
}

// assignRanks writes consecutive claimed_order values to ids, starting at
// startRank. ids[0] gets startRank, ids[1] gets startRank+1, and so on.
func (s *Service) assignRanks(ctx context.Context, ids []string, startRank int64) error {
	rank := startRank
	for _, id := range ids {
		r := rank
		if err := s.q.SetClaimedOrder(ctx, db.SetClaimedOrderParams{
			ID:           id,
			ClaimedOrder: &r,
		}); err != nil {
			return err
		}
		rank++
	}
	return nil
}

// Graveyard returns completed chores across the user's lists, newest first.
func (s *Service) Graveyard(ctx context.Context, userID string) ([]db.Chore, error) {
	return s.q.GraveyardForUser(ctx, userID)
}

// SearchGraveyard returns completed chores whose title contains q
// (case-insensitive), newest first. Empty q returns everything.
func (s *Service) SearchGraveyard(ctx context.Context, userID, q string) ([]db.Chore, error) {
	if q == "" {
		return s.Graveyard(ctx, userID)
	}
	return s.q.SearchGraveyard(ctx, db.SearchGraveyardParams{
		UserID: userID,
		Query:  q,
	})
}
