package fluxer

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"
)

// Guild is a Fluxer guild (server). Roles, Emojis, Stickers and Channels are only filled
// in by endpoints and events that include them, and the count fields depend on the
// request, for example GuildQuery.WithCounts. MessageHistoryCutoff limits how far back
// members without Read Message History can read. ThreadsActive is true when threads,
// forums and media channels are enabled, Visiting when the current user visits the guild
// without being a member.
type Guild struct {
	ID                          string      `json:"id"`
	Name                        string      `json:"name"`
	Icon                        string      `json:"icon,omitempty"`
	Banner                      string      `json:"banner,omitempty"`
	BannerWidth                 int         `json:"banner_width,omitempty"`
	BannerHeight                int         `json:"banner_height,omitempty"`
	Splash                      string      `json:"splash,omitempty"`
	SplashWidth                 int         `json:"splash_width,omitempty"`
	SplashHeight                int         `json:"splash_height,omitempty"`
	SplashCardAlignment         int         `json:"splash_card_alignment,omitempty"`
	EmbedSplash                 string      `json:"embed_splash,omitempty"`
	EmbedSplashWidth            int         `json:"embed_splash_width,omitempty"`
	EmbedSplashHeight           int         `json:"embed_splash_height,omitempty"`
	VanityURLCode               string      `json:"vanity_url_code,omitempty"`
	OwnerID                     string      `json:"owner_id"`
	SystemChannelID             string      `json:"system_channel_id,omitempty"`
	SystemChannelFlags          int         `json:"system_channel_flags,omitempty"`
	RulesChannelID              string      `json:"rules_channel_id,omitempty"`
	AFKChannelID                string      `json:"afk_channel_id,omitempty"`
	AFKTimeout                  int         `json:"afk_timeout,omitempty"`
	Features                    []string    `json:"features,omitempty"`
	VerificationLevel           int         `json:"verification_level,omitempty"`
	MFALevel                    int         `json:"mfa_level,omitempty"`
	NSFWLevel                   int         `json:"nsfw_level,omitempty"`
	NSFW                        bool        `json:"nsfw,omitempty"`
	ContentWarningLevel         int         `json:"content_warning_level,omitempty"`
	ContentWarningText          string      `json:"content_warning_text,omitempty"`
	ExplicitContentFilter       int         `json:"explicit_content_filter,omitempty"`
	DefaultMessageNotifications int         `json:"default_message_notifications,omitempty"`
	DisabledOperations          int         `json:"disabled_operations,omitempty"`
	MessageHistoryCutoff        *time.Time  `json:"message_history_cutoff,omitempty"`
	Permissions                 Permissions `json:"permissions,omitempty"`
	Roles                       []Role      `json:"roles,omitempty"`
	Emojis                      []Emoji     `json:"emojis,omitempty"`
	Stickers                    []Sticker   `json:"stickers,omitempty"`
	Channels                    []Channel   `json:"channels,omitempty"`
	MemberCount                 int         `json:"member_count,omitempty"`
	OnlineCount                 int         `json:"online_count,omitempty"`
	ApproximateMemberCount      int         `json:"approximate_member_count,omitempty"`
	ApproximatePresenceCount    int         `json:"approximate_presence_count,omitempty"`
	ThreadsActive               bool        `json:"threads_active,omitempty"`
	Visiting                    bool        `json:"visiting,omitempty"`
}

// HasFeature reports whether the guild has the named feature flag.
func (g Guild) HasFeature(feature string) bool { return slices.Contains(g.Features, feature) }

// IconURL returns the URL of the guild icon on the media host, or an empty string when
// the guild has no icon.
func (g Guild) IconURL(media string) string { return IconURL(media, g.ID, g.Icon) }

// IconURL builds the WebP guild icon URL for guildID and hash on the media host. It
// returns an empty string when hash or media is empty.
func IconURL(media, guildID, hash string) string {
	if hash == "" || media == "" {
		return ""
	}
	return media + "/icons/" + guildID + "/" + hash + ".webp"
}

// Role is a guild role. HoistPosition is nil unless the role has its own position in the
// member list.
type Role struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Color         int         `json:"color,omitempty"`
	Position      int         `json:"position"`
	HoistPosition *int        `json:"hoist_position,omitempty"`
	Permissions   Permissions `json:"permissions"`
	Hoist         bool        `json:"hoist,omitempty"`
	Mentionable   bool        `json:"mentionable,omitempty"`
	UnicodeEmoji  string      `json:"unicode_emoji,omitempty"`
}

// Mention returns the <@&id> mention string for the role.
func (r Role) Mention() string { return "<@&" + r.ID + ">" }

// Emoji is a custom guild emoji. User is the uploader and is nil when the API leaves it
// out.
type Emoji struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Animated bool   `json:"animated,omitempty"`
	NSFW     bool   `json:"nsfw,omitempty"`
	User     *User  `json:"user,omitempty"`
}

// String returns the emoji in message content form, <:name:id> or <a:name:id> when
// animated.
func (e Emoji) String() string {
	if e.Animated {
		return "<a:" + e.Name + ":" + e.ID + ">"
	}
	return "<:" + e.Name + ":" + e.ID + ">"
}

// APIName returns name:id, the form the reaction endpoints expect.
func (e Emoji) APIName() string { return e.Name + ":" + e.ID }

// Sticker is a custom guild sticker.
type Sticker struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Animated    bool     `json:"animated,omitempty"`
	NSFW        bool     `json:"nsfw,omitempty"`
	User        *User    `json:"user,omitempty"`
}

// ImageData encodes data as a base64 data URI, the format the API expects for icons,
// banners, avatars, emoji and sticker uploads.
func ImageData(contentType string, data []byte) string {
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// GuildEdit holds the fields to change on a guild. Only non-nil fields are sent. Image
// fields take a data URI from ImageData. Features is the complete feature list the guild
// should have; Fluxer only changes the features owners may toggle.
type GuildEdit struct {
	Name                        *string    `json:"name,omitempty"`
	Icon                        *string    `json:"icon,omitempty"`
	Banner                      *string    `json:"banner,omitempty"`
	Splash                      *string    `json:"splash,omitempty"`
	EmbedSplash                 *string    `json:"embed_splash,omitempty"`
	SplashCardAlignment         *int       `json:"splash_card_alignment,omitempty"`
	SystemChannelID             *string    `json:"system_channel_id,omitempty"`
	SystemChannelFlags          *int       `json:"system_channel_flags,omitempty"`
	AFKChannelID                *string    `json:"afk_channel_id,omitempty"`
	AFKTimeout                  *int       `json:"afk_timeout,omitempty"`
	VerificationLevel           *int       `json:"verification_level,omitempty"`
	MFALevel                    *int       `json:"mfa_level,omitempty"`
	NSFW                        *bool      `json:"nsfw,omitempty"`
	ContentWarningLevel         *int       `json:"content_warning_level,omitempty"`
	ContentWarningText          *string    `json:"content_warning_text,omitempty"`
	DefaultMessageNotifications *int       `json:"default_message_notifications,omitempty"`
	ExplicitContentFilter       *int       `json:"explicit_content_filter,omitempty"`
	Features                    *[]string  `json:"features,omitempty"`
	MessageHistoryCutoff        *time.Time `json:"message_history_cutoff,omitempty"`
}

// GuildQuery pages through the current user's guilds in Guilds. Before and After are
// guild IDs, a zero Limit leaves the page size to the server, and WithCounts asks for
// approximate member and presence counts.
type GuildQuery struct {
	Before     string
	After      string
	Limit      int
	WithCounts bool
}

func guildPath(guildID string) string { return "/guilds/" + guildID }

// Guild fetches a guild by ID.
func (c *Client) Guild(ctx context.Context, guildID string) (*Guild, error) {
	var g Guild
	return &g, c.Do(ctx, http.MethodGet, guildPath(guildID), nil, &g)
}

// ModifyGuild applies e to a guild and returns the updated guild.
func (c *Client) ModifyGuild(ctx context.Context, guildID string, e GuildEdit) (*Guild, error) {
	var g Guild
	return &g, c.Do(ctx, http.MethodPatch, guildPath(guildID), e, &g)
}

// Guilds lists the guilds the current user is in (GET /users/@me/guilds).
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

// LeaveGuild makes the current user leave a guild.
func (c *Client) LeaveGuild(ctx context.Context, guildID string) error {
	return c.Do(ctx, http.MethodDelete, "/users/@me/guilds/"+guildID, nil, nil)
}

// VanityURL is a guild's custom invite code and how often it has been used. Code is empty
// when none is set.
type VanityURL struct {
	Code string `json:"code,omitempty"`
	Uses int    `json:"uses"`
}

// VanityURL fetches the vanity invite of a guild.
func (c *Client) VanityURL(ctx context.Context, guildID string) (*VanityURL, error) {
	var v VanityURL
	return &v, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/vanity-url", nil, &v)
}

// SetVanityURL changes the vanity invite code of a guild.
func (c *Client) SetVanityURL(ctx context.Context, guildID, code string) (*VanityURL, error) {
	var v VanityURL
	return &v, c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/vanity-url", map[string]string{"code": code}, &v)
}
