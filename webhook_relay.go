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

type SlackMessage struct {
	Text        string           `json:"text,omitempty"`
	Username    string           `json:"username,omitempty"`
	IconURL     string           `json:"icon_url,omitempty"`
	Attachments []map[string]any `json:"attachments,omitempty"`
}

func (c *Client) ExecuteGitHubWebhook(ctx context.Context, w Webhook, threadID, event, delivery string, payload json.RawMessage) error {
	h := http.Header{"X-GitHub-Event": {event}}
	if delivery != "" {
		h.Set("X-GitHub-Delivery", delivery)
	}
	return c.Do(withHeaders(ctx, h), http.MethodPost, webhookPath(w.ID, w.Token)+"/github"+threadQuery(threadID), payload, nil)
}

func (c *Client) ExecuteSlackWebhook(ctx context.Context, w Webhook, threadID string, m SlackMessage) error {
	return c.Do(ctx, http.MethodPost, webhookPath(w.ID, w.Token)+"/slack"+threadQuery(threadID), m, nil)
}

func (c *Client) ExecuteInstatusWebhook(ctx context.Context, w Webhook, payload json.RawMessage) error {
	return c.Do(ctx, http.MethodPost, webhookPath(w.ID, w.Token)+"/instatus", payload, nil)
}

func (c *Client) ExecuteLibredeskWebhook(ctx context.Context, w Webhook, threadID string, payload json.RawMessage) error {
	return c.Do(ctx, http.MethodPost, webhookPath(w.ID, w.Token)+"/libredesk"+threadQuery(threadID), payload, nil)
}
