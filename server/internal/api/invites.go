package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/invites"
	"github.com/dopatraman/hearth/server/internal/lists"
)

type inviteDTO struct {
	Code      string `json:"code"`
	ListID    string `json:"list_id"`
	ExpiresAt string `json:"expires_at"`
}

type acceptInviteResponse struct {
	ListID string `json:"list_id"`
}

func toInviteDTO(i db.Invite) inviteDTO {
	return inviteDTO{
		Code:      i.Code,
		ListID:    i.ListID,
		ExpiresAt: i.ExpiresAt,
	}
}

// POST /invites — authenticated. Looks up the caller's list (lazily
// provisioning it if needed) and creates a fresh 6-digit code valid for 24h.
func (s *Server) handleCreateInvite(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}

	var created db.Invite
	err := s.UoW.InTx(r.Context(), func(q *db.Queries) error {
		listID, err := lists.NewService(q).EnsurePersonalMembership(r.Context(), u.ID)
		if err != nil {
			return err
		}
		created, err = invites.NewService(q).Create(r.Context(), listID, u.ID)
		return err
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "create invite", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, toInviteDTO(created))
}

// POST /invites/{code}/accept — authenticated. Looks up the invite, verifies
// it's not expired, and (idempotently) adds the caller as a member of the
// invite's list. Returns the list_id so the client knows where it just joined.
func (s *Server) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}

	code := chi.URLParam(r, "code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	var listID string
	err := s.UoW.InTx(r.Context(), func(q *db.Queries) error {
		inv, err := invites.NewService(q).Lookup(r.Context(), code)
		if err != nil {
			return err
		}
		if err := q.AddMembership(r.Context(), db.AddMembershipParams{
			ListID: inv.ListID,
			UserID: u.ID,
		}); err != nil {
			return err
		}
		listID = inv.ListID
		return nil
	})
	if err != nil {
		if errors.Is(err, invites.ErrNotFound) {
			writeError(w, http.StatusNotFound, "invite not found")
			return
		}
		if errors.Is(err, invites.ErrExpired) {
			writeError(w, http.StatusGone, "invite expired")
			return
		}
		slog.ErrorContext(r.Context(), "accept invite", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, acceptInviteResponse{ListID: listID})
}
