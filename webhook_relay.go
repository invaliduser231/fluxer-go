package fluxer

import (
	"context"
	"encoding/json"
	"net/http"
)

type headersKey struct{}

func withHeaders(ctx context.Context, h http.Header) context.Context {
	return context.WithValue(ctx, headersKey{}, h)
}

// SlackMessage is a Slack-compatible webhook payload for
// ExecuteSlackWebhook.
type SlackMessage struct {
	Text        string           `json:"text,omitempty"`
	Username    string           `json:"username,omitempty"`
	IconURL     string           `json:"icon_url,omitempty"`
	Attachments []map[string]any `json:"attachments,omitempty"`
}

// ExecuteGitHubWebhook forwards a GitHub webhook delivery to the /github
// endpoint of w. event and delivery are sent as the X-GitHub-Event and
// X-GitHub-Delivery headers, and payload is the body GitHub sent. The request is
// only retried on 429.
func (c *Client) ExecuteGitHubWebhook(ctx context.Context, w Webhook, threadID, event, delivery string, payload json.RawMessage) error {
	h := http.Header{"X-GitHub-Event": {event}}
	if delivery != "" {
		h.Set("X-GitHub-Delivery", delivery)
	}
	return c.Do(withHeaders(ctx, h), http.MethodPost, webhookPath(w.ID, w.Token)+"/github"+threadQuery(threadID), payload, nil)
}

// ExecuteSlackWebhook posts m to the Slack-compatible /slack endpoint of w.
func (c *Client) ExecuteSlackWebhook(ctx context.Context, w Webhook, threadID string, m SlackMessage) error {
	return c.Do(ctx, http.MethodPost, webhookPath(w.ID, w.Token)+"/slack"+threadQuery(threadID), m, nil)
}

// ExecuteInstatusWebhook forwards an Instatus notification to the /instatus
// endpoint of w. That endpoint takes no thread.
func (c *Client) ExecuteInstatusWebhook(ctx context.Context, w Webhook, payload json.RawMessage) error {
	return c.Do(ctx, http.MethodPost, webhookPath(w.ID, w.Token)+"/instatus", payload, nil)
}

// ExecuteLibredeskWebhook forwards a Libredesk webhook payload to the
// /libredesk endpoint of w.
func (c *Client) ExecuteLibredeskWebhook(ctx context.Context, w Webhook, threadID string, payload json.RawMessage) error {
	return c.Do(ctx, http.MethodPost, webhookPath(w.ID, w.Token)+"/libredesk"+threadQuery(threadID), payload, nil)
}
