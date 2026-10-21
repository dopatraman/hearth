package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dopatraman/hearth/server/internal/db"
)

type ctxKey int

const (
	ctxKeyUser ctxKey = iota
)

type authAppleRequest struct {
	IDToken     string  `json:"id_token"`
	DisplayName *string `json:"display_name,omitempty"`
}

type userDTO struct {
	ID                   string  `json:"id"`
	DisplayName          *string `json:"display_name"`
	NotificationsEnabled bool    `json:"notifications_enabled"`
}

type authAppleResponse struct {
	SessionToken string  `json:"session_token"`
	User         userDTO `json:"user"`
}

func toUserDTO(u db.User) userDTO {
	return userDTO{
		ID:                   u.ID,
		DisplayName:          u.DisplayName,
		NotificationsEnabled: u.NotificationsEnabled != 0,
	}
}

func (s *Server) handleAuthApple(w http.ResponseWriter, r *http.Request) {
	var req authAppleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.IDToken == "" {
		writeError(w, http.StatusBadRequest, "id_token is required")
		return
	}

	identity, err := s.AppleVerifier.Verify(r.Context(), req.IDToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid id_token")
		return
	}

	user, err := s.Users.UpsertByAppleSub(r.Context(), identity.Sub, req.DisplayName)
	if err != nil {
		slog.ErrorContext(r.Context(), "upsert user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	tok, err := s.Session.Issue(user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "issue session", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, authAppleResponse{
		SessionToken: tok,
		User:         toUserDTO(user),
	})
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		token := strings.TrimPrefix(h, "Bearer ")

		userID, err := s.Session.Verify(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid session")
			return
		}

		user, err := s.Users.ByID(r.Context(), userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusUnauthorized, "user not found")
				return
			}
			slog.ErrorContext(r.Context(), "lookup user", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// userFromContext returns the authenticated user set by requireAuth.
func userFromContext(ctx context.Context) (db.User, bool) {
	u, ok := ctx.Value(ctxKeyUser).(db.User)
	return u, ok
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}
	writeJSON(w, http.StatusOK, toUserDTO(u))
}

// --- helpers ---

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
