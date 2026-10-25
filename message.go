package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Message types for Message.Type.
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

// Message flags, used as bits in Message.Flags and MessageSend.Flags.
const (
	FlagCrossposted           = 1 << 0
	FlagIsCrosspost           = 1 << 1
	FlagSuppressEmbeds        = 1 << 2
	FlagSourceMessageDeleted  = 1 << 3
	FlagHasThread             = 1 << 5
	FlagSuppressNotifications = 1 << 12
	FlagVoiceMessage          = 1 << 13
)

// Message is a message in a channel. GuildID is empty for messages in DMs, and Member is
// only set for guild messages that carry the author's member data.
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
	MessageSnapshots  []MessageSnapshot `json:"message_snapshots,omitempty"`
	Users             []User            `json:"users,omitempty"`
	Call              *CallInfo         `json:"call,omitempty"`
	Nonce             string            `json:"nonce,omitempty"`
	Thread            *Channel          `json:"thread,omitempty"`
}

// MessageSnapshot is the copy of a forwarded message that Fluxer stores with
// the forward. Mentions holds user IDs. The author and the origin of the
// message are not part of it; MessageReference of the forward points at the
// original.
type MessageSnapshot struct {
	Content         string           `json:"content"`
	Timestamp       time.Time        `json:"timestamp"`
	EditedTimestamp *time.Time       `json:"edited_timestamp,omitempty"`
	Mentions        []string         `json:"mentions,omitempty"`
	MentionRoles    []string         `json:"mention_roles,omitempty"`
	MentionChannels []ChannelMention `json:"mention_channels,omitempty"`
	Embeds          []Embed          `json:"embeds,omitempty"`
	Attachments     []Attachment     `json:"attachments,omitempty"`
	Stickers        []MessageSticker `json:"stickers,omitempty"`
	Type            int              `json:"type"`
	Flags           int              `json:"flags"`
}

// CallInfo describes the call of a call message.
type CallInfo struct {
	Participants   []string   `json:"participants"`
	EndedTimestamp *time.Time `json:"ended_timestamp,omitempty"`
}

// URL returns a link to the message in the web app at webapp, using @me in place of the
// guild ID for DM messages.
func (m Message) URL(webapp string) string {
	guild := m.GuildID
	if guild == "" {
		guild = "@me"
	}
	return webapp + "/channels/" + guild + "/" + m.ChannelID + "/" + m.ID
}

// Attachment is a file attached to a message. Size is in bytes.
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

// ChannelMention is a channel referenced in a message's content.
type ChannelMention struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

// MessageSticker is the short form of a sticker attached to a message.
type MessageSticker struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Animated bool   `json:"animated"`
}

// ReactionEmoji is the emoji of a reaction. ID is empty for Unicode emoji.
type ReactionEmoji struct {
	Name     string `json:"name"`
	ID       string `json:"id,omitempty"`
	Animated *bool  `json:"animated,omitempty"`
}

// String returns the emoji as it is written in message content: the Unicode emoji itself,
// or <:name:id> for a custom emoji.
func (e ReactionEmoji) String() string {
	if e.ID == "" {
		return e.Name
	}
	if e.Animated != nil && *e.Animated {
		return "<a:" + e.Name + ":" + e.ID + ">"
	}
	return "<:" + e.Name + ":" + e.ID + ">"
}

// APIName returns the emoji in the form the reaction endpoints expect: the Unicode emoji,
// or name:id for a custom emoji.
func (e ReactionEmoji) APIName() string {
	if e.ID == "" {
		return e.Name
	}
	return e.Name + ":" + e.ID
}

// MessageReaction is the aggregated count of one emoji on a message. Me reports whether
// the current user is among the reactors.
type MessageReaction struct {
	Emoji ReactionEmoji `json:"emoji"`
	Count int           `json:"count"`
	Me    bool          `json:"me,omitempty"`
}

// Reference types for MessageReference.Type. ReferenceDefault is a reply,
// ReferenceForward a forwarded message.
const (
	ReferenceDefault = 0
	ReferenceForward = 1
)

// MessageReference points at another message, for replies and forwards. A
// forward can be limited to some of the original's media with AttachmentIDs and
// EmbedIndices.
type MessageReference struct {
	MessageID     string   `json:"message_id,omitempty"`
	ChannelID     string   `json:"channel_id,omitempty"`
	GuildID       string   `json:"guild_id,omitempty"`
	Type          int      `json:"type,omitempty"`
	AttachmentIDs []string `json:"attachment_ids,omitempty"`
	EmbedIndices  []int    `json:"embed_indices,omitempty"`
}

// AllowedMentions limits which mentions in a message notify anyone. Parse and RepliedUser
// are always sent, even when nil or false.
type AllowedMentions struct {
	Parse       []string `json:"parse"`
	Users       []string `json:"users,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	RepliedUser bool     `json:"replied_user"`
}

// MessageSend is the body of a new message. Zero fields are omitted. If Nonce is empty,
// SendMessage and SendMessageFiles fill in a random one, which lets the client retry
// the request without creating duplicates.
type MessageSend struct {
	Content          string            `json:"content,omitempty"`
	Embeds           []Embed           `json:"embeds,omitempty"`
	MessageReference *MessageReference `json:"message_reference,omitempty"`
	AllowedMentions  *AllowedMentions  `json:"allowed_mentions,omitempty"`
	StickerIDs       []string          `json:"sticker_ids,omitempty"`
	Attachments      []UploadedFile    `json:"attachments,omitempty"`
	Flags            int               `json:"flags,omitempty"`
	TTS              bool              `json:"tts,omitempty"`
	Nonce            string            `json:"nonce,omitempty"`
}

func (m MessageSend) nonce() string { return m.Nonce }

// MessageEdit holds the fields to change on a message. Only non-nil fields are sent, so a
// pointer to an empty string or an empty slice removes the content or the embeds.
type MessageEdit struct {
	Content         *string          `json:"content,omitempty"`
	Embeds          *[]Embed         `json:"embeds,omitempty"`
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
	Flags           *int             `json:"flags,omitempty"`
}

// MessageQuery selects the messages returned by Messages. Before, After and Around are
// message IDs and are only sent when non-empty; a Limit of zero leaves the page size to
// the server.
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

// PinnedMessage is a pinned message together with the time it was pinned.
type PinnedMessage struct {
	Message  Message   `json:"message"`
	PinnedAt time.Time `json:"pinned_at"`
}

// ChannelPins is one page of pinned messages. While HasMore is true, pass the PinnedAt
// of the last item to Pins to get the next page.
type ChannelPins struct {
	Items   []PinnedMessage `json:"items"`
	HasMore bool            `json:"has_more"`
}

func messagesPath(channelID string) string { return "/channels/" + channelID + "/messages" }

// SendMessage posts a message to a channel. It sets a random nonce when m.Nonce is empty,
// so transient failures are retried without risking duplicate messages.
func (c *Client) SendMessage(ctx context.Context, channelID string, m MessageSend) (*Message, error) {
	return c.SendMessageFiles(ctx, channelID, m, nil)
}

// SendMessageFiles posts a message with file uploads as multipart form data. Without
// files it behaves like SendMessage.
func (c *Client) SendMessageFiles(ctx context.Context, channelID string, m MessageSend, files []File) (*Message, error) {
	if m.Nonce == "" {
		m.Nonce = newNonce()
	}
	var msg Message
	return &msg, c.DoFiles(ctx, http.MethodPost, messagesPath(channelID), m, files, &msg)
}

// SendText posts a message with plain text content.
func (c *Client) SendText(ctx context.Context, channelID, content string) (*Message, error) {
	return c.SendMessage(ctx, channelID, MessageSend{Content: content})
}

// Reply sends m to the channel of to as a reply to it, replacing any MessageReference
// already set on m.
func (c *Client) Reply(ctx context.Context, to *Message, m MessageSend) (*Message, error) {
	m.MessageReference = &MessageReference{MessageID: to.ID, ChannelID: to.ChannelID, GuildID: to.GuildID}
	return c.SendMessage(ctx, to.ChannelID, m)
}

// Forward forwards m to channelID. A forward carries no content of its own;
// Fluxer copies the original into the new message's MessageSnapshots.
func (c *Client) Forward(ctx context.Context, m *Message, channelID string) (*Message, error) {
	return c.SendMessage(ctx, channelID, MessageSend{MessageReference: &MessageReference{
		Type: ReferenceForward, MessageID: m.ID, ChannelID: m.ChannelID, GuildID: m.GuildID,
	}})
}

// EditMessage applies m to a message and returns the updated message.
func (c *Client) EditMessage(ctx context.Context, channelID, messageID string, m MessageEdit) (*Message, error) {
	var msg Message
	return &msg, c.Do(ctx, http.MethodPatch, messagesPath(channelID)+"/"+messageID, m, &msg)
}

// DeleteMessage deletes a message.
func (c *Client) DeleteMessage(ctx context.Context, channelID, messageID string) error {
	return c.Do(ctx, http.MethodDelete, messagesPath(channelID)+"/"+messageID, nil, nil)
}

// Message fetches a single message.
func (c *Client) Message(ctx context.Context, channelID, messageID string) (*Message, error) {
	var msg Message
	return &msg, c.Do(ctx, http.MethodGet, messagesPath(channelID)+"/"+messageID, nil, &msg)
}

// Messages fetches a page of messages from a channel as selected by q.
func (c *Client) Messages(ctx context.Context, channelID string, q MessageQuery) ([]Message, error) {
	path := messagesPath(channelID)
	if v := q.values(); len(v) > 0 {
		path += "?" + v.Encode()
	}
	var msgs []Message
	return msgs, c.Do(ctx, http.MethodGet, path, nil, &msgs)
}

// BulkDelete deletes several messages in a channel with one request.
func (c *Client) BulkDelete(ctx context.Context, channelID string, ids []string) error {
	return c.Do(ctx, http.MethodPost, messagesPath(channelID)+"/bulk-delete", map[string][]string{"message_ids": ids}, nil)
}

// DeleteAttachment removes one attachment from a message.
func (c *Client) DeleteAttachment(ctx context.Context, channelID, messageID, attachmentID string) error {
	return c.Do(ctx, http.MethodDelete, messagesPath(channelID)+"/"+messageID+"/attachments/"+attachmentID, nil, nil)
}

// Crosspost publishes a message in an announcement channel to all following channels.
func (c *Client) Crosspost(ctx context.Context, channelID, messageID string) (*Message, error) {
	var msg Message
	return &msg, c.Do(ctx, http.MethodPost, messagesPath(channelID)+"/"+messageID+"/crosspost", nil, &msg)
}

// Typing shows the typing indicator for the current user in a channel.
func (c *Client) Typing(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodPost, "/channels/"+channelID+"/typing", nil, nil)
}

// Pin pins a message in its channel (PUT /channels/{id}/pins/{message}).
func (c *Client) Pin(ctx context.Context, channelID, messageID string) error {
	return c.Do(ctx, http.MethodPut, "/channels/"+channelID+"/pins/"+messageID, nil, nil)
}

// Unpin unpins a message.
func (c *Client) Unpin(ctx context.Context, channelID, messageID string) error {
	return c.Do(ctx, http.MethodDelete, "/channels/"+channelID+"/pins/"+messageID, nil, nil)
}

// Pins lists pinned messages pinned before the given time (GET
// /channels/{id}/messages/pins). A zero before starts with the newest pin, and a limit of
// zero or less uses the server default.
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
