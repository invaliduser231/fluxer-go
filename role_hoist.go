package fluxer

import (
	"context"
	"net/http"
)

// RoleHoistPosition sets where one hoisted role appears in the member list, independent
// of its permission position.
type RoleHoistPosition struct {
	ID            string `json:"id"`
	HoistPosition int    `json:"hoist_position"`
}

// ModifyRoleHoistPositions sets the member list order of hoisted roles.
func (c *Client) ModifyRoleHoistPositions(ctx context.Context, guildID string, positions []RoleHoistPosition) error {
	return c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/roles/hoist-positions", positions, nil)
}

// ResetRoleHoistPositions removes all custom hoist positions in a guild, so hoisted roles
// are listed by their regular position again.
func (c *Client) ResetRoleHoistPositions(ctx context.Context, guildID string) error {
	return c.Do(ctx, http.MethodDelete, guildPath(guildID)+"/roles/hoist-positions", nil, nil)
}
