package fluxer

import "context"

const (
	opPresenceUpdate             = 3
	opRequestGuildMembers        = 8
	opLazyRequest                = 14
	opRequestGuildCounts         = 15
	opRequestChannelMemberCounts = 16
)

// Flags for Session.Flags, sent with IDENTIFY.
const (
	FlagDebounceMessageReactions = 1 << 1
	FlagChannelThreads           = 1 << 2
)

// Presence status values for StatusUpdate and Presence.
const (
	StatusOnline    = "online"
	StatusIdle      = "idle"
	StatusDND       = "dnd"
	StatusInvisible = "invisible"
	StatusOffline   = "offline"
)

const maxRequestGuildMembersUserIDs = 100

// StatusUpdate is the presence the bot sends with UpdateStatus or Session.Presence.
type StatusUpdate struct {
	Status       string        `json:"status"`
	AFK          bool          `json:"afk"`
	Mobile       bool          `json:"mobile"`
	CustomStatus *CustomStatus `json:"custom_status,omitempty"`
}

// RequestMembers is the payload of RequestGuildMembers. Fluxer answers with GUILD_MEMBERS_CHUNK
// dispatches that carry the same Nonce.
type RequestMembers struct {
	GuildID   string   `json:"guild_id"`
	Query     string   `json:"query,omitempty"`
	Limit     int      `json:"limit"`
	UserIDs   []string `json:"user_ids,omitempty"`
	Presences bool     `json:"presences,omitempty"`
	Nonce     string   `json:"nonce,omitempty"`
}

// GuildSubscription is what the session subscribes to in one guild through Subscribe.
// MemberListChannels maps channel IDs to member list index ranges. Nil pointer fields are left out
// of the request.
type GuildSubscription struct {
	Active             *bool               `json:"active,omitempty"`
	Sync               *bool               `json:"sync,omitempty"`
	Typing             *bool               `json:"typing,omitempty"`
	MemberListChannels map[string][][2]int `json:"member_list_channels,omitempty"`
	Members            []string            `json:"members,omitempty"`
}

func (s *Session) setLive(conn *wire) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.live = conn
}

// Send writes a gateway command with opcode op and payload d on the live connection. It returns
// ErrNotConnected until the session has received READY or RESUMED, and again while it reconnects or
// identifies after INVALID_SESSION.
func (s *Session) Send(ctx context.Context, op int, d any) error {
	s.mu.Lock()
	conn := s.live
	s.mu.Unlock()
	if conn == nil {
		return ErrNotConnected
	}
	return s.send(ctx, conn, op, d)
}

// UpdateStatus changes the bot's presence.
func (s *Session) UpdateStatus(ctx context.Context, st StatusUpdate) error {
	return s.Send(ctx, opPresenceUpdate, st)
}

// RequestGuildMembers asks for members of one guild. Only the first 100 UserIDs are sent and the
// rest are dropped.
func (s *Session) RequestGuildMembers(ctx context.Context, r RequestMembers) error {
	if len(r.UserIDs) > maxRequestGuildMembersUserIDs {
		r.UserIDs = r.UserIDs[:maxRequestGuildMembersUserIDs]
	}
	return s.Send(ctx, opRequestGuildMembers, r)
}

// Subscribe sends a lazy guild request with subscriptions keyed by guild ID.
func (s *Session) Subscribe(ctx context.Context, subscriptions map[string]GuildSubscription) error {
	return s.Send(ctx, opLazyRequest, map[string]any{"subscriptions": subscriptions})
}

// RequestGuildCounts asks for the member and online counts of the given guilds. The answer arrives
// as GUILD_COUNTS_UPDATE. An empty nonce is left out.
func (s *Session) RequestGuildCounts(ctx context.Context, guildIDs []string, nonce string) error {
	d := map[string]any{"guild_ids": guildIDs}
	if nonce != "" {
		d["nonce"] = nonce
	}
	return s.Send(ctx, opRequestGuildCounts, d)
}

// RequestChannelMemberCounts asks for the member and online counts of channels in one guild. The
// answer arrives as CHANNEL_MEMBER_COUNTS_UPDATE. An empty nonce is left out.
func (s *Session) RequestChannelMemberCounts(ctx context.Context, guildID string, channelIDs []string, nonce string) error {
	d := map[string]any{"guild_id": guildID, "channel_ids": channelIDs}
	if nonce != "" {
		d["nonce"] = nonce
	}
	return s.Send(ctx, opRequestChannelMemberCounts, d)
}
