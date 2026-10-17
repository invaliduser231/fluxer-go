package fluxer

import "time"

const (
	MessageDefault              = 0
	MessageRecipientAdd         = 1
	MessageRecipientRemove      = 2
	MessageCall                 = 3
	MessageChannelNameChange    = 4
	MessageChannelIconChange    = 5
	MessageChannelPinnedMessage = 6
	MessageUserJoin             = 7
	MessageChannelFollowAdd     = 12
	MessageThreadCreated        = 18
	MessageReply                = 19
	MessageThreadStarter        = 21
)

const (
	FlagCrossposted           = 1 << 0
	FlagIsCrosspost           = 1 << 1
	FlagSuppressEmbeds        = 1 << 2
	FlagSourceMessageDeleted  = 1 << 3
	FlagHasThread             = 1 << 5
	FlagSuppressNotifications = 1 << 12
	FlagVoiceMessage          = 1 << 13
)

type Message struct {
	ID                string            `json:"id"`
	ChannelID         string            `json:"channel_id"`
	GuildID           string            `json:"guild_id,omitempty"`
	Author            User              `json:"author"`
	Member            *Member           `json:"member,omitempty"`
	WebhookID         string            `json:"webhook_id,omitempty"`
	Type              int               `json:"type,omitempty"`
	Flags             int               `json:"flags,omitempty"`
	Content           string            `json:"content"`
	Timestamp         time.Time         `json:"timestamp,omitzero"`
	EditedTimestamp   *time.Time        `json:"edited_timestamp,omitempty"`
	Pinned            bool              `json:"pinned,omitempty"`
	MentionEveryone   bool              `json:"mention_everyone,omitempty"`
	TTS               bool              `json:"tts,omitempty"`
	Mentions          []User            `json:"mentions,omitempty"`
	MentionRoles      []string          `json:"mention_roles,omitempty"`
	MentionChannels   []ChannelMention  `json:"mention_channels,omitempty"`
	Embeds            []Embed           `json:"embeds,omitempty"`
	Attachments       []Attachment      `json:"attachments,omitempty"`
	Stickers          []MessageSticker  `json:"stickers,omitempty"`
	Reactions         []MessageReaction `json:"reactions,omitempty"`
	MessageReference  *MessageReference `json:"message_reference,omitempty"`
	ReferencedMessage *Message          `json:"referenced_message,omitempty"`
	Nonce             string            `json:"nonce,omitempty"`
	Thread            *Channel          `json:"thread,omitempty"`
}

func (m Message) URL(webapp string) string {
	guild := m.GuildID
	if guild == "" {
		guild = "@me"
	}
	return webapp + "/channels/" + guild + "/" + m.ChannelID + "/" + m.ID
}

type Attachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size"`
	URL         string `json:"url,omitempty"`
	ProxyURL    string `json:"proxy_url,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Flags       int    `json:"flags,omitempty"`
	Duration    int    `json:"duration,omitempty"`
}

type ChannelMention struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

type MessageSticker struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Animated bool   `json:"animated"`
}

type ReactionEmoji struct {
	Name     string `json:"name"`
	ID       string `json:"id,omitempty"`
	Animated *bool  `json:"animated,omitempty"`
}

func (e ReactionEmoji) String() string {
	if e.ID == "" {
		return e.Name
	}
	return "<:" + e.Name + ":" + e.ID + ">"
}

func (e ReactionEmoji) APIName() string {
	if e.ID == "" {
		return e.Name
	}
	return e.Name + ":" + e.ID
}

type MessageReaction struct {
	Emoji ReactionEmoji `json:"emoji"`
	Count int           `json:"count"`
	Me    bool          `json:"me,omitempty"`
}

const (
	ReferenceDefault = 0
	ReferenceForward = 1
)

type MessageReference struct {
	MessageID string `json:"message_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
	GuildID   string `json:"guild_id,omitempty"`
	Type      int    `json:"type,omitempty"`
}

type AllowedMentions struct {
	Parse       []string `json:"parse"`
	Users       []string `json:"users,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	RepliedUser bool     `json:"replied_user"`
}

type MessageSend struct {
	Content          string            `json:"content,omitempty"`
	Embeds           []Embed           `json:"embeds,omitempty"`
	MessageReference *MessageReference `json:"message_reference,omitempty"`
	AllowedMentions  *AllowedMentions  `json:"allowed_mentions,omitempty"`
	StickerIDs       []string          `json:"sticker_ids,omitempty"`
	Flags            int               `json:"flags,omitempty"`
	TTS              bool              `json:"tts,omitempty"`
	Nonce            string            `json:"nonce,omitempty"`
}

func (m MessageSend) nonce() string { return m.Nonce }

type MessageEdit struct {
	Content         *string          `json:"content,omitempty"`
	Embeds          *[]Embed         `json:"embeds,omitempty"`
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
	Flags           *int             `json:"flags,omitempty"`
}
