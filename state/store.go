package state

import (
	"iter"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

type Table[V any] interface {
	Get(group, key string) (V, bool)
	Put(group, key string, v V)
	Delete(group, key string)
	DeleteGroup(group string)
	Group(group string) iter.Seq2[string, V]
	Len(group string) int
	Groups() iter.Seq[string]
	Locate(key string) (string, bool)
}

type GuildRecord struct {
	Guild       fluxer.Guild `json:"guild"`
	Unavailable bool         `json:"unavailable,omitempty"`
	JoinedAt    *time.Time   `json:"joined_at,omitempty"`
}

type Store interface {
	Guilds() Table[GuildRecord]
	Channels() Table[fluxer.Channel]
	Roles() Table[fluxer.Role]
	Members() Table[fluxer.Member]
	VoiceStates() Table[fluxer.VoiceState]
	Emojis() Table[fluxer.Emoji]
	Stickers() Table[fluxer.Sticker]
	Users() Table[fluxer.User]
	Presences() Table[fluxer.Presence]
	Messages() Table[fluxer.Message]
}
