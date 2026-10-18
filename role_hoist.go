package fluxer

import (
	"context"
	"net/http"
)

type RoleHoistPosition struct {
	ID            string `json:"id"`
	HoistPosition int    `json:"hoist_position"`
}

func (c *Client) ModifyRoleHoistPositions(ctx context.Context, guildID string, positions []RoleHoistPosition) error {
	return c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/roles/hoist-positions", positions, nil)
}

func (c *Client) ResetRoleHoistPositions(ctx context.Context, guildID string) error {
	return c.Do(ctx, http.MethodDelete, guildPath(guildID)+"/roles/hoist-positions", nil, nil)
}
