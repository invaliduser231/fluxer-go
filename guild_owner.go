package fluxer

import (
	"context"
	"net/http"
)

func (c *Client) DeleteGuild(ctx context.Context, guildID string) error {
	return c.Do(ctx, http.MethodPost, guildPath(guildID)+"/delete", map[string]any{}, nil)
}

func (c *Client) TransferOwnership(ctx context.Context, guildID, newOwnerID string) (*Guild, error) {
	var g Guild
	return &g, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/transfer-ownership", map[string]string{"new_owner_id": newOwnerID}, &g)
}
