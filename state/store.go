package state

import (
	"iter"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

// Table is a two-level key-value table used by a Store. Values live under a group,
// usually a guild or channel ID, and a key inside that group.
type Table[V any] interface {
	// Get returns the value stored under group and key.
	Get(group, key string) (V, bool)
	// Put stores v under group and key, replacing any previous value.
	Put(group, key string, v V)
	// Delete removes one value.
	Delete(group, key string)
	// DeleteGroup removes every value in group.
	DeleteGroup(group string)
	// Group returns the entries of group. The entries are read when Group is called, so
	// the table can be changed while iterating.
	Group(group string) iter.Seq2[string, V]
	// Len returns the number of values in group.
	Len(group string) int
	// Groups returns the names of all groups that hold at least one value.
	Groups() iter.Seq[string]
	// Locate returns the group that holds key. Tables that keep no key index always
	// return false.
	Locate(key string) (string, bool)
}

// GuildRecord is the stored form of a guild. Guild has no roles, channels, emojis or
// stickers; the cache keeps those in their own tables and adds them back on read.
type GuildRecord struct {
	Guild fluxer.Guild `json:"guild"`
	// Unavailable is set while the gateway reports the guild as unavailable.
	Unavailable bool `json:"unavailable,omitempty"`
	// JoinedAt is when the bot joined the guild, if the gateway sent it.
	JoinedAt *time.Time `json:"joined_at,omitempty"`
}

// Store holds the tables a Cache reads and writes. NewMemoryStore is the default;
// the state/redis module provides a Store that several processes can share.
type Store interface {
	// Guilds holds guild records under the empty group, keyed by guild ID.
	Guilds() Table[GuildRecord]
	// Channels holds channels grouped by guild ID, with DM channels under the empty
	// group. The cache uses Locate to find a channel by ID alone.
	Channels() Table[fluxer.Channel]
	// Roles holds roles grouped by guild ID.
	Roles() Table[fluxer.Role]
	// Members holds members grouped by guild ID and keyed by user ID.
	Members() Table[fluxer.Member]
	// VoiceStates holds voice states grouped by guild ID and keyed by connection ID, or
	// by user ID when a state has no connection ID.
	VoiceStates() Table[fluxer.VoiceState]
	// Emojis holds custom emojis grouped by guild ID.
	Emojis() Table[fluxer.Emoji]
	// Stickers holds stickers grouped by guild ID.
	Stickers() Table[fluxer.Sticker]
	// Users holds users under the empty group, keyed by user ID.
	Users() Table[fluxer.User]
	// Presences holds presences grouped by guild ID and keyed by user ID.
	Presences() Table[fluxer.Presence]
	// Messages holds messages grouped by channel ID. Cache.MessageByID needs Locate to
	// work on this table.
	Messages() Table[fluxer.Message]
}
