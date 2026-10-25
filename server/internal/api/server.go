package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/dopatraman/hearth/server/internal/auth"
	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/notifications"
)

type Server struct {
	UoW           db.UoW
	AppleVerifier auth.AppleVerifier
	Session       *auth.SessionTokens
	Dispatcher    *notifications.Dispatcher
}

func (s *Server) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", healthz)

	r.Post("/auth/apple", s.handleAuthApple)

	r.Group(func(r chi.Router) {
		r.Use(s.requireAuth)

		r.Get("/me", s.handleMe)

		r.Get("/chores", s.handleListChores)
		r.Post("/chores", s.handleCreateChore)
		r.Patch("/chores/{id}/claim", s.handleClaimChore)
		r.Delete("/chores/{id}/claim", s.handleReleaseClaim)
		r.Post("/chores/{id}/complete", s.handleCompleteChore)
		r.Post("/chores/{id}/snooze", s.handleSnoozeChore)
		r.Patch("/chores/{id}/will-complete-by", s.handleSetWillCompleteBy)
		r.Get("/chores/graveyard", s.handleGraveyard)

		r.Post("/invites", s.handleCreateInvite)
		r.Post("/invites/{code}/accept", s.handleAcceptInvite)

		r.Post("/devices", s.handleRegisterDevice)
		r.Delete("/devices", s.handleUnregisterDevice)
		r.Put("/me/notifications", s.handleSetNotifications)
	})

	return r
}
