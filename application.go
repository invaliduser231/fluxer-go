package fluxer

import (
	"context"
	"net/http"
)

type Application struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Icon                string   `json:"icon,omitempty"`
	Description         string   `json:"description,omitempty"`
	BotPublic           bool     `json:"bot_public"`
	BotRequireCodeGrant bool     `json:"bot_require_code_grant"`
	VerifyKey           string   `json:"verify_key,omitempty"`
	Owner               *User    `json:"owner"`
	RedirectURIs        []string `json:"redirect_uris,omitempty"`
}

type GatewayBot struct {
	URL               string            `json:"url"`
	Shards            int               `json:"shards"`
	SessionStartLimit SessionStartLimit `json:"session_start_limit"`
}

type SessionStartLimit struct {
	Total          int `json:"total"`
	Remaining      int `json:"remaining"`
	ResetAfter     int `json:"reset_after"`
	MaxConcurrency int `json:"max_concurrency"`
}

func (c *Client) Application(ctx context.Context) (*Application, error) {
	var a Application
	return &a, c.Do(ctx, http.MethodGet, "/applications/@me", nil, &a)
}

func (c *Client) GatewayBot(ctx context.Context) (*GatewayBot, error) {
	var g GatewayBot
	return &g, c.Do(ctx, http.MethodGet, "/gateway/bot", nil, &g)
}
