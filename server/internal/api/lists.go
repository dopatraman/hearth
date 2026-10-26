package api

import (
	"log/slog"
	"net/http"
)

type memberDTO struct {
	ID          string  `json:"id"`
	DisplayName *string `json:"display_name"`
}

type membersResponse struct {
	Members []memberDTO `json:"members"`
}

// GET /lists/mine/members — returns the distinct users (including self) who
// share any list with the caller. Empty array when the caller has no
// memberships yet.
func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}

	rows, err := s.UoW.Queries().MembersForUser(r.Context(), u.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "members for user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	out := membersResponse{Members: make([]memberDTO, 0, len(rows))}
	for _, m := range rows {
		out.Members = append(out.Members, memberDTO{ID: m.ID, DisplayName: m.DisplayName})
	}
	writeJSON(w, http.StatusOK, out)
}
