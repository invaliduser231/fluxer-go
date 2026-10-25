package fluxer

import (
	"context"
	"net/http"
)

// Application is an OAuth2 application, as returned for the bot's own
// application. Bot is the application's bot user when it has one.
type Application struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Icon                string   `json:"icon,omitempty"`
	Description         string   `json:"description,omitempty"`
	BotPublic           bool     `json:"bot_public"`
	BotRequireCodeGrant bool     `json:"bot_require_code_grant"`
	VerifyKey           string   `json:"verify_key,omitempty"`
	Owner               *User    `json:"owner"`
	Bot                 *User    `json:"bot,omitempty"`
	RedirectURIs        []string `json:"redirect_uris,omitempty"`
}

// GatewayBot is the gateway URL, the recommended number of shards and the
// session start limit for a bot.
type GatewayBot struct {
	URL               string            `json:"url"`
	Shards            int               `json:"shards"`
	SessionStartLimit SessionStartLimit `json:"session_start_limit"`
}

// SessionStartLimit reports how many gateway sessions the bot may still start
// before the limit resets after ResetAfter. MaxConcurrency is the number of
// shards that may identify at the same time.
type SessionStartLimit struct {
	Total          int `json:"total"`
	Remaining      int `json:"remaining"`
	ResetAfter     int `json:"reset_after"`
	MaxConcurrency int `json:"max_concurrency"`
}

// Application returns the application the bot belongs to. It calls
// GET /applications/@me.
func (c *Client) Application(ctx context.Context) (*Application, error) {
	var a Application
	return &a, c.Do(ctx, http.MethodGet, "/applications/@me", nil, &a)
}

// GatewayBot returns the gateway URL and sharding information for the bot. It
// calls GET /gateway/bot.
func (c *Client) GatewayBot(ctx context.Context) (*GatewayBot, error) {
	var g GatewayBot
	return &g, c.Do(ctx, http.MethodGet, "/gateway/bot", nil, &g)
}
