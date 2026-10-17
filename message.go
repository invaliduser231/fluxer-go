package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

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

type MessageQuery struct {
	Before string
	After  string
	Around string
	Limit  int
}

func (q MessageQuery) values() url.Values {
	v := url.Values{}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	for key, id := range map[string]string{"before": q.Before, "after": q.After, "around": q.Around} {
		if id != "" {
			v.Set(key, id)
		}
	}
	return v
}

type PinnedMessage struct {
	Message  Message   `json:"message"`
	PinnedAt time.Time `json:"pinned_at"`
}

type ChannelPins struct {
	Items   []PinnedMessage `json:"items"`
	HasMore bool            `json:"has_more"`
}

func messagesPath(channelID string) string { return "/channels/" + channelID + "/messages" }

func (c *Client) SendMessage(ctx context.Context, channelID string, m MessageSend) (*Message, error) {
	return c.SendMessageFiles(ctx, channelID, m, nil)
}

func (c *Client) SendMessageFiles(ctx context.Context, channelID string, m MessageSend, files []File) (*Message, error) {
	if m.Nonce == "" {
		m.Nonce = newNonce()
	}
	var msg Message
	return &msg, c.DoFiles(ctx, http.MethodPost, messagesPath(channelID), m, files, &msg)
}

func (c *Client) SendText(ctx context.Context, channelID, content string) (*Message, error) {
	return c.SendMessage(ctx, channelID, MessageSend{Content: content})
}

func (c *Client) Reply(ctx context.Context, to *Message, m MessageSend) (*Message, error) {
	m.MessageReference = &MessageReference{MessageID: to.ID, ChannelID: to.ChannelID, GuildID: to.GuildID}
	return c.SendMessage(ctx, to.ChannelID, m)
}

func (c *Client) EditMessage(ctx context.Context, channelID, messageID string, m MessageEdit) (*Message, error) {
	var msg Message
	return &msg, c.Do(ctx, http.MethodPatch, messagesPath(channelID)+"/"+messageID, m, &msg)
}

func (c *Client) DeleteMessage(ctx context.Context, channelID, messageID string) error {
	return c.Do(ctx, http.MethodDelete, messagesPath(channelID)+"/"+messageID, nil, nil)
}

func (c *Client) Message(ctx context.Context, channelID, messageID string) (*Message, error) {
	var msg Message
	return &msg, c.Do(ctx, http.MethodGet, messagesPath(channelID)+"/"+messageID, nil, &msg)
}

func (c *Client) Messages(ctx context.Context, channelID string, q MessageQuery) ([]Message, error) {
	path := messagesPath(channelID)
	if v := q.values(); len(v) > 0 {
		path += "?" + v.Encode()
	}
	var msgs []Message
	return msgs, c.Do(ctx, http.MethodGet, path, nil, &msgs)
}

func (c *Client) BulkDelete(ctx context.Context, channelID string, ids []string) error {
	return c.Do(ctx, http.MethodPost, messagesPath(channelID)+"/bulk-delete", map[string][]string{"message_ids": ids}, nil)
}

func (c *Client) DeleteAttachment(ctx context.Context, channelID, messageID, attachmentID string) error {
	return c.Do(ctx, http.MethodDelete, messagesPath(channelID)+"/"+messageID+"/attachments/"+attachmentID, nil, nil)
}

func (c *Client) Crosspost(ctx context.Context, channelID, messageID string) (*Message, error) {
	var msg Message
	return &msg, c.Do(ctx, http.MethodPost, messagesPath(channelID)+"/"+messageID+"/crosspost", nil, &msg)
}

func (c *Client) Typing(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodPost, "/channels/"+channelID+"/typing", nil, nil)
}

func (c *Client) Pin(ctx context.Context, channelID, messageID string) error {
	return c.Do(ctx, http.MethodPut, "/channels/"+channelID+"/pins/"+messageID, nil, nil)
}

func (c *Client) Unpin(ctx context.Context, channelID, messageID string) error {
	return c.Do(ctx, http.MethodDelete, "/channels/"+channelID+"/pins/"+messageID, nil, nil)
}

func (c *Client) Pins(ctx context.Context, channelID string, before time.Time, limit int) (*ChannelPins, error) {
	v := url.Values{}
	if !before.IsZero() {
		v.Set("before", before.UTC().Format(time.RFC3339Nano))
	}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	path := messagesPath(channelID) + "/pins"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var p ChannelPins
	return &p, c.Do(ctx, http.MethodGet, path, nil, &p)
}
