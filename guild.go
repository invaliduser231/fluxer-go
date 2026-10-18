package fluxer

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"slices"
	"strconv"
)

type Guild struct {
	ID                          string    `json:"id"`
	Name                        string    `json:"name"`
	Icon                        string    `json:"icon,omitempty"`
	Banner                      string    `json:"banner,omitempty"`
	Splash                      string    `json:"splash,omitempty"`
	VanityURLCode               string    `json:"vanity_url_code,omitempty"`
	OwnerID                     string    `json:"owner_id"`
	SystemChannelID             string    `json:"system_channel_id,omitempty"`
	SystemChannelFlags          int       `json:"system_channel_flags,omitempty"`
	RulesChannelID              string    `json:"rules_channel_id,omitempty"`
	AFKChannelID                string    `json:"afk_channel_id,omitempty"`
	AFKTimeout                  int       `json:"afk_timeout,omitempty"`
	Features                    []string  `json:"features,omitempty"`
	VerificationLevel           int       `json:"verification_level,omitempty"`
	MFALevel                    int       `json:"mfa_level,omitempty"`
	NSFWLevel                   int       `json:"nsfw_level,omitempty"`
	NSFW                        bool      `json:"nsfw,omitempty"`
	ExplicitContentFilter       int       `json:"explicit_content_filter,omitempty"`
	DefaultMessageNotifications int       `json:"default_message_notifications,omitempty"`
	DisabledOperations          int       `json:"disabled_operations,omitempty"`
	Permissions                 string    `json:"permissions,omitempty"`
	Roles                       []Role    `json:"roles,omitempty"`
	Emojis                      []Emoji   `json:"emojis,omitempty"`
	Stickers                    []Sticker `json:"stickers,omitempty"`
	Channels                    []Channel `json:"channels,omitempty"`
	MemberCount                 int       `json:"member_count,omitempty"`
	OnlineCount                 int       `json:"online_count,omitempty"`
	ApproximateMemberCount      int       `json:"approximate_member_count,omitempty"`
	ApproximatePresenceCount    int       `json:"approximate_presence_count,omitempty"`
}

func (g Guild) HasFeature(feature string) bool { return slices.Contains(g.Features, feature) }

func (g Guild) IconURL(media string) string { return IconURL(media, g.ID, g.Icon) }

func IconURL(media, guildID, hash string) string {
	if hash == "" || media == "" {
		return ""
	}
	return media + "/icons/" + guildID + "/" + hash + ".webp"
}

type Role struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Color         int    `json:"color,omitempty"`
	Position      int    `json:"position"`
	HoistPosition *int   `json:"hoist_position,omitempty"`
	Permissions   string `json:"permissions"`
	Hoist         bool   `json:"hoist,omitempty"`
	Mentionable   bool   `json:"mentionable,omitempty"`
	UnicodeEmoji  string `json:"unicode_emoji,omitempty"`
}

func (r Role) Mention() string { return "<@&" + r.ID + ">" }

type Emoji struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Animated bool   `json:"animated,omitempty"`
	NSFW     bool   `json:"nsfw,omitempty"`
	User     *User  `json:"user,omitempty"`
}

func (e Emoji) String() string {
	if e.Animated {
		return "<a:" + e.Name + ":" + e.ID + ">"
	}
	return "<:" + e.Name + ":" + e.ID + ">"
}

func (e Emoji) APIName() string { return e.Name + ":" + e.ID }

type Sticker struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Animated    bool     `json:"animated,omitempty"`
	NSFW        bool     `json:"nsfw,omitempty"`
	User        *User    `json:"user,omitempty"`
}

func ImageData(contentType string, data []byte) string {
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

type GuildEdit struct {
	Name                        *string `json:"name,omitempty"`
	Icon                        *string `json:"icon,omitempty"`
	Banner                      *string `json:"banner,omitempty"`
	Splash                      *string `json:"splash,omitempty"`
	SystemChannelID             *string `json:"system_channel_id,omitempty"`
	SystemChannelFlags          *int    `json:"system_channel_flags,omitempty"`
	AFKChannelID                *string `json:"afk_channel_id,omitempty"`
	AFKTimeout                  *int    `json:"afk_timeout,omitempty"`
	VerificationLevel           *int    `json:"verification_level,omitempty"`
	DefaultMessageNotifications *int    `json:"default_message_notifications,omitempty"`
	ExplicitContentFilter       *int    `json:"explicit_content_filter,omitempty"`
}

type GuildQuery struct {
	Before     string
	After      string
	Limit      int
	WithCounts bool
}

func guildPath(guildID string) string { return "/guilds/" + guildID }

func (c *Client) Guild(ctx context.Context, guildID string) (*Guild, error) {
	var g Guild
	return &g, c.Do(ctx, http.MethodGet, guildPath(guildID), nil, &g)
}

func (c *Client) ModifyGuild(ctx context.Context, guildID string, e GuildEdit) (*Guild, error) {
	var g Guild
	return &g, c.Do(ctx, http.MethodPatch, guildPath(guildID), e, &g)
}

func (c *Client) Guilds(ctx context.Context, q GuildQuery) ([]Guild, error) {
	v := url.Values{}
	if q.Before != "" {
		v.Set("before", q.Before)
	}
	if q.After != "" {
		v.Set("after", q.After)
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.WithCounts {
		v.Set("with_counts", "true")
	}
	path := "/users/@me/guilds"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var gs []Guild
	return gs, c.Do(ctx, http.MethodGet, path, nil, &gs)
}

func (c *Client) LeaveGuild(ctx context.Context, guildID string) error {
	return c.Do(ctx, http.MethodDelete, "/users/@me/guilds/"+guildID, nil, nil)
}

type VanityURL struct {
	Code string `json:"code,omitempty"`
	Uses int    `json:"uses"`
}

func (c *Client) VanityURL(ctx context.Context, guildID string) (*VanityURL, error) {
	var v VanityURL
	return &v, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/vanity-url", nil, &v)
}

func (c *Client) SetVanityURL(ctx context.Context, guildID, code string) (*VanityURL, error) {
	var v VanityURL
	return &v, c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/vanity-url", map[string]string{"code": code}, &v)
}
