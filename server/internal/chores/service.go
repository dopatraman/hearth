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
	ErrWillCompleteByRequiresClaim = errors.New("chores: will_complete_by requires claim")
	ErrReorderEmpty     = errors.New("chores: reorder requires at least one id")
	ErrReorderDuplicate = errors.New("chores: reorder ids must be unique")
	ErrReorderUnknown   = errors.New("chores: reorder references an unknown chore")
	ErrReorderNotOpen   = errors.New("chores: reorder references a completed chore")
	ErrReorderNotClaimed = errors.New("chores: reorder references an unclaimed chore")
	ErrReorderMixedLists = errors.New("chores: reorder ids span multiple lists")
	ErrReorderNotMember  = errors.New("chores: reorder user is not a member of the list")
)

// Create inserts a new chore + a 'created' event, and optionally claims it
// for the creator and sets a will_complete_by date, all in the caller's
// transaction. Pass tx-bound Queries (UoW.InTx) so every step either lands
// together or not at all — a household member reading between steps should
// never see a half-formed row.
//
// `willCompleteBy` is only honored when `claim` is true; passing a date
// without claiming returns ErrWillCompleteByRequiresClaim (nothing is
// written).
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

// Backlog returns open chores in the given list, newest first.
func (s *Service) Backlog(ctx context.Context, listID string) ([]db.Chore, error) {
	return s.q.BacklogForList(ctx, listID)
}

// BacklogForUser returns the union of open chores across every list the user
// is a member of, newest first. Snoozed chores are included and carry a
// non-nil SnoozedUntil for the caller — the UI uses that to render them in a
// dedicated Snoozed section instead of hiding them outright.
func (s *Service) BacklogForUser(ctx context.Context, userID string) ([]db.BacklogForUserRow, error) {
	return s.q.BacklogForUser(ctx, userID)
}

// Wake removes the caller's snooze on a chore so it returns to its normal
// backlog section on the next read. Idempotent: a DELETE on a non-existent
// snooze row does not error. No event is emitted — the events.type CHECK
// does not include an "unsnoozed" value and extending it needs a SQLite
// table rewrite we're deferring; the original "snoozed" event remains the
// audit record of the user's intent at that moment.
func (s *Service) Wake(ctx context.Context, choreID, userID string) error {
	return s.q.DeleteSnooze(ctx, db.DeleteSnoozeParams{
		UserID:  userID,
		ChoreID: choreID,
	})
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

// Snooze inserts (or refreshes) a per-user snooze for 8 hours. Writes a
// 'snoozed' event. Snoozes are per-(user, chore) — Alice snoozing doesn't
// affect Bob. Does not check claim/membership — anyone who can see a chore
// can snooze it from their own view.
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

// Reorder rewrites claimed_order for the chores in `order` as 1..N (in the
// submitted order), then appends any still-claimed chore in the same list
// that was omitted from the body at N+1, N+2... preserving its existing
// relative order. All ids must be currently claimed, in state 'open', and
// belong to one list that userID is a member of.
//
// Returns the list_id affected so the caller can dispatch the silent push.
func (s *Service) Reorder(ctx context.Context, userID string, order []string) (string, error) {
	seen, err := validateReorderShape(order)
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

	if err := s.rewriteClaimedOrder(ctx, listID, order, seen); err != nil {
		return "", err
	}

	return listID, nil
}

// validateReorderShape checks the request body's shape alone — no DB.
// Returns the set of submitted ids so rewriteClaimedOrder can skip them
// when it walks the remaining claimed chores.
func validateReorderShape(order []string) (map[string]struct{}, error) {
	if len(order) == 0 {
		return nil, ErrReorderEmpty
	}
	seen := make(map[string]struct{}, len(order))
	for _, id := range order {
		if _, ok := seen[id]; ok {
			return nil, ErrReorderDuplicate
		}
		seen[id] = struct{}{}
	}
	return seen, nil
}

// validateReorderTargets fetches each submitted chore, enforces the
// open+claimed invariant, and derives the single list_id the request
// applies to. The list_id is only returned after uniqueness is proven —
// the caller never sees a value that might not be the final answer.
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

// requireListMembership returns ErrReorderNotMember iff userID is not
// a member of listID. One DB read + a scan over the user's memberships.
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

// rewriteClaimedOrder renumbers the entire claimed section for listID so
// that submitted ids occupy ranks 1..N and everything else claimed in the
// list fills N+1..K, preserving its existing relative order. The two
// phases run together because together they maintain the invariant "every
// open+claimed chore in the list has a distinct rank in 1..K"; running
// only the first phase could leave non-submitted rows colliding with the
// newly-written 1..N.
func (s *Service) rewriteClaimedOrder(
	ctx context.Context,
	listID string,
	submitted []string,
	seen map[string]struct{},
) error {
	for i, id := range submitted {
		rank := int64(i + 1)
		if err := s.q.SetClaimedOrder(ctx, db.SetClaimedOrderParams{
			ID:           id,
			ClaimedOrder: &rank,
		}); err != nil {
			return err
		}
	}
	claimed, err := s.q.ClaimedChoresForList(ctx, listID)
	if err != nil {
		return err
	}
	next := int64(len(submitted) + 1)
	for _, c := range claimed {
		if _, ok := seen[c.ID]; ok {
			continue
		}
		rank := next
		if err := s.q.SetClaimedOrder(ctx, db.SetClaimedOrderParams{
			ID:           c.ID,
			ClaimedOrder: &rank,
		}); err != nil {
			return err
		}
		next++
	}
	return nil
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
