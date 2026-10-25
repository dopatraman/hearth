package api

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type logFields struct {
	mu     sync.Mutex
	userID string
}

type logFieldsKey struct{}

func logFieldsFromContext(ctx context.Context) *logFields {
	if v, ok := ctx.Value(logFieldsKey{}).(*logFields); ok {
		return v
	}
	return nil
}

func setLogUserID(ctx context.Context, id string) {
	if lf := logFieldsFromContext(ctx); lf != nil {
		lf.mu.Lock()
		lf.userID = id
		lf.mu.Unlock()
	}
}

type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := middleware.GetReqID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	if lf := logFieldsFromContext(ctx); lf != nil {
		lf.mu.Lock()
		uid := lf.userID
		lf.mu.Unlock()
		if uid != "" {
			r.AddAttrs(slog.String("user_id", uid))
		}
	} else if u, ok := userFromContext(ctx); ok {
		r.AddAttrs(slog.String("user_id", u.ID))
	}
	return h.Handler.Handle(ctx, r)
}

func NewContextSlogHandler(base slog.Handler) slog.Handler {
	return contextHandler{Handler: base}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		lf := &logFields{}
		ctx := context.WithValue(r.Context(), logFieldsKey{}, lf)
		r = r.WithContext(ctx)
		next.ServeHTTP(ww, r)
		slog.LogAttrs(r.Context(), slog.LevelInfo, "http",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.Status()),
			slog.Int64("latency_ms", time.Since(start).Milliseconds()),
			slog.Int("bytes", ww.BytesWritten()),
			slog.String("remote_ip", clientIP(r)),
		)
	})
}
