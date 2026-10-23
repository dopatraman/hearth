package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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
	}
}

type createChoreRequest struct {
	Title string  `json:"title"`
	Notes *string `json:"notes,omitempty"`
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
		created, err = chores.NewService(q).Create(r.Context(), listID, u.ID, req.Title, req.Notes)
		return err
	})
	if err != nil {
		if errors.Is(err, chores.ErrEmptyTitle) {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
		slog.ErrorContext(r.Context(), "create chore", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, toChoreDTO(created))
}

// GET /chores — returns the backlog (open chores) for the authenticated
// user's list. Lazily provisions the list if it doesn't exist, in which case
// the response is an empty array.
func (s *Server) handleListChores(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}

	var backlog []db.Chore
	err := s.UoW.InTx(r.Context(), func(q *db.Queries) error {
		listID, err := lists.NewService(q).EnsurePersonalMembership(r.Context(), u.ID)
		if err != nil {
			return err
		}
		backlog, err = chores.NewService(q).Backlog(r.Context(), listID)
		return err
	})
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
