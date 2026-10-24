package notifications

import (
	"context"
	"log/slog"
	"time"

	"github.com/dopatraman/hearth/server/internal/db"
)

type Dispatcher struct {
	UoW    db.UoW
	Pusher Pusher
}

func (d *Dispatcher) Dispatch(ctx context.Context, listID, actorID, alert string) {
	if d == nil || d.Pusher == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "dispatch panic", "err", r)
			}
		}()
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tokens, err := d.UoW.Queries().TokensForList(bgCtx, db.TokensForListParams{
			ListID:  listID,
			ActorID: actorID,
		})
		if err != nil {
			slog.ErrorContext(bgCtx, "tokens for list", "err", err, "list_id", listID)
			return
		}
		if len(tokens) == 0 {
			return
		}
		results := d.Pusher.Push(bgCtx, tokens, Payload{Alert: alert, ListID: listID})
		for _, r := range results {
			if r.Gone {
				if err := d.UoW.Queries().DeleteDeviceToken(bgCtx, r.Token); err != nil {
					slog.ErrorContext(bgCtx, "delete gone token", "err", err)
				}
			}
		}
	}()
}
