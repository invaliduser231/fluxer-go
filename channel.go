package fluxer

import "time"

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
