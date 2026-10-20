package fluxer

import (
	"context"
	"net/http"
)

type NewRole struct {
	Name        string      `json:"name"`
	Color       int         `json:"color,omitempty"`
	Permissions Permissions `json:"permissions,omitempty"`
}

type RoleEdit struct {
	Name          *string      `json:"name,omitempty"`
	Color         *int         `json:"color,omitempty"`
	Permissions   *Permissions `json:"permissions,omitempty"`
	Hoist         *bool        `json:"hoist,omitempty"`
	HoistPosition *int         `json:"hoist_position,omitempty"`
	Mentionable   *bool        `json:"mentionable,omitempty"`
}

type RolePosition struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

func (c *Client) Roles(ctx context.Context, guildID string) ([]Role, error) {
	var roles []Role
	return roles, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/roles", nil, &roles)
}

func (c *Client) CreateRole(ctx context.Context, guildID string, r NewRole) (*Role, error) {
	var role Role
	return &role, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/roles", r, &role)
}

func (c *Client) ModifyRole(ctx context.Context, guildID, roleID string, e RoleEdit) (*Role, error) {
	var role Role
	return &role, c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/roles/"+roleID, e, &role)
}

func (c *Client) DeleteRole(ctx context.Context, guildID, roleID string) error {
	return c.Do(ctx, http.MethodDelete, guildPath(guildID)+"/roles/"+roleID, nil, nil)
}

func (c *Client) ModifyRolePositions(ctx context.Context, guildID string, positions []RolePosition) ([]Role, error) {
	var roles []Role
	return roles, c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/roles", positions, &roles)
}
