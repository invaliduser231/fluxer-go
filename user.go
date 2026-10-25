package fluxer

import (
	"context"
	"net/http"
)

// Public user flags for User.Flags.
const (
	UserStaff                     = 1 << 0
	UserPartner                   = 1 << 2
	UserBugHunter                 = 1 << 3
	UserFriendlyBot               = 1 << 4
	UserFriendlyBotManualApproval = 1 << 5
	UserSpammer                   = 1 << 6
)

// User is a Fluxer user account. GlobalName and Avatar are nil when the user has not set them.
// MentionFlags is the user's reply mention preference, one of the Mention constants.
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
	MentionFlags  int     `json:"mention_flags,omitempty"`
}

// DisplayName returns the user's global name, or the username when no global name is set.
func (u User) DisplayName() string {
	if u.GlobalName != nil && *u.GlobalName != "" {
		return *u.GlobalName
	}
	return u.Username
}

// Tag returns username#discriminator, or the plain username when the discriminator is empty or "0".
func (u User) Tag() string {
	if u.Discriminator == "" || u.Discriminator == "0" {
		return u.Username
	}
	return u.Username + "#" + u.Discriminator
}

// Mention returns the <@id> mention string for the user.
func (u User) Mention() string { return "<@" + u.ID + ">" }

// AvatarURL returns the URL of the user's avatar on the given media host, or an empty
// string when the user has no avatar.
func (u User) AvatarURL(media string) string {
	if u.Avatar == nil {
		return ""
	}
	return AvatarURL(media, u.ID, *u.Avatar)
}

// AvatarURL builds the WebP avatar URL for userID and hash on the media host, such as
// DefaultMedia. It returns an empty string when hash or media is empty.
func AvatarURL(media, userID, hash string) string {
	if hash == "" || media == "" {
		return ""
	}
	return media + "/avatars/" + userID + "/" + hash + ".webp"
}

// CurrentUser fetches the user the client is authenticated as (GET /users/@me).
func (c *Client) CurrentUser(ctx context.Context) (*User, error) {
	var u User
	return &u, c.Do(ctx, http.MethodGet, "/users/@me", nil, &u)
}

// User fetches a user by ID.
func (c *Client) User(ctx context.Context, userID string) (*User, error) {
	var u User
	return &u, c.Do(ctx, http.MethodGet, "/users/"+userID, nil, &u)
}
