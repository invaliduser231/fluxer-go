package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Member is a user's membership in a guild. User may be nil in payloads that omit it,
// and GuildID is only set in some gateway events.
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

// DisplayName returns the member's guild nickname, falling back to the user's display
// name. It returns an empty string when neither is available.
func (m Member) DisplayName() string {
	if m.Nick != nil && *m.Nick != "" {
		return *m.Nick
	}
	if m.User != nil {
		return m.User.DisplayName()
	}
	return ""
}

// TimedOut reports whether the member's timeout is still in effect.
func (m Member) TimedOut() bool {
	return m.CommunicationDisabledUntil != nil && m.CommunicationDisabledUntil.After(time.Now())
}

// MemberAvatarURL builds the WebP URL of a guild-specific member avatar on the media host.
// It returns an empty string when hash or media is empty.
func MemberAvatarURL(media, guildID, userID, hash string) string {
	if hash == "" || media == "" {
		return ""
	}
	return media + "/guilds/" + guildID + "/users/" + userID + "/avatars/" + hash + ".webp"
}

// MemberEdit holds the fields to change on a member. Only non-nil fields are sent.
// CommunicationDisabledUntil cannot be cleared through this struct; use SetTimeout with
// a zero time to lift a timeout. ChannelID moves the member to another voice channel.
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

// MemberSearch is the query for SearchMembers. Zero values are omitted, which leaves
// paging, sorting and filtering to the server defaults. IsBot filters by bot accounts
// when set.
type MemberSearch struct {
	Query     string   `json:"query,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	Offset    int      `json:"offset,omitempty"`
	RoleIDs   []string `json:"role_ids,omitempty"`
	IsBot     *bool    `json:"is_bot,omitempty"`
	SortBy    string   `json:"sort_by,omitempty"`
	SortOrder string   `json:"sort_order,omitempty"`
}

// MemberSearchResult is a single member entry returned by SearchMembers.
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

// MemberSearchPage is one page of SearchMembers results. Indexing is true while the
// server is still building the guild's search index, in which case results may be incomplete.
type MemberSearchPage struct {
	GuildID          string               `json:"guild_id"`
	Members          []MemberSearchResult `json:"members"`
	PageResultCount  int                  `json:"page_result_count"`
	TotalResultCount int                  `json:"total_result_count"`
	Indexing         bool                 `json:"indexing"`
}

const maxMembersPage = 1000

func membersPath(guildID string) string { return guildPath(guildID) + "/members" }

// Member fetches a single guild member.
func (c *Client) Member(ctx context.Context, guildID, userID string) (*Member, error) {
	var m Member
	return &m, c.Do(ctx, http.MethodGet, membersPath(guildID)+"/"+userID, nil, &m)
}

// CurrentMember fetches the client's own member object in a guild.
func (c *Client) CurrentMember(ctx context.Context, guildID string) (*Member, error) {
	return c.Member(ctx, guildID, "@me")
}

// Members lists one page of guild members, starting after the user ID given in after.
// An empty after starts from the beginning, and a limit of zero or less requests the
// maximum page size of 1000.
func (c *Client) Members(ctx context.Context, guildID, after string, limit int) ([]Member, error) {
	if limit <= 0 {
		limit = maxMembersPage
	}
	v := url.Values{"limit": {strconv.Itoa(limit)}}
	if after != "" {
		v.Set("after", after)
	}
	var ms []Member
	return ms, c.Do(ctx, http.MethodGet, membersPath(guildID)+"?"+v.Encode(), nil, &ms)
}

// SearchMembers runs a member search in a guild (POST /guilds/{id}/members-search).
func (c *Client) SearchMembers(ctx context.Context, guildID string, q MemberSearch) (*MemberSearchPage, error) {
	var p MemberSearchPage
	return &p, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/members-search", q, &p)
}

// ModifyMember applies e to a guild member and returns the updated member.
func (c *Client) ModifyMember(ctx context.Context, guildID, userID string, e MemberEdit) (*Member, error) {
	var m Member
	return &m, c.Do(ctx, http.MethodPatch, membersPath(guildID)+"/"+userID, e, &m)
}

// ModifyCurrentMember applies e to the client's own member object in a guild.
func (c *Client) ModifyCurrentMember(ctx context.Context, guildID string, e MemberEdit) (*Member, error) {
	return c.ModifyMember(ctx, guildID, "@me", e)
}

// SetNickname sets a member's guild nickname. The nick is always sent, including an
// empty string.
func (c *Client) SetNickname(ctx context.Context, guildID, userID, nick string) (*Member, error) {
	return c.ModifyMember(ctx, guildID, userID, MemberEdit{Nick: &nick})
}

// SetTimeout times a member out until the given time. A zero time sends null, which
// removes an active timeout.
func (c *Client) SetTimeout(ctx context.Context, guildID, userID string, until time.Time) (*Member, error) {
	body := map[string]any{"communication_disabled_until": nil}
	if !until.IsZero() {
		body["communication_disabled_until"] = until.UTC().Format(time.RFC3339)
	}
	var m Member
	return &m, c.Do(ctx, http.MethodPatch, membersPath(guildID)+"/"+userID, body, &m)
}

// RemoveMember kicks a member from a guild. Use WithReason on ctx to record an audit log reason.
func (c *Client) RemoveMember(ctx context.Context, guildID, userID string) error {
	return c.Do(ctx, http.MethodDelete, membersPath(guildID)+"/"+userID, nil, nil)
}

// AddMemberRole grants a role to a guild member.
func (c *Client) AddMemberRole(ctx context.Context, guildID, userID, roleID string) error {
	return c.Do(ctx, http.MethodPut, membersPath(guildID)+"/"+userID+"/roles/"+roleID, nil, nil)
}

// RemoveMemberRole takes a role away from a guild member.
func (c *Client) RemoveMemberRole(ctx context.Context, guildID, userID, roleID string) error {
	return c.Do(ctx, http.MethodDelete, membersPath(guildID)+"/"+userID+"/roles/"+roleID, nil, nil)
}
