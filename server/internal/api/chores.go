package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/dopatraman/hearth/server/internal/chores"
	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/lists"
)

type choreDTO struct {
	ID             string  `json:"id"`
	ListID         string  `json:"list_id"`
	CreatorID      string  `json:"creator_id"`
	Title          string  `json:"title"`
	Notes          *string `json:"notes"`
	State          string  `json:"state"`
	ClaimedBy      *string `json:"claimed_by"`
	ClaimedAt      *string `json:"claimed_at"`
	WillCompleteBy *string `json:"will_complete_by"`
	CompletedBy    *string `json:"completed_by"`
	CompletedAt    *string `json:"completed_at"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	Overdue        bool    `json:"overdue"`
}

func toChoreDTO(c db.Chore) choreDTO {
	return choreDTO{
		ID:             c.ID,
		ListID:         c.ListID,
		CreatorID:      c.CreatorID,
		Title:          c.Title,
		Notes:          c.Notes,
		State:          c.State,
		ClaimedBy:      c.ClaimedBy,
		ClaimedAt:      c.ClaimedAt,
		WillCompleteBy: c.WillCompleteBy,
		CompletedBy:    c.CompletedBy,
		CompletedAt:    c.CompletedAt,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		Overdue:        isOverdue(c),
	}
}

// isOverdue — pure derivation from (state, will_complete_by, now). Chore is
// overdue iff it's still open AND has a will_complete_by in the past.
func isOverdue(c db.Chore) bool {
	if c.State != "open" || c.WillCompleteBy == nil {
		return false
	}
	t, err := time.Parse(time.RFC3339Nano, *c.WillCompleteBy)
	if err != nil {
		// Fall back to the SQLite strftime format we write.
		t, err = time.Parse("2006-01-02T15:04:05.000Z", *c.WillCompleteBy)
		if err != nil {
			return false
		}
	}
	return t.Before(time.Now().UTC())
}

type createChoreRequest struct {
	Title          string  `json:"title"`
	Notes          *string `json:"notes,omitempty"`
	Claim          bool    `json:"claim,omitempty"`
	WillCompleteBy *string `json:"will_complete_by,omitempty"`
}

type backlogResponse struct {
	Chores []choreDTO `json:"chores"`
}

// POST /chores — creates a new chore on the authenticated user's list
// (lazily provisioning the list if it doesn't yet exist). Chore + 'created'
// event are written in a single transaction.
func (s *Server) handleCreateChore(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}

	var req createChoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var created db.Chore
	err := s.UoW.InTx(r.Context(), func(q *db.Queries) error {
		listID, err := lists.NewService(q).EnsurePersonalMembership(r.Context(), u.ID)
		if err != nil {
			return err
		}
		created, err = chores.NewService(q).Create(r.Context(), listID, u.ID, req.Title, req.Notes, req.Claim, req.WillCompleteBy)
		return err
	})
	if err != nil {
		if errors.Is(err, chores.ErrEmptyTitle) {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
		if errors.Is(err, chores.ErrWillCompleteByRequiresClaim) {
			writeError(w, http.StatusBadRequest, "will_complete_by requires claim")
			return
		}
		slog.ErrorContext(r.Context(), "create chore", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	s.Dispatcher.Dispatch(r.Context(), created.ListID, u.ID, "New chore: "+created.Title)
	writeJSON(w, http.StatusCreated, toChoreDTO(created))
}

// GET /chores — returns the backlog (open chores) across every list the
// authenticated user is a member of. Does not auto-provision a list — a brand
// new user with no memberships gets an empty array, which is the correct
// pre-invite / pre-first-add state.
func (s *Server) handleListChores(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}

	backlog, err := chores.NewService(s.UoW.Queries()).BacklogForUser(r.Context(), u.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list chores", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	out := backlogResponse{Chores: make([]choreDTO, 0, len(backlog))}
	for _, c := range backlog {
		out.Chores = append(out.Chores, toChoreDTO(c))
	}
	writeJSON(w, http.StatusOK, out)
}

// PATCH /chores/{id}/claim — atomic claim-if-unclaimed.
// 409 if the chore doesn't exist, is already claimed, or is completed.
func (s *Server) handleClaimChore(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}
	choreID := chi.URLParam(r, "id")
	if choreID == "" {
		writeError(w, http.StatusBadRequest, "chore id is required")
		return
	}

	var claimed db.Chore
	err := s.UoW.InTx(r.Context(), func(q *db.Queries) error {
		var err error
		claimed, err = chores.NewService(q).Claim(r.Context(), choreID, u.ID)
		return err
	})
	if err != nil {
		if errors.Is(err, chores.ErrNotClaimable) {
			writeError(w, http.StatusConflict, "chore cannot be claimed")
			return
		}
		slog.ErrorContext(r.Context(), "claim chore", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.Dispatcher.Dispatch(r.Context(), claimed.ListID, u.ID, "Claimed: "+claimed.Title)
	writeJSON(w, http.StatusOK, toChoreDTO(claimed))
}

// POST /chores/{id}/complete — mark completed. Only the current claimant can
// complete; 409 otherwise.
func (s *Server) handleCompleteChore(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}
	choreID := chi.URLParam(r, "id")
	if choreID == "" {
		writeError(w, http.StatusBadRequest, "chore id is required")
		return
	}

	var completed db.Chore
	err := s.UoW.InTx(r.Context(), func(q *db.Queries) error {
		var err error
		completed, err = chores.NewService(q).Complete(r.Context(), choreID, u.ID)
		return err
	})
	if err != nil {
		if errors.Is(err, chores.ErrNotCompletable) {
			writeError(w, http.StatusConflict, "chore cannot be completed")
			return
		}
		slog.ErrorContext(r.Context(), "complete chore", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.Dispatcher.Dispatch(r.Context(), completed.ListID, u.ID, "Done: "+completed.Title)
	writeJSON(w, http.StatusOK, toChoreDTO(completed))
}

// PATCH /chores/{id}/will-complete-by — sets (or clears, with null body) the
// claimant's committed completion date. 409 if the chore is not claimed by
// the caller or already completed.
func (s *Server) handleSetWillCompleteBy(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}
	choreID := chi.URLParam(r, "id")
	if choreID == "" {
		writeError(w, http.StatusBadRequest, "chore id is required")
		return
	}

	var req struct {
		WillCompleteBy *string `json:"will_complete_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var updated db.Chore
	err := s.UoW.InTx(r.Context(), func(q *db.Queries) error {
		var err error
		updated, err = chores.NewService(q).SetWillCompleteBy(r.Context(), choreID, u.ID, req.WillCompleteBy)
		return err
	})
	if err != nil {
		if errors.Is(err, chores.ErrNotModifiable) {
			writeError(w, http.StatusConflict, "chore cannot be modified")
			return
		}
		slog.ErrorContext(r.Context(), "set will_complete_by", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.Dispatcher.Dispatch(r.Context(), updated.ListID, u.ID, "Scheduled: "+updated.Title)
	writeJSON(w, http.StatusOK, toChoreDTO(updated))
}

// POST /chores/{id}/snooze — upserts a per-user snooze (8h). Returns 204.
// Idempotent: snoozing an already-snoozed chore refreshes the window.
func (s *Server) handleSnoozeChore(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}
	choreID := chi.URLParam(r, "id")
	if choreID == "" {
		writeError(w, http.StatusBadRequest, "chore id is required")
		return
	}

	err := s.UoW.InTx(r.Context(), func(q *db.Queries) error {
		return chores.NewService(q).Snooze(r.Context(), choreID, u.ID)
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "snooze chore", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /chores/{id}/claim — releases the caller's claim, clearing
// claimed_by, claimed_at, and will_complete_by. 409 if the caller isn't the
// current claimant or the chore is already completed.
func (s *Server) handleReleaseClaim(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}
	choreID := chi.URLParam(r, "id")
	if choreID == "" {
		writeError(w, http.StatusBadRequest, "chore id is required")
		return
	}

	var released db.Chore
	err := s.UoW.InTx(r.Context(), func(q *db.Queries) error {
		var err error
		released, err = chores.NewService(q).Release(r.Context(), choreID, u.ID)
		return err
	})
	if err != nil {
		if errors.Is(err, chores.ErrNotReleasable) {
			writeError(w, http.StatusConflict, "chore cannot be released")
			return
		}
		slog.ErrorContext(r.Context(), "release chore", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.Dispatcher.Dispatch(r.Context(), released.ListID, u.ID, "Released: "+released.Title)
	writeJSON(w, http.StatusOK, toChoreDTO(released))
}

// GET /chores/graveyard?q=<substring> — completed chores across the user's
// lists, optionally filtered by title substring (case-insensitive).
func (s *Server) handleGraveyard(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}
	query := r.URL.Query().Get("q")

	results, err := chores.NewService(s.UoW.Queries()).SearchGraveyard(r.Context(), u.ID, query)
	if err != nil {
		slog.ErrorContext(r.Context(), "graveyard", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	out := backlogResponse{Chores: make([]choreDTO, 0, len(results))}
	for _, c := range results {
		out.Chores = append(out.Chores, toChoreDTO(c))
	}
	writeJSON(w, http.StatusOK, out)
}
