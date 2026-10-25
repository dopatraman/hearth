package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"

	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"
)

type APNSConfig struct {
	KeyID      string
	TeamID     string
	BundleID   string
	KeyP8Path  string
	Production bool
}

func APNSConfigFromEnv() (APNSConfig, bool) {
	c := APNSConfig{
		KeyID:     os.Getenv("APNS_KEY_ID"),
		TeamID:    os.Getenv("APNS_TEAM_ID"),
		BundleID:  os.Getenv("APNS_BUNDLE_ID"),
		KeyP8Path: os.Getenv("APNS_KEY_P8_PATH"),
	}
	if c.KeyID == "" || c.TeamID == "" || c.BundleID == "" || c.KeyP8Path == "" {
		return APNSConfig{}, false
	}
	c.Production = os.Getenv("APNS_PRODUCTION") == "1"
	return c, true
}

type apnsClient interface {
	Push(*apns2.Notification) (*apns2.Response, error)
}

type APNSPusher struct {
	client   apnsClient
	bundleID string
}

func NewAPNSPusher(cfg APNSConfig) (*APNSPusher, error) {
	keyBytes, err := os.ReadFile(cfg.KeyP8Path)
	if err != nil {
		return nil, err
	}
	authKey, err := token.AuthKeyFromBytes(keyBytes)
	if err != nil {
		return nil, err
	}
	tok := &token.Token{
		AuthKey: authKey,
		KeyID:   cfg.KeyID,
		TeamID:  cfg.TeamID,
	}
	client := apns2.NewTokenClient(tok)
	if cfg.Production {
		client = client.Production()
	} else {
		client = client.Development()
	}
	return &APNSPusher{client: client, bundleID: cfg.BundleID}, nil
}

func (p *APNSPusher) Push(ctx context.Context, tokens []db.TokensForListRow, pl Payload) []PushResult {
	results := make([]PushResult, 0, len(tokens))
	for _, t := range tokens {
		body := payload.NewPayload().
			AlertTitle("Hearth").
			AlertBody(pl.Alert).
			Sound("default").
			ContentAvailable().
			Custom("list_id", pl.ListID)
		if pl.LastEventID != "" {
			body = body.Custom("last_event_id", pl.LastEventID)
		}
		raw, err := json.Marshal(body)
		if err != nil {
			results = append(results, PushResult{Token: t.Token, Platform: t.Platform, Err: err})
			continue
		}
		n := &apns2.Notification{
			DeviceToken: t.Token,
			Topic:       p.bundleID,
			Payload:     raw,
		}
		resp, err := p.client.Push(n)
		result := PushResult{Token: t.Token, Platform: t.Platform}
		if err != nil {
			result.Err = err
		} else if resp == nil {
			result.Err = errors.New("apns: nil response")
		} else if resp.StatusCode == 410 || resp.Reason == "Unregistered" {
			result.Gone = true
		} else if resp.StatusCode >= 400 {
			result.Err = errors.New("apns: " + resp.Reason)
		}
		if result.Err != nil {
			slog.ErrorContext(ctx, "apns push", "token", t.Token, "err", result.Err)
		}
		results = append(results, result)
	}
	return results
}
