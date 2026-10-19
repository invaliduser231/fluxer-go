package fluxer

import (
	"encoding/json"
	"time"
)

type Event interface{ EventName() string }

type UnavailableGuild struct {
	ID          string `json:"id"`
	Unavailable bool   `json:"unavailable"`
}

type RTCRegion struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
}

type Ready struct {
	SessionID       string             `json:"session_id"`
	Version         int                `json:"version"`
	User            User               `json:"user"`
	Guilds          []UnavailableGuild `json:"guilds"`
	PrivateChannels []Channel          `json:"private_channels"`
	RTCRegions      []RTCRegion        `json:"rtc_regions"`
	CountryCode     string             `json:"country_code"`
	Shard           []int              `json:"shard,omitempty"`
}

type Resumed struct{}

type GuildState struct {
	ID          string       `json:"id"`
	Properties  Guild        `json:"properties"`
	Roles       []Role       `json:"roles"`
	Channels    []Channel    `json:"channels"`
	Emojis      []Emoji      `json:"emojis"`
	Stickers    []Sticker    `json:"stickers"`
	Members     []Member     `json:"members"`
	MemberCount int          `json:"member_count"`
	OnlineCount int          `json:"online_count"`
	VoiceStates []VoiceState `json:"voice_states"`
	JoinedAt    *time.Time   `json:"joined_at,omitempty"`
	Unavailable bool         `json:"unavailable,omitempty"`
	Threads     []Channel    `json:"threads,omitempty"`
}

func (g GuildState) Guild() Guild {
	out := g.Properties
	out.ID = g.ID
	out.Roles, out.Channels, out.Emojis, out.Stickers = g.Roles, g.Channels, g.Emojis, g.Stickers
	out.MemberCount, out.OnlineCount = g.MemberCount, g.OnlineCount
	return out
}

type GuildCreate struct{ GuildState }

type GuildSync struct{ GuildState }

type GuildUpdate struct{ Guild }

type GuildDelete struct {
	ID          string `json:"id"`
	Unavailable bool   `json:"unavailable,omitempty"`
}

type GuildRoleCreate struct {
	GuildID string `json:"guild_id"`
	Role    Role   `json:"role"`
}

type GuildRoleUpdate struct {
	GuildID string `json:"guild_id"`
	Role    Role   `json:"role"`
}

type GuildRoleUpdateBulk struct {
	GuildID string `json:"guild_id"`
	Roles   []Role `json:"roles"`
}

type GuildRoleDelete struct {
	GuildID string `json:"guild_id"`
	RoleID  string `json:"role_id"`
}

type GuildEmojisUpdate struct {
	GuildID string  `json:"guild_id"`
	Emojis  []Emoji `json:"emojis"`
}

type GuildStickersUpdate struct {
	GuildID  string    `json:"guild_id"`
	Stickers []Sticker `json:"stickers"`
}

type ChannelCreate struct{ Channel }

type ChannelUpdate struct{ Channel }

type ChannelUpdateBulk struct {
	GuildID  string    `json:"guild_id"`
	Channels []Channel `json:"channels"`
}

type ChannelDelete struct{ Channel }

type ChannelRecipientAdd struct {
	ChannelID string `json:"channel_id"`
	User      User   `json:"user"`
}

type ChannelRecipientRemove struct {
	ChannelID string `json:"channel_id"`
	User      User   `json:"user"`
}

type ChannelPinsUpdate struct {
	ChannelID        string     `json:"channel_id"`
	GuildID          string     `json:"guild_id,omitempty"`
	LastPinTimestamp *time.Time `json:"last_pin_timestamp"`
}

type WebhooksUpdate struct {
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
}

type InviteCreate struct{ Invite }

type InviteDelete struct {
	Code      string `json:"code"`
	ChannelID string `json:"channel_id,omitempty"`
	GuildID   string `json:"guild_id,omitempty"`
}

type ThreadCreate struct {
	Channel
	NewlyCreated bool `json:"newly_created,omitempty"`
}

type ThreadUpdate struct{ Channel }

type ThreadDelete struct {
	ID       string `json:"id"`
	GuildID  string `json:"guild_id"`
	ParentID string `json:"parent_id"`
	Type     int    `json:"type"`
}

type ThreadListSync struct {
	GuildID    string         `json:"guild_id"`
	ChannelIDs []string       `json:"channel_ids,omitempty"`
	Threads    []Channel      `json:"threads"`
	Members    []ThreadMember `json:"members"`
}

type ThreadMemberUpdate struct {
	ThreadMember
	GuildID string `json:"guild_id"`
}

type ThreadMembersUpdate struct {
	ID               string         `json:"id"`
	GuildID          string         `json:"guild_id"`
	MemberCount      int            `json:"member_count"`
	AddedMembers     []ThreadMember `json:"added_members,omitempty"`
	RemovedMemberIDs []string       `json:"removed_member_ids,omitempty"`
}

type GuildMemberAdd struct{ Member }

type GuildMemberUpdate struct{ Member }

type GuildMemberRemove struct {
	GuildID string `json:"guild_id"`
	User    User   `json:"user"`
}

type GuildMembersChunk struct {
	GuildID    string     `json:"guild_id"`
	Members    []Member   `json:"members"`
	ChunkIndex int        `json:"chunk_index"`
	ChunkCount int        `json:"chunk_count"`
	Presences  []Presence `json:"presences,omitempty"`
	Nonce      string     `json:"nonce,omitempty"`
}

type GuildBanAdd struct {
	GuildID string `json:"guild_id"`
	User    User   `json:"user"`
}

type GuildBanRemove struct {
	GuildID string `json:"guild_id"`
	User    User   `json:"user"`
}

type CustomStatus struct {
	Text          *string    `json:"text,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	EmojiID       *string    `json:"emoji_id,omitempty"`
	EmojiName     *string    `json:"emoji_name,omitempty"`
	EmojiAnimated bool       `json:"emoji_animated,omitempty"`
}

type Presence struct {
	User         User          `json:"user"`
	Status       string        `json:"status"`
	Mobile       bool          `json:"mobile"`
	AFK          bool          `json:"afk"`
	CustomStatus *CustomStatus `json:"custom_status"`
	GuildID      string        `json:"guild_id,omitempty"`
}

type PresenceUpdate struct{ Presence }

type PresenceUpdateBulk struct {
	GuildID   string     `json:"guild_id"`
	Presences []Presence `json:"presences"`
}

type PassiveUpdates struct {
	GuildID     string            `json:"guild_id"`
	Channels    map[string]string `json:"channels"`
	VoiceStates []VoiceState      `json:"voice_states,omitempty"`
}

type MessageCreate struct {
	Message
	ChannelType int  `json:"channel_type"`
	MentionHere bool `json:"mention_here,omitempty"`
}

type MessageUpdate struct{ Message }

type MessageDelete struct {
	ID        string  `json:"id"`
	ChannelID string  `json:"channel_id"`
	GuildID   string  `json:"guild_id,omitempty"`
	Content   *string `json:"content,omitempty"`
	AuthorID  string  `json:"author_id,omitempty"`
	Member    *Member `json:"member,omitempty"`
}

type MessageDeleteBulk struct {
	IDs       []string `json:"ids"`
	ChannelID string   `json:"channel_id"`
	GuildID   string   `json:"guild_id,omitempty"`
}

type MessageReactionAdd struct {
	UserID    string        `json:"user_id"`
	ChannelID string        `json:"channel_id"`
	MessageID string        `json:"message_id"`
	GuildID   string        `json:"guild_id,omitempty"`
	Emoji     ReactionEmoji `json:"emoji"`
	Member    *Member       `json:"member,omitempty"`
}

type ReactionAddition struct {
	UserID string        `json:"user_id"`
	Emoji  ReactionEmoji `json:"emoji"`
	Member *Member       `json:"member,omitempty"`
}

type MessageReactionAddMany struct {
	ChannelID string             `json:"channel_id"`
	MessageID string             `json:"message_id"`
	GuildID   string             `json:"guild_id,omitempty"`
	Reactions []ReactionAddition `json:"reactions"`
}

type MessageReactionRemove struct {
	UserID    string        `json:"user_id"`
	ChannelID string        `json:"channel_id"`
	MessageID string        `json:"message_id"`
	GuildID   string        `json:"guild_id,omitempty"`
	Emoji     ReactionEmoji `json:"emoji"`
}

type MessageReactionRemoveAll struct {
	ChannelID string `json:"channel_id"`
	MessageID string `json:"message_id"`
	GuildID   string `json:"guild_id,omitempty"`
}

type MessageReactionRemoveEmoji struct {
	ChannelID string        `json:"channel_id"`
	MessageID string        `json:"message_id"`
	GuildID   string        `json:"guild_id,omitempty"`
	Emoji     ReactionEmoji `json:"emoji"`
}

type TypingStart struct {
	ChannelID string  `json:"channel_id"`
	UserID    string  `json:"user_id"`
	Timestamp int64   `json:"timestamp"`
	GuildID   string  `json:"guild_id,omitempty"`
	Member    *Member `json:"member,omitempty"`
}

type VoiceState struct {
	GuildID          string   `json:"guild_id,omitempty"`
	ChannelID        string   `json:"channel_id,omitempty"`
	UserID           string   `json:"user_id"`
	ConnectionID     string   `json:"connection_id,omitempty"`
	SessionID        string   `json:"session_id,omitempty"`
	Member           *Member  `json:"member,omitempty"`
	Mute             bool     `json:"mute"`
	Deaf             bool     `json:"deaf"`
	SelfMute         bool     `json:"self_mute"`
	SelfDeaf         bool     `json:"self_deaf"`
	SelfVideo        bool     `json:"self_video"`
	SelfStream       bool     `json:"self_stream"`
	IsMobile         bool     `json:"is_mobile"`
	Suppress         bool     `json:"suppress"`
	ViewerStreamKeys []string `json:"viewer_stream_keys,omitempty"`
	Version          int64    `json:"version"`
}

type VoiceStateUpdate struct{ VoiceState }

type VoiceServerUpdate struct {
	Token        string `json:"token"`
	Endpoint     string `json:"endpoint"`
	ConnectionID string `json:"connection_id"`
	ChannelID    string `json:"channel_id"`
	GuildID      string `json:"guild_id,omitempty"`
	E2EEKey      string `json:"e2ee_key,omitempty"`
}

type EntranceSoundPlay struct {
	UserID      string `json:"user_id"`
	ChannelID   string `json:"channel_id"`
	GuildID     string `json:"guild_id,omitempty"`
	SoundID     string `json:"sound_id"`
	Hash        string `json:"hash"`
	URL         string `json:"url"`
	DurationMS  int    `json:"duration_ms"`
	ContentType string `json:"content_type"`
}

type Call struct {
	ChannelID   string       `json:"channel_id"`
	MessageID   string       `json:"message_id"`
	Region      *string      `json:"region"`
	Ringing     []string     `json:"ringing"`
	VoiceStates []VoiceState `json:"voice_states"`
}

type CallCreate struct {
	Call
	Recipients []string `json:"recipients,omitempty"`
}

type CallUpdate struct{ Call }

type CallDelete struct {
	ChannelID   string `json:"channel_id"`
	Unavailable bool   `json:"unavailable,omitempty"`
}

type GuildCount struct {
	GuildID     string `json:"guild_id"`
	MemberCount int    `json:"member_count"`
	OnlineCount int    `json:"online_count"`
}

type GuildCountsUpdate struct {
	Counts []GuildCount `json:"counts"`
	Nonce  string       `json:"nonce,omitempty"`
}

type ChannelMemberCount struct {
	GuildID     string `json:"guild_id"`
	ChannelID   string `json:"channel_id"`
	MemberCount int    `json:"member_count"`
	OnlineCount int    `json:"online_count"`
}

type ChannelMemberCountsUpdate struct {
	Counts []ChannelMemberCount `json:"counts"`
	Nonce  string               `json:"nonce,omitempty"`
}

type RateLimited struct {
	Opcode     int               `json:"opcode"`
	RetryAfter float64           `json:"retry_after"`
	Meta       map[string]string `json:"meta"`
}

type UserUpdate struct{ User }

func (Ready) EventName() string                      { return "READY" }
func (Resumed) EventName() string                    { return "RESUMED" }
func (GuildCreate) EventName() string                { return "GUILD_CREATE" }
func (GuildSync) EventName() string                  { return "GUILD_SYNC" }
func (GuildUpdate) EventName() string                { return "GUILD_UPDATE" }
func (GuildDelete) EventName() string                { return "GUILD_DELETE" }
func (GuildRoleCreate) EventName() string            { return "GUILD_ROLE_CREATE" }
func (GuildRoleUpdate) EventName() string            { return "GUILD_ROLE_UPDATE" }
func (GuildRoleUpdateBulk) EventName() string        { return "GUILD_ROLE_UPDATE_BULK" }
func (GuildRoleDelete) EventName() string            { return "GUILD_ROLE_DELETE" }
func (GuildEmojisUpdate) EventName() string          { return "GUILD_EMOJIS_UPDATE" }
func (GuildStickersUpdate) EventName() string        { return "GUILD_STICKERS_UPDATE" }
func (ChannelCreate) EventName() string              { return "CHANNEL_CREATE" }
func (ChannelUpdate) EventName() string              { return "CHANNEL_UPDATE" }
func (ChannelUpdateBulk) EventName() string          { return "CHANNEL_UPDATE_BULK" }
func (ChannelDelete) EventName() string              { return "CHANNEL_DELETE" }
func (ChannelRecipientAdd) EventName() string        { return "CHANNEL_RECIPIENT_ADD" }
func (ChannelRecipientRemove) EventName() string     { return "CHANNEL_RECIPIENT_REMOVE" }
func (ChannelPinsUpdate) EventName() string          { return "CHANNEL_PINS_UPDATE" }
func (WebhooksUpdate) EventName() string             { return "WEBHOOKS_UPDATE" }
func (InviteCreate) EventName() string               { return "INVITE_CREATE" }
func (InviteDelete) EventName() string               { return "INVITE_DELETE" }
func (ThreadCreate) EventName() string               { return "THREAD_CREATE" }
func (ThreadUpdate) EventName() string               { return "THREAD_UPDATE" }
func (ThreadDelete) EventName() string               { return "THREAD_DELETE" }
func (ThreadListSync) EventName() string             { return "THREAD_LIST_SYNC" }
func (ThreadMemberUpdate) EventName() string         { return "THREAD_MEMBER_UPDATE" }
func (ThreadMembersUpdate) EventName() string        { return "THREAD_MEMBERS_UPDATE" }
func (GuildMemberAdd) EventName() string             { return "GUILD_MEMBER_ADD" }
func (GuildMemberUpdate) EventName() string          { return "GUILD_MEMBER_UPDATE" }
func (GuildMemberRemove) EventName() string          { return "GUILD_MEMBER_REMOVE" }
func (GuildMembersChunk) EventName() string          { return "GUILD_MEMBERS_CHUNK" }
func (GuildBanAdd) EventName() string                { return "GUILD_BAN_ADD" }
func (GuildBanRemove) EventName() string             { return "GUILD_BAN_REMOVE" }
func (PresenceUpdate) EventName() string             { return "PRESENCE_UPDATE" }
func (PresenceUpdateBulk) EventName() string         { return "PRESENCE_UPDATE_BULK" }
func (PassiveUpdates) EventName() string             { return "PASSIVE_UPDATES" }
func (MessageCreate) EventName() string              { return "MESSAGE_CREATE" }
func (MessageUpdate) EventName() string              { return "MESSAGE_UPDATE" }
func (MessageDelete) EventName() string              { return "MESSAGE_DELETE" }
func (MessageDeleteBulk) EventName() string          { return "MESSAGE_DELETE_BULK" }
func (MessageReactionAdd) EventName() string         { return "MESSAGE_REACTION_ADD" }
func (MessageReactionAddMany) EventName() string     { return "MESSAGE_REACTION_ADD_MANY" }
func (MessageReactionRemove) EventName() string      { return "MESSAGE_REACTION_REMOVE" }
func (MessageReactionRemoveAll) EventName() string   { return "MESSAGE_REACTION_REMOVE_ALL" }
func (MessageReactionRemoveEmoji) EventName() string { return "MESSAGE_REACTION_REMOVE_EMOJI" }
func (TypingStart) EventName() string                { return "TYPING_START" }
func (VoiceStateUpdate) EventName() string           { return "VOICE_STATE_UPDATE" }
func (VoiceServerUpdate) EventName() string          { return "VOICE_SERVER_UPDATE" }
func (EntranceSoundPlay) EventName() string          { return "ENTRANCE_SOUND_PLAY" }
func (CallCreate) EventName() string                 { return "CALL_CREATE" }
func (CallUpdate) EventName() string                 { return "CALL_UPDATE" }
func (CallDelete) EventName() string                 { return "CALL_DELETE" }
func (GuildCountsUpdate) EventName() string          { return "GUILD_COUNTS_UPDATE" }
func (ChannelMemberCountsUpdate) EventName() string  { return "CHANNEL_MEMBER_COUNTS_UPDATE" }
func (RateLimited) EventName() string                { return "RATE_LIMITED" }
func (UserUpdate) EventName() string                 { return "USER_UPDATE" }

var eventTypes = map[string]func() Event{}

func registerEvents(fs ...func() Event) {
	for _, f := range fs {
		eventTypes[f().EventName()] = f
	}
}

func init() {
	registerEvents(
		func() Event { return &Ready{} }, func() Event { return &Resumed{} },
		func() Event { return &GuildCreate{} }, func() Event { return &GuildSync{} },
		func() Event { return &GuildUpdate{} }, func() Event { return &GuildDelete{} },
		func() Event { return &GuildRoleCreate{} }, func() Event { return &GuildRoleUpdate{} },
		func() Event { return &GuildRoleUpdateBulk{} }, func() Event { return &GuildRoleDelete{} },
		func() Event { return &GuildEmojisUpdate{} }, func() Event { return &GuildStickersUpdate{} },
		func() Event { return &ChannelCreate{} }, func() Event { return &ChannelUpdate{} },
		func() Event { return &ChannelUpdateBulk{} }, func() Event { return &ChannelDelete{} },
		func() Event { return &ChannelRecipientAdd{} }, func() Event { return &ChannelRecipientRemove{} },
		func() Event { return &ChannelPinsUpdate{} }, func() Event { return &WebhooksUpdate{} },
		func() Event { return &InviteCreate{} }, func() Event { return &InviteDelete{} },
		func() Event { return &ThreadCreate{} }, func() Event { return &ThreadUpdate{} },
		func() Event { return &ThreadDelete{} }, func() Event { return &ThreadListSync{} },
		func() Event { return &ThreadMemberUpdate{} }, func() Event { return &ThreadMembersUpdate{} },
		func() Event { return &GuildMemberAdd{} }, func() Event { return &GuildMemberUpdate{} },
		func() Event { return &GuildMemberRemove{} }, func() Event { return &GuildMembersChunk{} },
		func() Event { return &GuildBanAdd{} }, func() Event { return &GuildBanRemove{} },
		func() Event { return &PresenceUpdate{} }, func() Event { return &PresenceUpdateBulk{} },
		func() Event { return &PassiveUpdates{} },
		func() Event { return &MessageCreate{} }, func() Event { return &MessageUpdate{} },
		func() Event { return &MessageDelete{} }, func() Event { return &MessageDeleteBulk{} },
		func() Event { return &MessageReactionAdd{} }, func() Event { return &MessageReactionAddMany{} },
		func() Event { return &MessageReactionRemove{} }, func() Event { return &MessageReactionRemoveAll{} },
		func() Event { return &MessageReactionRemoveEmoji{} }, func() Event { return &TypingStart{} },
		func() Event { return &VoiceStateUpdate{} }, func() Event { return &VoiceServerUpdate{} },
		func() Event { return &EntranceSoundPlay{} },
		func() Event { return &CallCreate{} }, func() Event { return &CallUpdate{} }, func() Event { return &CallDelete{} },
		func() Event { return &GuildCountsUpdate{} }, func() Event { return &ChannelMemberCountsUpdate{} },
		func() Event { return &RateLimited{} }, func() Event { return &UserUpdate{} },
	)
}

func Decode(event string, data json.RawMessage) (Event, error) {
	f, ok := eventTypes[event]
	if !ok {
		return nil, nil
	}
	e := f()
	if len(data) == 0 || string(data) == "null" {
		return e, nil
	}
	return e, json.Unmarshal(data, e)
}
