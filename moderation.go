package fluxer

import (
	"context"
	"net/http"
	"time"
)

type GuildBan struct {
	User        User       `json:"user"`
	Reason      string     `json:"reason,omitempty"`
	ModeratorID string     `json:"moderator_id,omitempty"`
	BannedAt    time.Time  `json:"banned_at,omitzero"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type BanCreate struct {
	DeleteMessageSeconds int    `json:"delete_message_seconds"`
	BanDurationSeconds   int    `json:"ban_duration_seconds,omitempty"`
	Reason               string `json:"reason,omitempty"`
}

func (c *Client) Bans(ctx context.Context, guildID string) ([]GuildBan, error) {
	var bans []GuildBan
	return bans, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/bans", nil, &bans)
}

func (c *Client) Ban(ctx context.Context, guildID, userID string, b BanCreate) error {
	return c.Do(ctx, http.MethodPut, guildPath(guildID)+"/bans/"+userID, b, nil)
}

func (c *Client) Unban(ctx context.Context, guildID, userID string) error {
	return c.Do(ctx, http.MethodDelete, guildPath(guildID)+"/bans/"+userID, nil, nil)
}
