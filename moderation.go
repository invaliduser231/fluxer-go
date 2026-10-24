package fluxer

import (
	"context"
	"net/http"
	"time"
)

// GuildBan is a ban entry. ExpiresAt is nil for a permanent ban.
type GuildBan struct {
	User        User       `json:"user"`
	Reason      string     `json:"reason,omitempty"`
	ModeratorID string     `json:"moderator_id,omitempty"`
	BannedAt    time.Time  `json:"banned_at,omitzero"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// BanOptions configures Ban. DeleteMessageSeconds is always sent and removes the user's
// messages from that many seconds before the ban; zero keeps them. A zero
// BanDurationSeconds bans permanently.
type BanOptions struct {
	DeleteMessageSeconds int    `json:"delete_message_seconds"`
	BanDurationSeconds   int    `json:"ban_duration_seconds,omitempty"`
	Reason               string `json:"reason,omitempty"`
}

// Bans lists the bans of a guild.
func (c *Client) Bans(ctx context.Context, guildID string) ([]GuildBan, error) {
	var bans []GuildBan
	return bans, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/bans", nil, &bans)
}

// Ban bans a user from a guild.
func (c *Client) Ban(ctx context.Context, guildID, userID string, b BanOptions) error {
	return c.Do(ctx, http.MethodPut, guildPath(guildID)+"/bans/"+userID, b, nil)
}

// Unban lifts a user's ban.
func (c *Client) Unban(ctx context.Context, guildID, userID string) error {
	return c.Do(ctx, http.MethodDelete, guildPath(guildID)+"/bans/"+userID, nil, nil)
}
