package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Invite types for Invite.Type.
const (
	InviteGuild   = 0
	InviteGroupDM = 1
)

// Invite is a guild or group DM invite. ExpiresAt is nil for invites that do not expire.
// Uses, MaxUses and MaxAge are only included when the caller may manage the invite.
type Invite struct {
	Code          string     `json:"code"`
	Type          int        `json:"type"`
	Inviter       *User      `json:"inviter,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	Temporary     bool       `json:"temporary"`
	Guild         *Guild     `json:"guild,omitempty"`
	Channel       *Channel   `json:"channel,omitempty"`
	MemberCount   int        `json:"member_count,omitempty"`
	PresenceCount int        `json:"presence_count,omitempty"`
	CreatedAt     time.Time  `json:"created_at,omitzero"`
	Uses          int        `json:"uses,omitempty"`
	MaxUses       int        `json:"max_uses,omitempty"`
	MaxAge        int        `json:"max_age,omitempty"`
}

// URL returns the invite link for the given base, such as the instance's invite host.
func (i Invite) URL(base string) string { return base + "/" + i.Code }

// NewInvite configures CreateInvite. A zero MaxUses is omitted and leaves the limit to the
// server. MaxAge is in seconds; nil uses the server default, while a pointer to 0 is
// sent as is.
type NewInvite struct {
	MaxUses   int  `json:"max_uses,omitempty"`
	MaxAge    *int `json:"max_age,omitempty"`
	Unique    bool `json:"unique,omitempty"`
	Temporary bool `json:"temporary,omitempty"`
}

// Invite resolves an invite code. The code is path-escaped.
func (c *Client) Invite(ctx context.Context, code string) (*Invite, error) {
	var inv Invite
	return &inv, c.Do(ctx, http.MethodGet, "/invites/"+url.PathEscape(code), nil, &inv)
}

// DeleteInvite revokes an invite.
func (c *Client) DeleteInvite(ctx context.Context, code string) error {
	return c.Do(ctx, http.MethodDelete, "/invites/"+url.PathEscape(code), nil, nil)
}

// CreateInvite creates an invite for a channel.
func (c *Client) CreateInvite(ctx context.Context, channelID string, i NewInvite) (*Invite, error) {
	var inv Invite
	return &inv, c.Do(ctx, http.MethodPost, channelPath(channelID)+"/invites", i, &inv)
}

// ChannelInvites lists the invites of a channel.
func (c *Client) ChannelInvites(ctx context.Context, channelID string) ([]Invite, error) {
	var invs []Invite
	return invs, c.Do(ctx, http.MethodGet, channelPath(channelID)+"/invites", nil, &invs)
}

// GuildInvites lists all invites of a guild.
func (c *Client) GuildInvites(ctx context.Context, guildID string) ([]Invite, error) {
	var invs []Invite
	return invs, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/invites", nil, &invs)
}
