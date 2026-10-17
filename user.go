package fluxer

import (
	"context"
	"net/http"
)

type User struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	Discriminator string  `json:"discriminator"`
	GlobalName    *string `json:"global_name"`
	Avatar        *string `json:"avatar,omitempty"`
	AvatarColor   *int    `json:"avatar_color,omitempty"`
	Bot           bool    `json:"bot,omitempty"`
	System        bool    `json:"system,omitempty"`
	Flags         int64   `json:"flags,omitempty"`
}

func (u User) DisplayName() string {
	if u.GlobalName != nil && *u.GlobalName != "" {
		return *u.GlobalName
	}
	return u.Username
}

func (u User) Tag() string {
	if u.Discriminator == "" || u.Discriminator == "0" {
		return u.Username
	}
	return u.Username + "#" + u.Discriminator
}

func (u User) Mention() string { return "<@" + u.ID + ">" }

func (u User) AvatarURL(media string) string {
	if u.Avatar == nil {
		return ""
	}
	return AvatarURL(media, u.ID, *u.Avatar)
}

func AvatarURL(media, userID, hash string) string {
	if hash == "" || media == "" {
		return ""
	}
	return media + "/avatars/" + userID + "/" + hash + ".webp"
}

func (c *Client) CurrentUser(ctx context.Context) (*User, error) {
	var u User
	return &u, c.Do(ctx, http.MethodGet, "/users/@me", nil, &u)
}

func (c *Client) User(ctx context.Context, userID string) (*User, error) {
	var u User
	return &u, c.Do(ctx, http.MethodGet, "/users/"+userID, nil, &u)
}
