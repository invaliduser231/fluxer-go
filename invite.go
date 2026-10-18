package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

const (
	InviteGuild   = 0
	InviteGroupDM = 1
)

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

func (i Invite) URL(base string) string { return base + "/" + i.Code }

type InviteCreate struct {
	MaxUses   int  `json:"max_uses,omitempty"`
	MaxAge    *int `json:"max_age,omitempty"`
	Unique    bool `json:"unique,omitempty"`
	Temporary bool `json:"temporary,omitempty"`
}

func (c *Client) Invite(ctx context.Context, code string) (*Invite, error) {
	var inv Invite
	return &inv, c.Do(ctx, http.MethodGet, "/invites/"+url.PathEscape(code), nil, &inv)
}

func (c *Client) DeleteInvite(ctx context.Context, code string) error {
	return c.Do(ctx, http.MethodDelete, "/invites/"+url.PathEscape(code), nil, nil)
}

func (c *Client) CreateInvite(ctx context.Context, channelID string, i InviteCreate) (*Invite, error) {
	var inv Invite
	return &inv, c.Do(ctx, http.MethodPost, channelPath(channelID)+"/invites", i, &inv)
}

func (c *Client) ChannelInvites(ctx context.Context, channelID string) ([]Invite, error) {
	var invs []Invite
	return invs, c.Do(ctx, http.MethodGet, channelPath(channelID)+"/invites", nil, &invs)
}

func (c *Client) GuildInvites(ctx context.Context, guildID string) ([]Invite, error) {
	var invs []Invite
	return invs, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/invites", nil, &invs)
}
