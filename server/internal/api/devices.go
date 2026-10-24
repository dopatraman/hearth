package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dopatraman/hearth/server/internal/db"
)

type registerDeviceRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

func (s *Server) handleRegisterDevice(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}
	var req registerDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}
	if req.Platform != "ios" && req.Platform != "watchos" {
		writeError(w, http.StatusBadRequest, "platform must be 'ios' or 'watchos'")
		return
	}
	err := s.UoW.Queries().UpsertDeviceToken(r.Context(), db.UpsertDeviceTokenParams{
		UserID:   u.ID,
		Token:    req.Token,
		Platform: req.Platform,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "upsert device token", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type unregisterDeviceRequest struct {
	Token string `json:"token"`
}

func (s *Server) handleUnregisterDevice(w http.ResponseWriter, r *http.Request) {
	var req unregisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}
	if err := s.UoW.Queries().DeleteDeviceToken(r.Context(), req.Token); err != nil {
		slog.ErrorContext(r.Context(), "delete device token", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type setNotificationsRequest struct {
	Enabled bool `json:"enabled"`
}

func (s *Server) handleSetNotifications(w http.ResponseWriter, r *http.Request) {
	u, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "no user in context")
		return
	}
	var req setNotificationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	enabled := int64(0)
	if req.Enabled {
		enabled = 1
	}
	if err := s.UoW.Queries().SetNotificationsEnabled(r.Context(), db.SetNotificationsEnabledParams{
		NotificationsEnabled: enabled,
		ID:                   u.ID,
	}); err != nil {
		slog.ErrorContext(r.Context(), "set notifications", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
