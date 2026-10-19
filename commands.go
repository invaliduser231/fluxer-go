package fluxer

import (
	"context"

	"github.com/coder/websocket"
)

const (
	opPresenceUpdate             = 3
	opRequestGuildMembers        = 8
	opLazyRequest                = 14
	opRequestGuildCounts         = 15
	opRequestChannelMemberCounts = 16
)

const (
	FlagDebounceMessageReactions = 1 << 1
	FlagChannelThreads           = 1 << 2
)

const (
	StatusOnline    = "online"
	StatusIdle      = "idle"
	StatusDND       = "dnd"
	StatusInvisible = "invisible"
	StatusOffline   = "offline"
)

const maxRequestGuildMembersUserIDs = 100

type StatusUpdate struct {
	Status       string        `json:"status"`
	AFK          bool          `json:"afk"`
	Mobile       bool          `json:"mobile"`
	CustomStatus *CustomStatus `json:"custom_status,omitempty"`
}

type RequestMembers struct {
	GuildID   string   `json:"guild_id"`
	Query     string   `json:"query,omitempty"`
	Limit     int      `json:"limit"`
	UserIDs   []string `json:"user_ids,omitempty"`
	Presences bool     `json:"presences,omitempty"`
	Nonce     string   `json:"nonce,omitempty"`
}

type GuildSubscription struct {
	Active             *bool               `json:"active,omitempty"`
	Sync               *bool               `json:"sync,omitempty"`
	Typing             *bool               `json:"typing,omitempty"`
	MemberListChannels map[string][][2]int `json:"member_list_channels,omitempty"`
	Members            []string            `json:"members,omitempty"`
}

func (s *Session) setLive(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.live = conn
}

func (s *Session) Send(ctx context.Context, op int, d any) error {
	s.mu.Lock()
	conn := s.live
	s.mu.Unlock()
	if conn == nil {
		return ErrNotConnected
	}
	return s.send(ctx, conn, op, d)
}

func (s *Session) UpdateStatus(ctx context.Context, st StatusUpdate) error {
	return s.Send(ctx, opPresenceUpdate, st)
}

func (s *Session) RequestGuildMembers(ctx context.Context, r RequestMembers) error {
	if len(r.UserIDs) > maxRequestGuildMembersUserIDs {
		r.UserIDs = r.UserIDs[:maxRequestGuildMembersUserIDs]
	}
	return s.Send(ctx, opRequestGuildMembers, r)
}

func (s *Session) Subscribe(ctx context.Context, subscriptions map[string]GuildSubscription) error {
	return s.Send(ctx, opLazyRequest, map[string]any{"subscriptions": subscriptions})
}

func (s *Session) RequestGuildCounts(ctx context.Context, guildIDs []string, nonce string) error {
	d := map[string]any{"guild_ids": guildIDs}
	if nonce != "" {
		d["nonce"] = nonce
	}
	return s.Send(ctx, opRequestGuildCounts, d)
}

func (s *Session) RequestChannelMemberCounts(ctx context.Context, guildID string, channelIDs []string, nonce string) error {
	d := map[string]any{"guild_id": guildID, "channel_ids": channelIDs}
	if nonce != "" {
		d["nonce"] = nonce
	}
	return s.Send(ctx, opRequestChannelMemberCounts, d)
}
