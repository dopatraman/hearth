package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dopatraman/hearth/server/internal/api"
	"github.com/dopatraman/hearth/server/internal/auth"
	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/dopatraman/hearth/server/internal/notifications"
	"github.com/dopatraman/hearth/server/migrations"

	_ "modernc.org/sqlite"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	sqlitePath := envDefault("SQLITE_PATH", "./hearth.db")
	addr := envDefault("HEARTH_ADDR", ":8080")
	appleAud := envDefault("APPLE_AUDIENCE", "com.dopatraman.hearth")
	sessionSecret := []byte(os.Getenv("SESSION_SECRET"))

	if len(sessionSecret) < 32 {
		logger.Error("SESSION_SECRET must be at least 32 bytes", "got", len(sessionSecret))
		os.Exit(1)
	}

	database, err := openDB(sqlitePath)
	if err != nil {
		logger.Error("open db", "path", sqlitePath, "err", err)
		os.Exit(1)
	}
	defer database.Close()

	logger.Info("applying migrations", "path", sqlitePath)
	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), 30*time.Second)
	if err := migrations.Up(migrateCtx, database); err != nil {
		cancelMigrate()
		logger.Error("migrate", "err", err)
		os.Exit(1)
	}
	cancelMigrate()

	logger.Info("initializing apple verifier", "audience", appleAud)
	appleCtx, cancelApple := context.WithTimeout(context.Background(), 10*time.Second)
	appleVerifier, err := auth.NewAppleVerifier(appleCtx, appleAud)
	cancelApple()
	if err != nil {
		logger.Error("apple verifier", "err", err)
		os.Exit(1)
	}

	sessionTokens, err := auth.NewSessionTokens(sessionSecret, 30*24*time.Hour)
	if err != nil {
		logger.Error("session tokens", "err", err)
		os.Exit(1)
	}

	uow := &db.PoolUoW{Pool: database}
	var pusher notifications.Pusher = notifications.NoopPusher{}
	if cfg, ok := notifications.APNSConfigFromEnv(); ok {
		ap, err := notifications.NewAPNSPusher(cfg)
		if err != nil {
			logger.Error("apns init", "err", err)
			os.Exit(1)
		}
		pusher = ap
		logger.Info("apns pusher configured", "production", cfg.Production, "bundle_id", cfg.BundleID)
	} else {
		logger.Info("apns env not set — using noop pusher")
	}
	dispatcher := &notifications.Dispatcher{UoW: uow, Pusher: pusher}

	srv := &http.Server{
		Addr: addr,
		Handler: (&api.Server{
			UoW:           uow,
			AppleVerifier: appleVerifier,
			Session:       sessionTokens,
			Dispatcher:    dispatcher,
		}).Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
}

func openDB(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	for _, pragma := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	} {
		if _, err := database.Exec(pragma); err != nil {
			_ = database.Close()
			return nil, err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, err
	}
	return database, nil
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
