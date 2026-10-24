package fluxer

import (
	"context"
	"net/http"
)

// NewRole is the body for CreateRole. Zero Color and Permissions are omitted, so the
// server applies its defaults for them.
type NewRole struct {
	Name        string      `json:"name"`
	Color       int         `json:"color,omitempty"`
	Permissions Permissions `json:"permissions,omitempty"`
}

// RoleEdit holds the fields to change on a role. Only non-nil fields are sent.
type RoleEdit struct {
	Name          *string      `json:"name,omitempty"`
	Color         *int         `json:"color,omitempty"`
	Permissions   *Permissions `json:"permissions,omitempty"`
	Hoist         *bool        `json:"hoist,omitempty"`
	HoistPosition *int         `json:"hoist_position,omitempty"`
	Mentionable   *bool        `json:"mentionable,omitempty"`
}

// RolePosition sets the position of one role in ModifyRolePositions.
type RolePosition struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

// Roles lists the roles of a guild.
func (c *Client) Roles(ctx context.Context, guildID string) ([]Role, error) {
	var roles []Role
	return roles, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/roles", nil, &roles)
}

// CreateRole creates a role in a guild.
func (c *Client) CreateRole(ctx context.Context, guildID string, r NewRole) (*Role, error) {
	var role Role
	return &role, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/roles", r, &role)
}

// ModifyRole applies e to a role and returns the updated role.
func (c *Client) ModifyRole(ctx context.Context, guildID, roleID string, e RoleEdit) (*Role, error) {
	var role Role
	return &role, c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/roles/"+roleID, e, &role)
}

// DeleteRole deletes a role.
func (c *Client) DeleteRole(ctx context.Context, guildID, roleID string) error {
	return c.Do(ctx, http.MethodDelete, guildPath(guildID)+"/roles/"+roleID, nil, nil)
}

// ModifyRolePositions reorders roles in a guild and returns the guild's roles after the
// change.
func (c *Client) ModifyRolePositions(ctx context.Context, guildID string, positions []RolePosition) ([]Role, error) {
	var roles []Role
	return roles, c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/roles", positions, &roles)
}
