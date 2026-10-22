package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/items"
	"github.com/dopatraman/hearth/server/internal/lists"
)

type itemDTO struct {
	ID             string  `json:"id"`
	ListID         string  `json:"list_id"`
	CreatorID      string  `json:"creator_id"`
	Title          string  `json:"title"`
	Notes          *string `json:"notes"`
	State          string  `json:"state"`
	ClaimedBy      *string `json:"claimed_by"`
	ClaimedAt      *string `json:"claimed_at"`
	AssignedBy     *string `json:"assigned_by"`
	WillCompleteBy *string `json:"will_complete_by"`
	CompletedBy    *string `json:"completed_by"`
	CompletedAt    *string `json:"completed_at"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func toItemDTO(i db.Item) itemDTO {
	return itemDTO{
		ID:             i.ID,
		ListID:         i.ListID,
		CreatorID:      i.CreatorID,
		Title:          i.Title,
		Notes:          i.Notes,
		State:          i.State,
		ClaimedBy:      i.ClaimedBy,
		ClaimedAt:      i.ClaimedAt,
		AssignedBy:     i.AssignedBy,
		WillCompleteBy: i.WillCompleteBy,
		CompletedBy:    i.CompletedBy,
		CompletedAt:    i.CompletedAt,
		CreatedAt:      i.CreatedAt,
		UpdatedAt:      i.UpdatedAt,
	}
}

type createItemRequest struct {
	Title string  `json:"title"`
	Notes *string `json:"notes,omitempty"`
}

type backlogResponse struct {
	Items []itemDTO `json:"items"`
}

// POST /items — creates a new item on the authenticated user's list (lazily
// provisioning the list if it doesn't yet exist). Item + 'created' event
// written in a single transaction.
func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}

	var req createItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var created db.Item
	err := s.UoW.InTx(r.Context(), func(q *db.Queries) error {
		listID, err := lists.NewService(q).EnsurePersonalMembership(r.Context(), u.ID)
		if err != nil {
			return err
		}
		created, err = items.NewService(q).Create(r.Context(), listID, u.ID, req.Title, req.Notes)
		return err
	})
	if err != nil {
		if errors.Is(err, items.ErrEmptyTitle) {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
		slog.ErrorContext(r.Context(), "create item", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, toItemDTO(created))
}

// GET /items — returns the backlog (open items) for the authenticated user's
// list. Lazily provisions the list if it doesn't exist, in which case the
// response is an empty array.
func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}

	var backlog []db.Item
	err := s.UoW.InTx(r.Context(), func(q *db.Queries) error {
		listID, err := lists.NewService(q).EnsurePersonalMembership(r.Context(), u.ID)
		if err != nil {
			return err
		}
		backlog, err = items.NewService(q).Backlog(r.Context(), listID)
		return err
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list items", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	out := backlogResponse{Items: make([]itemDTO, 0, len(backlog))}
	for _, it := range backlog {
		out.Items = append(out.Items, toItemDTO(it))
	}
	writeJSON(w, http.StatusOK, out)
}
