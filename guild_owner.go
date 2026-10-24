package fluxer

import (
	"context"
	"net/http"
)

// DeleteGuild permanently deletes a guild (POST /guilds/{id}/delete). Only the owner can
// do this.
func (c *Client) DeleteGuild(ctx context.Context, guildID string) error {
	return c.Do(ctx, http.MethodPost, guildPath(guildID)+"/delete", map[string]any{}, nil)
}

// TransferOwnership makes newOwnerID the owner of a guild and returns the updated guild.
func (c *Client) TransferOwnership(ctx context.Context, guildID, newOwnerID string) (*Guild, error) {
	var g Guild
	return &g, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/transfer-ownership", map[string]string{"new_owner_id": newOwnerID}, &g)
}
