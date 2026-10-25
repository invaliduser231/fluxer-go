package fluxer

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// WebhookIncoming and WebhookChannelFollower are the values of Webhook.Type.
const (
	WebhookIncoming        = 1
	WebhookChannelFollower = 2
)

// Webhook is a Fluxer webhook. ExecuteWebhook and the methods ending in
// WithToken need ID and Token; ParseWebhookURL extracts both from a webhook
// URL.
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

// WebhookEdit holds the changes for ModifyWebhook. Nil fields are left
// unchanged.
type WebhookEdit struct {
	Name      *string `json:"name,omitempty"`
	Avatar    *string `json:"avatar,omitempty"`
	ChannelID *string `json:"channel_id,omitempty"`
}

// WebhookMessage is a message sent through a webhook. Username and AvatarURL
// replace the webhook's name and avatar for this message only.
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

// ChannelWebhooks lists the webhooks of a channel. It calls
// GET /channels/{id}/webhooks.
func (c *Client) ChannelWebhooks(ctx context.Context, channelID string) ([]Webhook, error) {
	var hooks []Webhook
	return hooks, c.Do(ctx, http.MethodGet, channelPath(channelID)+"/webhooks", nil, &hooks)
}

// GuildWebhooks lists the webhooks of a guild. It calls
// GET /guilds/{id}/webhooks.
func (c *Client) GuildWebhooks(ctx context.Context, guildID string) ([]Webhook, error) {
	var hooks []Webhook
	return hooks, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/webhooks", nil, &hooks)
}

// CreateWebhook creates a webhook in channelID. An empty avatar is left out of
// the request.
func (c *Client) CreateWebhook(ctx context.Context, channelID, name, avatar string) (*Webhook, error) {
	body := map[string]string{"name": name}
	if avatar != "" {
		body["avatar"] = avatar
	}
	var w Webhook
	return &w, c.Do(ctx, http.MethodPost, channelPath(channelID)+"/webhooks", body, &w)
}

// Webhook returns a webhook by ID. It calls GET /webhooks/{id}.
func (c *Client) Webhook(ctx context.Context, webhookID string) (*Webhook, error) {
	var w Webhook
	return &w, c.Do(ctx, http.MethodGet, webhookPath(webhookID, ""), nil, &w)
}

// ModifyWebhook changes a webhook. Setting ChannelID moves it to another
// channel.
func (c *Client) ModifyWebhook(ctx context.Context, webhookID string, e WebhookEdit) (*Webhook, error) {
	var w Webhook
	return &w, c.Do(ctx, http.MethodPatch, webhookPath(webhookID, ""), e, &w)
}

// DeleteWebhook deletes a webhook by ID.
func (c *Client) DeleteWebhook(ctx context.Context, webhookID string) error {
	return c.Do(ctx, http.MethodDelete, webhookPath(webhookID, ""), nil, nil)
}

// ExecuteWebhook posts m through w and returns the created message. When
// threadID is non-empty the message goes to that thread. files are sent as
// multipart attachments, see DoFiles. A nonce is generated when m.Nonce is
// empty, which makes the request safe to retry.
func (c *Client) ExecuteWebhook(ctx context.Context, w Webhook, threadID string, m WebhookMessage, files []File) (*Message, error) {
	var msg Message
	return &msg, c.executeWebhook(ctx, w, threadID, m, files, true, &msg)
}

// ExecuteWebhookNoWait posts m through w like ExecuteWebhook but does not ask Fluxer to
// return the message, which makes the call a little faster.
func (c *Client) ExecuteWebhookNoWait(ctx context.Context, w Webhook, threadID string, m WebhookMessage, files []File) error {
	return c.executeWebhook(ctx, w, threadID, m, files, false, nil)
}

func (c *Client) executeWebhook(ctx context.Context, w Webhook, threadID string, m WebhookMessage, files []File, wait bool, out any) error {
	if m.Nonce == "" {
		m.Nonce = newNonce()
	}
	q := url.Values{"wait": {strconv.FormatBool(wait)}}
	if threadID != "" {
		q.Set("thread_id", threadID)
	}
	return c.DoFiles(ctx, http.MethodPost, webhookPath(w.ID, w.Token)+"?"+q.Encode(), m, files, out)
}

// ParseWebhookURL extracts the webhook ID and token from a URL that contains
// /webhooks/{id}/{token}. Further path segments after the token are
// allowed.
func ParseWebhookURL(raw string) (Webhook, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Webhook{}, err
	}
	parts := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	for i := len(parts) - 3; i >= 0; i-- {
		if parts[i] == "webhooks" && isSnowflake(parts[i+1]) {
			token, err := url.PathUnescape(parts[i+2])
			if err != nil || token == "" {
				break
			}
			return Webhook{ID: parts[i+1], Token: token}, nil
		}
	}
	return Webhook{}, errors.New("fluxer: not a webhook url")
}

func threadQuery(threadID string) string {
	if threadID == "" {
		return ""
	}
	return "?" + url.Values{"thread_id": {threadID}}.Encode()
}

// WebhookWithToken returns w through its token route,
// GET /webhooks/{id}/{token}.
func (c *Client) WebhookWithToken(ctx context.Context, w Webhook) (*Webhook, error) {
	var out Webhook
	return &out, c.Do(ctx, http.MethodGet, webhookPath(w.ID, w.Token), nil, &out)
}

// ModifyWebhookWithToken changes the name or avatar of w through its token
// route. A nil argument leaves that field unchanged.
func (c *Client) ModifyWebhookWithToken(ctx context.Context, w Webhook, name, avatar *string) (*Webhook, error) {
	var out Webhook
	return &out, c.Do(ctx, http.MethodPatch, webhookPath(w.ID, w.Token), WebhookEdit{Name: name, Avatar: avatar}, &out)
}

// DeleteWebhookWithToken deletes w through its token route.
func (c *Client) DeleteWebhookWithToken(ctx context.Context, w Webhook) error {
	return c.Do(ctx, http.MethodDelete, webhookPath(w.ID, w.Token), nil, nil)
}

// WebhookMessage returns a message that w has sent. Pass threadID when the
// message is in a thread.
func (c *Client) WebhookMessage(ctx context.Context, w Webhook, threadID, messageID string) (*Message, error) {
	var msg Message
	return &msg, c.Do(ctx, http.MethodGet, webhookPath(w.ID, w.Token)+"/messages/"+messageID+threadQuery(threadID), nil, &msg)
}

// EditWebhookMessage edits a message that w has sent. Pass threadID when the
// message is in a thread.
func (c *Client) EditWebhookMessage(ctx context.Context, w Webhook, threadID, messageID string, m MessageEdit) (*Message, error) {
	var msg Message
	return &msg, c.Do(ctx, http.MethodPatch, webhookPath(w.ID, w.Token)+"/messages/"+messageID+threadQuery(threadID), m, &msg)
}

// DeleteWebhookMessage deletes a message that w has sent. Pass threadID when
// the message is in a thread.
func (c *Client) DeleteWebhookMessage(ctx context.Context, w Webhook, threadID, messageID string) error {
	return c.Do(ctx, http.MethodDelete, webhookPath(w.ID, w.Token)+"/messages/"+messageID+threadQuery(threadID), nil, nil)
}
