package fluxer_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/fluxertest"
)

func setup(t *testing.T) (*fluxertest.Server, *fluxer.Client) {
	t.Helper()
	srv := fluxertest.NewServer("1000.secret")
	t.Cleanup(srv.Close)
	return srv, srv.Client()
}

func TestSendMessage(t *testing.T) {
	srv, c := setup(t)
	color := 0xcba6f7
	msg, err := c.SendMessage(context.Background(), "42", fluxer.MessageSend{
		Content:          "hi",
		Embeds:           []fluxer.Embed{{Title: "T", Color: &color}},
		MessageReference: &fluxer.MessageReference{MessageID: "7"},
		AllowedMentions:  &fluxer.AllowedMentions{Parse: []string{"users"}},
		Nonce:            "n1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.ID == "" || msg.ChannelID != "42" || msg.Type != fluxer.MessageReply {
		t.Errorf("got %+v", msg)
	}
	req := srv.Requests()[0]
	want := `{"content":"hi","embeds":[{"title":"T","color":13346551}],"message_reference":{"message_id":"7"},"allowed_mentions":{"parse":["users"],"replied_user":false},"nonce":"n1"}`
	if req.Path != "/v1/channels/42/messages" || strings.TrimSpace(req.Body) != want {
		t.Errorf("got %s %s", req.Path, req.Body)
	}
}

func TestReply(t *testing.T) {
	srv, c := setup(t)
	to := &fluxer.Message{ID: "7", ChannelID: "42", GuildID: "3"}
	if _, err := c.Reply(context.Background(), to, fluxer.MessageSend{Content: "pong"}); err != nil {
		t.Fatal(err)
	}
	if body := srv.Requests()[0].Body; !strings.Contains(body, `"message_reference":{"message_id":"7","channel_id":"42","guild_id":"3"}`) {
		t.Errorf("body %s", body)
	}
}

func TestEditMessageSendsOnlySetFields(t *testing.T) {
	srv, c := setup(t)
	ctx := context.Background()
	msg, err := c.SendText(ctx, "1", "before")
	if err != nil {
		t.Fatal(err)
	}
	content := "after"
	edited, err := c.EditMessage(ctx, "1", msg.ID, fluxer.MessageEdit{Content: &content})
	if err != nil || edited.Content != "after" || edited.EditedTimestamp == nil {
		t.Fatalf("got %+v, %v", edited, err)
	}
	if body := strings.TrimSpace(srv.Requests()[1].Body); body != `{"content":"after"}` {
		t.Errorf("body %s", body)
	}
}

func TestMessagesQuery(t *testing.T) {
	srv, c := setup(t)
	for _, id := range []string{"10", "11", "12", "13"} {
		srv.AddMessage(fluxer.Message{ID: id, ChannelID: "1"})
	}
	msgs, err := c.Messages(context.Background(), "1", fluxer.MessageQuery{Before: "13", Limit: 2})
	if err != nil || len(msgs) != 2 || msgs[0].ID != "12" || msgs[1].ID != "11" {
		t.Errorf("got %+v, %v", msgs, err)
	}
	if path := srv.Requests()[0].Path; path != "/v1/channels/1/messages" {
		t.Errorf("path %s", path)
	}
}

func TestReactionPath(t *testing.T) {
	srv, c := setup(t)
	if err := c.AddReaction(context.Background(), "1", "2", "\U0001FA99"); err != nil {
		t.Fatal(err)
	}
	if err := c.AddReaction(context.Background(), "1", "2", "coin:123"); err != nil {
		t.Fatal(err)
	}
	reqs := srv.Requests()
	if reqs[0].Path != "/v1/channels/1/messages/2/reactions/%F0%9F%AA%99/@me" || reqs[1].Path != "/v1/channels/1/messages/2/reactions/coin:123/@me" {
		t.Errorf("got %s and %s", reqs[0].Path, reqs[1].Path)
	}
}

func TestRetryAfterRateLimit(t *testing.T) {
	srv, c := setup(t)
	srv.RateLimitOnce("POST /v1/channels/{channel}/messages", 0.05)
	start := time.Now()
	if _, err := c.SendText(context.Background(), "1", "x"); err != nil {
		t.Fatal(err)
	}
	if n := len(srv.Requests()); n != 2 {
		t.Errorf("got %d requests, want 2", n)
	}
	if time.Since(start) < 50*time.Millisecond {
		t.Error("did not wait for retry_after")
	}
}

func TestWaitsForExhaustedBucket(t *testing.T) {
	srv, c := setup(t)
	srv.AddMessage(fluxer.Message{ID: "2", ChannelID: "1"})
	srv.AddMessage(fluxer.Message{ID: "3", ChannelID: "1"})
	srv.AddMessage(fluxer.Message{ID: "3", ChannelID: "9"})
	srv.ExhaustOnce("DELETE /v1/channels/{channel}/messages/{message}", 0.1)
	ctx := context.Background()
	if err := c.DeleteMessage(ctx, "1", "2"); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if err := c.DeleteMessage(ctx, "1", "3"); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) < 80*time.Millisecond {
		t.Error("second request did not wait for the bucket to reset")
	}
	if err := c.DeleteMessage(ctx, "9", "3"); err != nil {
		t.Fatal(err)
	}
}

func TestAPIError(t *testing.T) {
	srv, _ := setup(t)
	c := srv.Client()
	c.Token = "wrong"
	_, err := c.CurrentUser(context.Background())
	var apiErr *fluxer.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 401 || apiErr.Code != "UNAUTHORIZED" {
		t.Errorf("got %v", err)
	}
}

func TestGiveUpAfterRetries(t *testing.T) {
	srv, c := setup(t)
	c.MaxRetries = 0
	srv.RateLimitOnce("GET /v1/users/@me", 0.01)
	_, err := c.CurrentUser(context.Background())
	var apiErr *fluxer.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "RATE_LIMITED" || apiErr.RetryAfter < 10*time.Millisecond {
		t.Errorf("got %v", err)
	}
	if fluxer.ErrorCode(err) != "RATE_LIMITED" {
		t.Errorf("code %q", fluxer.ErrorCode(err))
	}
}

func TestContextCancelsWait(t *testing.T) {
	srv, c := setup(t)
	srv.RateLimitOnce("GET /v1/users/@me", 5)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := c.CurrentUser(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v", err)
	}
}

func TestRejectedTokenStopsRequests(t *testing.T) {
	srv, _ := setup(t)
	c := srv.Client()
	c.Token = "1000.wrong"
	for range 3 {
		if _, err := c.CurrentUser(context.Background()); !errors.Is(err, fluxer.ErrTokenRejected) {
			t.Fatalf("got %v", err)
		}
	}
	if n := len(srv.Requests()); n != 1 {
		t.Errorf("got %d requests after a 401, want 1", n)
	}
}

func TestRetriesSendWithNonce(t *testing.T) {
	srv, c := setup(t)
	srv.FailOnce("POST /v1/channels/{channel}/messages", 503, map[string]string{"code": "SERVICE_UNAVAILABLE", "message": "busy"})
	if _, err := c.SendText(context.Background(), "1", "x"); err != nil {
		t.Fatal(err)
	}
	reqs := srv.Requests()
	if len(reqs) != 2 || reqs[0].Body != reqs[1].Body || !strings.Contains(reqs[0].Body, `"nonce":"`) {
		t.Errorf("got %v", reqs)
	}
}

func TestFieldErrors(t *testing.T) {
	srv, c := setup(t)
	srv.FailOnce("POST /v1/channels/{channel}/messages", 400, map[string]any{
		"code": "INVALID_FORM_BODY", "message": "Invalid form body",
		"errors": []map[string]string{{"path": "embeds.0.title", "code": "BASE_TYPE_MAX_LENGTH", "message": "Must be 256 or fewer in length."}},
	})
	_, err := c.SendText(context.Background(), "1", "x")
	var apiErr *fluxer.APIError
	if !errors.As(err, &apiErr) || len(apiErr.Errors) != 1 || apiErr.Errors[0].Path != "embeds.0.title" {
		t.Fatalf("got %v", err)
	}
	if !strings.HasSuffix(err.Error(), "(embeds.0.title: Must be 256 or fewer in length.)") {
		t.Errorf("got %q", err)
	}
	srv.FailOnce("DELETE /v1/channels/{channel}/messages/{message}", 404, "not json")
	err = c.DeleteMessage(context.Background(), "1", "2")
	if !errors.As(err, &apiErr) || apiErr.Code != "NOT_FOUND" {
		t.Errorf("got %v", err)
	}
}

func TestAuditLogReason(t *testing.T) {
	srv, c := setup(t)
	srv.AddMessage(fluxer.Message{ID: "2", ChannelID: "1"})
	ctx := fluxer.WithReason(context.Background(), "spam\nagain")
	if err := c.DeleteMessage(ctx, "1", "2"); err != nil {
		t.Fatal(err)
	}
	if r := srv.Requests()[0].Reason; r != "spam again" {
		t.Errorf("reason %q", r)
	}
}

func TestAuditLogWebhookAvatar(t *testing.T) {
	srv, c := setup(t)
	srv.Handle("GET /v1/guilds/{guild}/audit-logs", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"audit_log_entries":[{"id":"1","action_type":50}],"users":[],"webhooks":[{"id":"5","guild_id":"9","channel_id":"2","name":"CI","avatar_hash":"abc","type":1}]}`))
	})
	l, err := c.AuditLog(context.Background(), "9", fluxer.AuditLogQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Entries) != 1 || len(l.Webhooks) != 1 || l.Webhooks[0].Avatar != "abc" || l.Webhooks[0].Name != "CI" {
		t.Errorf("got %+v", l)
	}
}

func TestSendMessageFiles(t *testing.T) {
	srv, c := setup(t)
	msg, err := c.SendMessageFiles(context.Background(), "20", fluxer.MessageSend{Content: "two files"}, []fluxer.File{
		{Name: "cat.png", Data: []byte("png bytes"), Description: "a cat"},
		{Name: "notes.txt", Data: []byte("hello")},
	})
	if err != nil {
		t.Fatal(err)
	}
	stored := srv.Messages("20")[0]
	if msg.Content != "two files" || len(stored.Attachments) != 2 || stored.Attachments[0].Filename != "cat.png" ||
		stored.Attachments[0].Size != 9 || stored.Attachments[1].Filename != "notes.txt" || !strings.HasSuffix(stored.Attachments[1].URL, "/notes.txt") {
		t.Errorf("got %+v", stored)
	}
	body := srv.Requests()[0].Body
	for _, want := range []string{`name="payload_json"`, `"nonce":"`, `name="files[1]"; filename="notes.txt"`, `"attachments":[{"id":0,"filename":"cat.png","description":"a cat"}`} {
		if !strings.Contains(body, want) {
			t.Errorf("body has no %s:\n%s", want, body)
		}
	}
}

func TestDiscover(t *testing.T) {
	srv, _ := setup(t)
	inst, err := fluxer.Discover(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if inst.API != srv.API() || inst.Gateway != srv.GatewayURL() || !inst.Features.SelfHosted {
		t.Errorf("got %+v", inst)
	}
	c := inst.NewClient(srv.Token)
	if u, err := c.CurrentUser(context.Background()); err != nil || u.ID != "1000" {
		t.Errorf("got %+v, %v", u, err)
	}
}

func TestNormalizeOrigin(t *testing.T) {
	for in, want := range map[string]string{
		"fluxer.example.com":          "https://fluxer.example.com",
		"HTTPS://Fluxer.Example.com/": "https://fluxer.example.com",
		"http://localhost:8080":       "http://localhost:8080",
	} {
		if got, err := fluxer.NormalizeOrigin(in); err != nil || got != want {
			t.Errorf("%s: got %q, %v", in, got, err)
		}
	}
	for _, bad := range []string{"ftp://x.org", "https://x.org/app", "https://user@x.org"} {
		if _, err := fluxer.NormalizeOrigin(bad); err == nil {
			t.Errorf("%s: expected error", bad)
		}
	}
}
