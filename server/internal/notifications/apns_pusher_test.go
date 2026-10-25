package notifications

import (
	"context"
	"errors"
	"testing"

	"github.com/dopatraman/hearth/server/internal/db"
	"github.com/sideshow/apns2"
)

type fakeAPNSClient struct {
	byToken map[string]*apns2.Response
	errs    map[string]error
}

func (f *fakeAPNSClient) Push(n *apns2.Notification) (*apns2.Response, error) {
	if err, ok := f.errs[n.DeviceToken]; ok {
		return nil, err
	}
	return f.byToken[n.DeviceToken], nil
}

func TestAPNSPusher_MapsResponses(t *testing.T) {
	fake := &fakeAPNSClient{
		byToken: map[string]*apns2.Response{
			"ok-token":   {StatusCode: 200},
			"gone-token": {StatusCode: 410, Reason: "Unregistered"},
			"bad-token":  {StatusCode: 400, Reason: "BadDeviceToken"},
		},
		errs: map[string]error{
			"err-token": errors.New("transport boom"),
		},
	}
	p := &APNSPusher{client: fake, bundleID: "com.example.app"}

	tokens := []db.TokensForListRow{
		{Token: "ok-token", Platform: "ios"},
		{Token: "gone-token", Platform: "ios"},
		{Token: "bad-token", Platform: "ios"},
		{Token: "err-token", Platform: "ios"},
	}
	results := p.Push(context.Background(), tokens, Payload{Alert: "hi", ListID: "l1"})

	if len(results) != 4 {
		t.Fatalf("results: got %d want 4", len(results))
	}
	cases := []struct {
		token    string
		wantGone bool
		wantErr  bool
	}{
		{"ok-token", false, false},
		{"gone-token", true, false},
		{"bad-token", false, true},
		{"err-token", false, true},
	}
	for i, c := range cases {
		got := results[i]
		if got.Token != c.token {
			t.Errorf("results[%d].Token = %q want %q", i, got.Token, c.token)
		}
		if got.Gone != c.wantGone {
			t.Errorf("results[%d].Gone = %v want %v", i, got.Gone, c.wantGone)
		}
		if (got.Err != nil) != c.wantErr {
			t.Errorf("results[%d].Err = %v wantErr=%v", i, got.Err, c.wantErr)
		}
	}
}
