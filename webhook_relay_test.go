package fluxer_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestWebhookRelays(t *testing.T) {
	srv, _ := setup(t)
	var got []string
	record := func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, r.URL.Path+"?"+r.URL.RawQuery+" "+r.Header.Get("X-GitHub-Event")+" "+strings.TrimSpace(string(body)))
		w.WriteHeader(http.StatusNoContent)
	}
	for _, kind := range []string{"github", "slack", "instatus", "libredesk"} {
		srv.HandleRaw("POST /v1/webhooks/{id}/{token}/"+kind, record)
	}
	c := srv.Client()
	c.Token = ""
	hook := fluxer.Webhook{ID: "5", Token: "tok"}
	ctx := context.Background()
	if err := c.ExecuteGitHubWebhook(ctx, hook, "60", "push", "d-1", json.RawMessage(`{"ref":"main"}`)); err != nil {
		t.Fatal(err)
	}
	if err := c.ExecuteSlackWebhook(ctx, hook, "", fluxer.SlackMessage{Text: "deployed"}); err != nil {
		t.Fatal(err)
	}
	if err := c.ExecuteInstatusWebhook(ctx, hook, json.RawMessage(`{"page":{}}`)); err != nil {
		t.Fatal(err)
	}
	if err := c.ExecuteLibredeskWebhook(ctx, hook, "", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	want := []string{
		`/v1/webhooks/5/tok/github?thread_id=60 push {"ref":"main"}`,
		`/v1/webhooks/5/tok/slack?  {"text":"deployed"}`,
		`/v1/webhooks/5/tok/instatus?  {"page":{}}`,
		`/v1/webhooks/5/tok/libredesk?  {}`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s", strings.Join(got, "\n"))
	}
}
