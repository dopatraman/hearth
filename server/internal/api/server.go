package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/dopatraman/hearth/server/internal/auth"
	"github.com/dopatraman/hearth/server/internal/users"
)

// Server bundles the handler-layer dependencies and exposes Router.
// Zero-value is useful for narrow tests (e.g. /healthz only); production
// construction fills in every field.
type Server struct {
	Users         *users.Service
	AppleVerifier auth.AppleVerifier
	Session       *auth.SessionTokens
}

// Router returns the chi mux with all routes mounted.
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
	})

	return r
}
