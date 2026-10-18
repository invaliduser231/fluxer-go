package fluxer

import (
	"context"
	"net/http"
	"net/url"
)

const (
	WebhookIncoming        = 1
	WebhookChannelFollower = 2
)

type Webhook struct {
	ID        string `json:"id"`
	GuildID   string `json:"guild_id,omitempty"`
	ChannelID string `json:"channel_id"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar,omitempty"`
	Type      int    `json:"type"`
	Token     string `json:"token,omitempty"`
	User      *User  `json:"user,omitempty"`
}

type WebhookEdit struct {
	Name      *string `json:"name,omitempty"`
	Avatar    *string `json:"avatar,omitempty"`
	ChannelID *string `json:"channel_id,omitempty"`
}

type WebhookMessage struct {
	MessageSend
	Username    string   `json:"username,omitempty"`
	AvatarURL   string   `json:"avatar_url,omitempty"`
	ThreadName  string   `json:"thread_name,omitempty"`
	AppliedTags []string `json:"applied_tags,omitempty"`
}

func webhookPath(id, token string) string {
	path := "/webhooks/" + id
	if token != "" {
		path += "/" + url.PathEscape(token)
	}
	return path
}

func (c *Client) ChannelWebhooks(ctx context.Context, channelID string) ([]Webhook, error) {
	var hooks []Webhook
	return hooks, c.Do(ctx, http.MethodGet, channelPath(channelID)+"/webhooks", nil, &hooks)
}

func (c *Client) GuildWebhooks(ctx context.Context, guildID string) ([]Webhook, error) {
	var hooks []Webhook
	return hooks, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/webhooks", nil, &hooks)
}

func (c *Client) CreateWebhook(ctx context.Context, channelID, name, avatar string) (*Webhook, error) {
	body := map[string]string{"name": name}
	if avatar != "" {
		body["avatar"] = avatar
	}
	var w Webhook
	return &w, c.Do(ctx, http.MethodPost, channelPath(channelID)+"/webhooks", body, &w)
}

func (c *Client) Webhook(ctx context.Context, webhookID string) (*Webhook, error) {
	var w Webhook
	return &w, c.Do(ctx, http.MethodGet, webhookPath(webhookID, ""), nil, &w)
}

func (c *Client) ModifyWebhook(ctx context.Context, webhookID string, e WebhookEdit) (*Webhook, error) {
	var w Webhook
	return &w, c.Do(ctx, http.MethodPatch, webhookPath(webhookID, ""), e, &w)
}

func (c *Client) DeleteWebhook(ctx context.Context, webhookID string) error {
	return c.Do(ctx, http.MethodDelete, webhookPath(webhookID, ""), nil, nil)
}

func (c *Client) ExecuteWebhook(ctx context.Context, w Webhook, threadID string, m WebhookMessage, files []File) (*Message, error) {
	if m.Nonce == "" {
		m.Nonce = newNonce()
	}
	q := url.Values{"wait": {"true"}}
	if threadID != "" {
		q.Set("thread_id", threadID)
	}
	var msg Message
	return &msg, c.DoFiles(ctx, http.MethodPost, webhookPath(w.ID, w.Token)+"?"+q.Encode(), m, files, &msg)
}
