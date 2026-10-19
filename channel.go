package fluxer

import (
	"context"
	"net/http"
	"time"
)

const (
	ChannelText                  = 0
	ChannelDM                    = 1
	ChannelVoice                 = 2
	ChannelGroupDM               = 3
	ChannelCategory              = 4
	ChannelAnnouncement          = 5
	ThreadAnnouncement           = 10
	ThreadPublic                 = 11
	ThreadPrivate                = 12
	ChannelForum                 = 15
	ChannelMedia                 = 16
	ChannelLink                  = 998
	ChannelPersonalNotes         = 999
	ChannelFlagPinned            = 1 << 1
	ChannelFlagRequireTag        = 1 << 4
	ChannelFlagHideMediaDownload = 1 << 15
)

const (
	OverwriteRole   = 0
	OverwriteMember = 1
)

type Channel struct {
	ID                   string      `json:"id"`
	GuildID              string      `json:"guild_id,omitempty"`
	Type                 int         `json:"type"`
	Name                 string      `json:"name,omitempty"`
	Topic                string      `json:"topic,omitempty"`
	URL                  string      `json:"url,omitempty"`
	Icon                 string      `json:"icon,omitempty"`
	OwnerID              string      `json:"owner_id,omitempty"`
	Position             int         `json:"position,omitempty"`
	ParentID             string      `json:"parent_id,omitempty"`
	Bitrate              int         `json:"bitrate,omitempty"`
	UserLimit            int         `json:"user_limit,omitempty"`
	RTCRegion            string      `json:"rtc_region,omitempty"`
	LastMessageID        string      `json:"last_message_id,omitempty"`
	LastPinTimestamp     *time.Time  `json:"last_pin_timestamp,omitempty"`
	PermissionOverwrites []Overwrite `json:"permission_overwrites,omitempty"`
	Recipients           []User      `json:"recipients,omitempty"`
	NSFW                 bool        `json:"nsfw,omitempty"`
	RateLimitPerUser     int         `json:"rate_limit_per_user,omitempty"`
	Flags                int         `json:"flags,omitempty"`

	ThreadMetadata             *ThreadMetadata  `json:"thread_metadata,omitempty"`
	AppliedTags                []string         `json:"applied_tags,omitempty"`
	MessageCount               int              `json:"message_count,omitempty"`
	MemberCount                int              `json:"member_count,omitempty"`
	Member                     *ThreadMember    `json:"member,omitempty"`
	DefaultAutoArchiveDuration int              `json:"default_auto_archive_duration,omitempty"`
	AvailableTags              []ForumTag       `json:"available_tags,omitempty"`
	DefaultReactionEmoji       *DefaultReaction `json:"default_reaction_emoji,omitempty"`
}

func (c Channel) IsThread() bool {
	return c.Type == ThreadAnnouncement || c.Type == ThreadPublic || c.Type == ThreadPrivate
}

func (c Channel) IsPrivate() bool {
	return c.Type == ChannelDM || c.Type == ChannelGroupDM || c.Type == ChannelPersonalNotes
}

func (c Channel) Mention() string { return "<#" + c.ID + ">" }

type Overwrite struct {
	ID    string `json:"id"`
	Type  int    `json:"type"`
	Allow string `json:"allow"`
	Deny  string `json:"deny"`
}

type ThreadMetadata struct {
	Archived            bool      `json:"archived"`
	AutoArchiveDuration int       `json:"auto_archive_duration,omitempty"`
	ArchiveTimestamp    time.Time `json:"archive_timestamp,omitzero"`
	Locked              bool      `json:"locked"`
	Invitable           bool      `json:"invitable,omitempty"`
	CreateTimestamp     time.Time `json:"create_timestamp,omitzero"`
}

type ThreadMember struct {
	ID            string    `json:"id,omitempty"`
	UserID        string    `json:"user_id,omitempty"`
	JoinTimestamp time.Time `json:"join_timestamp,omitzero"`
	Flags         int       `json:"flags"`
	Member        *Member   `json:"member,omitempty"`
}

type ForumTag struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Moderated bool   `json:"moderated"`
	EmojiID   string `json:"emoji_id,omitempty"`
	EmojiName string `json:"emoji_name,omitempty"`
}

type DefaultReaction struct {
	EmojiID   string `json:"emoji_id,omitempty"`
	EmojiName string `json:"emoji_name,omitempty"`
}

type NewChannel struct {
	Name                       string      `json:"name"`
	Type                       int         `json:"type"`
	Topic                      string      `json:"topic,omitempty"`
	URL                        string      `json:"url,omitempty"`
	ParentID                   string      `json:"parent_id,omitempty"`
	Bitrate                    int         `json:"bitrate,omitempty"`
	UserLimit                  int         `json:"user_limit,omitempty"`
	RateLimitPerUser           int         `json:"rate_limit_per_user,omitempty"`
	NSFW                       bool        `json:"nsfw,omitempty"`
	PermissionOverwrites       []Overwrite `json:"permission_overwrites,omitempty"`
	DefaultAutoArchiveDuration int         `json:"default_auto_archive_duration,omitempty"`
}

type ChannelEdit struct {
	Name                       *string          `json:"name,omitempty"`
	Type                       *int             `json:"type,omitempty"`
	Topic                      *string          `json:"topic,omitempty"`
	URL                        *string          `json:"url,omitempty"`
	ParentID                   *string          `json:"parent_id,omitempty"`
	Bitrate                    *int             `json:"bitrate,omitempty"`
	UserLimit                  *int             `json:"user_limit,omitempty"`
	RTCRegion                  *string          `json:"rtc_region,omitempty"`
	RateLimitPerUser           *int             `json:"rate_limit_per_user,omitempty"`
	NSFW                       *bool            `json:"nsfw,omitempty"`
	PermissionOverwrites       *[]Overwrite     `json:"permission_overwrites,omitempty"`
	Flags                      *int             `json:"flags,omitempty"`
	DefaultAutoArchiveDuration *int             `json:"default_auto_archive_duration,omitempty"`
	AvailableTags              *[]ForumTag      `json:"available_tags,omitempty"`
	DefaultReactionEmoji       *DefaultReaction `json:"default_reaction_emoji,omitempty"`
	Archived                   *bool            `json:"archived,omitempty"`
	Locked                     *bool            `json:"locked,omitempty"`
	Invitable                  *bool            `json:"invitable,omitempty"`
	AutoArchiveDuration        *int             `json:"auto_archive_duration,omitempty"`
	AppliedTags                *[]string        `json:"applied_tags,omitempty"`
}

type ChannelPosition struct {
	ID              string  `json:"id"`
	Position        *int    `json:"position,omitempty"`
	ParentID        *string `json:"parent_id,omitempty"`
	LockPermissions bool    `json:"lock_permissions,omitempty"`
}

type FollowedChannel struct {
	ChannelID string `json:"channel_id"`
	WebhookID string `json:"webhook_id"`
}

type SlowmodeState struct {
	RateLimitPerUser  int        `json:"rate_limit_per_user"`
	RetryAfterMS      int        `json:"retry_after_ms"`
	NextSendAllowedAt *time.Time `json:"next_send_allowed_at,omitempty"`
	CanBypass         bool       `json:"can_bypass"`
}

func channelPath(channelID string) string { return "/channels/" + channelID }

func (c *Client) Channel(ctx context.Context, channelID string) (*Channel, error) {
	var ch Channel
	return &ch, c.Do(ctx, http.MethodGet, channelPath(channelID), nil, &ch)
}

func (c *Client) ModifyChannel(ctx context.Context, channelID string, e ChannelEdit) (*Channel, error) {
	var ch Channel
	return &ch, c.Do(ctx, http.MethodPatch, channelPath(channelID), e, &ch)
}

func (c *Client) DeleteChannel(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodDelete, channelPath(channelID), nil, nil)
}

func (c *Client) GuildChannels(ctx context.Context, guildID string) ([]Channel, error) {
	var chs []Channel
	return chs, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/channels", nil, &chs)
}

func (c *Client) CreateChannel(ctx context.Context, guildID string, ch NewChannel) (*Channel, error) {
	var out Channel
	return &out, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/channels", ch, &out)
}

func (c *Client) ModifyChannelPositions(ctx context.Context, guildID string, positions []ChannelPosition) error {
	return c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/channels", positions, nil)
}

func (c *Client) SetPermission(ctx context.Context, channelID string, o Overwrite) error {
	body := map[string]any{"type": o.Type, "allow": o.Allow, "deny": o.Deny}
	return c.Do(ctx, http.MethodPut, channelPath(channelID)+"/permissions/"+o.ID, body, nil)
}

func (c *Client) DeletePermission(ctx context.Context, channelID, overwriteID string) error {
	return c.Do(ctx, http.MethodDelete, channelPath(channelID)+"/permissions/"+overwriteID, nil, nil)
}

func (c *Client) CreateDM(ctx context.Context, userID string) (*Channel, error) {
	var ch Channel
	return &ch, c.Do(ctx, http.MethodPost, "/users/@me/channels", map[string]string{"recipient_id": userID}, &ch)
}

func (c *Client) PrivateChannels(ctx context.Context) ([]Channel, error) {
	var chs []Channel
	return chs, c.Do(ctx, http.MethodGet, "/users/@me/channels", nil, &chs)
}

func (c *Client) FollowChannel(ctx context.Context, channelID, targetChannelID string) (*FollowedChannel, error) {
	var f FollowedChannel
	return &f, c.Do(ctx, http.MethodPost, channelPath(channelID)+"/followers", map[string]string{"webhook_channel_id": targetChannelID}, &f)
}

func (c *Client) Slowmode(ctx context.Context, channelID string) (*SlowmodeState, error) {
	var st SlowmodeState
	return &st, c.Do(ctx, http.MethodGet, channelPath(channelID)+"/slowmode", nil, &st)
}
