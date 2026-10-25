package notifications

import (
	"context"
	"log/slog"

	"github.com/dopatraman/hearth/server/internal/db"
)

type Payload struct {
	Alert       string
	ListID      string
	LastEventID string
}

type PushResult struct {
	Token    string
	Platform string
	Gone     bool
	Err      error
}

type Pusher interface {
	Push(ctx context.Context, tokens []db.TokensForListRow, p Payload) []PushResult
}

type NoopPusher struct{}

func (NoopPusher) Push(ctx context.Context, tokens []db.TokensForListRow, p Payload) []PushResult {
	slog.InfoContext(ctx, "noop push", "recipients", len(tokens), "alert", p.Alert, "list_id", p.ListID)
	results := make([]PushResult, 0, len(tokens))
	for _, t := range tokens {
		results = append(results, PushResult{Token: t.Token, Platform: t.Platform})
	}
	return results
}
