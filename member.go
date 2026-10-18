package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Member struct {
	User                       *User      `json:"user,omitempty"`
	Nick                       *string    `json:"nick,omitempty"`
	Avatar                     *string    `json:"avatar,omitempty"`
	Banner                     *string    `json:"banner,omitempty"`
	AccentColor                *int       `json:"accent_color,omitempty"`
	Roles                      []string   `json:"roles"`
	JoinedAt                   time.Time  `json:"joined_at,omitzero"`
	Mute                       bool       `json:"mute,omitempty"`
	Deaf                       bool       `json:"deaf,omitempty"`
	CommunicationDisabledUntil *time.Time `json:"communication_disabled_until,omitempty"`
	GuildID                    string     `json:"guild_id,omitempty"`
}

func (m Member) DisplayName() string {
	if m.Nick != nil && *m.Nick != "" {
		return *m.Nick
	}
	if m.User != nil {
		return m.User.DisplayName()
	}
	return ""
}

func (m Member) TimedOut() bool {
	return m.CommunicationDisabledUntil != nil && m.CommunicationDisabledUntil.After(time.Now())
}

func MemberAvatarURL(media, guildID, userID, hash string) string {
	if hash == "" || media == "" {
		return ""
	}
	return media + "/guilds/" + guildID + "/users/" + userID + "/avatars/" + hash + ".webp"
}

type MemberEdit struct {
	Nick                       *string   `json:"nick,omitempty"`
	Roles                      *[]string `json:"roles,omitempty"`
	Mute                       *bool     `json:"mute,omitempty"`
	Deaf                       *bool     `json:"deaf,omitempty"`
	CommunicationDisabledUntil *string   `json:"communication_disabled_until,omitempty"`
	TimeoutReason              *string   `json:"timeout_reason,omitempty"`
	ChannelID                  *string   `json:"channel_id,omitempty"`
	ConnectionID               *string   `json:"connection_id,omitempty"`
}

type MemberSearch struct {
	Query     string   `json:"query,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	Offset    int      `json:"offset,omitempty"`
	RoleIDs   []string `json:"role_ids,omitempty"`
	IsBot     *bool    `json:"is_bot,omitempty"`
	SortBy    string   `json:"sort_by,omitempty"`
	SortOrder string   `json:"sort_order,omitempty"`
}

type MemberSearchResult struct {
	ID            string   `json:"id"`
	GuildID       string   `json:"guild_id"`
	UserID        string   `json:"user_id"`
	Username      string   `json:"username"`
	Discriminator string   `json:"discriminator"`
	GlobalName    *string  `json:"global_name"`
	Nickname      *string  `json:"nickname"`
	RoleIDs       []string `json:"role_ids"`
	JoinedAt      int64    `json:"joined_at"`
	IsBot         bool     `json:"is_bot"`
}

type MemberSearchPage struct {
	GuildID          string               `json:"guild_id"`
	Members          []MemberSearchResult `json:"members"`
	PageResultCount  int                  `json:"page_result_count"`
	TotalResultCount int                  `json:"total_result_count"`
	Indexing         bool                 `json:"indexing"`
}

func membersPath(guildID string) string { return guildPath(guildID) + "/members" }

func (c *Client) Member(ctx context.Context, guildID, userID string) (*Member, error) {
	var m Member
	return &m, c.Do(ctx, http.MethodGet, membersPath(guildID)+"/"+userID, nil, &m)
}

func (c *Client) CurrentMember(ctx context.Context, guildID string) (*Member, error) {
	return c.Member(ctx, guildID, "@me")
}

func (c *Client) Members(ctx context.Context, guildID, after string, limit int) ([]Member, error) {
	v := url.Values{}
	if after != "" {
		v.Set("after", after)
	}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	path := membersPath(guildID)
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var ms []Member
	return ms, c.Do(ctx, http.MethodGet, path, nil, &ms)
}

func (c *Client) SearchMembers(ctx context.Context, guildID string, q MemberSearch) (*MemberSearchPage, error) {
	var p MemberSearchPage
	return &p, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/members-search", q, &p)
}

func (c *Client) ModifyMember(ctx context.Context, guildID, userID string, e MemberEdit) (*Member, error) {
	var m Member
	return &m, c.Do(ctx, http.MethodPatch, membersPath(guildID)+"/"+userID, e, &m)
}

func (c *Client) ModifyCurrentMember(ctx context.Context, guildID string, e MemberEdit) (*Member, error) {
	return c.ModifyMember(ctx, guildID, "@me", e)
}

func (c *Client) SetNickname(ctx context.Context, guildID, userID, nick string) (*Member, error) {
	return c.ModifyMember(ctx, guildID, userID, MemberEdit{Nick: &nick})
}

func (c *Client) SetTimeout(ctx context.Context, guildID, userID string, until time.Time) (*Member, error) {
	body := map[string]any{"communication_disabled_until": nil}
	if !until.IsZero() {
		body["communication_disabled_until"] = until.UTC().Format(time.RFC3339)
	}
	var m Member
	return &m, c.Do(ctx, http.MethodPatch, membersPath(guildID)+"/"+userID, body, &m)
}

func (c *Client) RemoveMember(ctx context.Context, guildID, userID string) error {
	return c.Do(ctx, http.MethodDelete, membersPath(guildID)+"/"+userID, nil, nil)
}

func (c *Client) AddMemberRole(ctx context.Context, guildID, userID, roleID string) error {
	return c.Do(ctx, http.MethodPut, membersPath(guildID)+"/"+userID+"/roles/"+roleID, nil, nil)
}

func (c *Client) RemoveMemberRole(ctx context.Context, guildID, userID, roleID string) error {
	return c.Do(ctx, http.MethodDelete, membersPath(guildID)+"/"+userID+"/roles/"+roleID, nil, nil)
}
