package notifications

import (
	"context"
	"errors"
	"strings"
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

type recordingAPNSClient struct {
	sent []*apns2.Notification
}

func (r *recordingAPNSClient) Push(n *apns2.Notification) (*apns2.Response, error) {
	r.sent = append(r.sent, n)
	return &apns2.Response{StatusCode: 200}, nil
}

func TestAPNSPusher_SilentWhenAlertEmpty(t *testing.T) {
	rec := &recordingAPNSClient{}
	p := &APNSPusher{client: rec, bundleID: "com.example.app"}
	tokens := []db.TokensForListRow{{Token: "t", Platform: "ios"}}

	p.Push(context.Background(), tokens, Payload{ListID: "l1"})
	p.Push(context.Background(), tokens, Payload{Alert: "hi", ListID: "l1"})

	silent, alert := rec.sent[0], rec.sent[1]
	if silent.PushType != apns2.PushTypeBackground || silent.Priority != apns2.PriorityLow {
		t.Errorf("silent: push type %q priority %d", silent.PushType, silent.Priority)
	}
	if strings.Contains(string(silent.Payload.([]byte)), "alert") {
		t.Errorf("silent payload has alert: %s", silent.Payload)
	}
	if alert.PushType != apns2.PushTypeAlert || alert.Priority != apns2.PriorityHigh {
		t.Errorf("alert: push type %q priority %d", alert.PushType, alert.Priority)
	}
	if !strings.Contains(string(alert.Payload.([]byte)), `"body":"hi"`) {
		t.Errorf("alert payload missing body: %s", alert.Payload)
	}
}
