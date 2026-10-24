package state

import (
	"slices"

	"github.com/invaliduser231/fluxer-go"
)

// Flags selects which kinds of data a Cache stores.
type Flags uint32

// Flags for the individual tables. FlagThreads covers thread channels and
// FlagChannels all other channels. FlagsAll is the default for New.
const (
	FlagGuilds Flags = 1 << iota
	FlagChannels
	FlagThreads
	FlagRoles
	FlagMembers
	FlagVoiceStates
	FlagEmojis
	FlagStickers
	FlagUsers
	FlagPresences
	FlagMessages

	FlagsAll = FlagGuilds | FlagChannels | FlagThreads | FlagRoles | FlagMembers | FlagVoiceStates | FlagEmojis | FlagStickers |
		FlagUsers | FlagPresences | FlagMessages
)

// Has reports whether every bit of flag is set in f.
func (f Flags) Has(flag Flags) bool { return f&flag == flag }

// Policy decides whether a value is kept in the cache. It runs on every write of
// that kind of value, so it should be cheap, and it must not apply events to the
// same cache.
type Policy[T any] func(c *Cache, v T) bool

// And returns a policy that keeps a value only if both p and other keep it.
func (p Policy[T]) And(other Policy[T]) Policy[T] {
	return func(c *Cache, v T) bool { return p(c, v) && other(c, v) }
}

// Or returns a policy that keeps a value if p or other keeps it.
func (p Policy[T]) Or(other Policy[T]) Policy[T] {
	return func(c *Cache, v T) bool { return p(c, v) || other(c, v) }
}

// Not returns a policy that keeps exactly the values p rejects.
func (p Policy[T]) Not() Policy[T] { return func(c *Cache, v T) bool { return !p(c, v) } }

// All returns a policy that keeps every value.
func All[T any]() Policy[T] { return func(*Cache, T) bool { return true } }

// None returns a policy that keeps no value.
func None[T any]() Policy[T] { return func(*Cache, T) bool { return false } }

// MembersInGuilds keeps members of the listed guilds.
func MembersInGuilds(guildIDs ...string) Policy[fluxer.Member] {
	return func(_ *Cache, m fluxer.Member) bool { return slices.Contains(guildIDs, m.GuildID) }
}

// MembersWithRole keeps members that have at least one of the listed roles.
func MembersWithRole(roleIDs ...string) Policy[fluxer.Member] {
	return func(_ *Cache, m fluxer.Member) bool {
		for _, id := range m.Roles {
			if slices.Contains(roleIDs, id) {
				return true
			}
		}
		return false
	}
}

// MembersInVoice keeps members that are in a voice channel according to the cached
// voice states, so it needs FlagVoiceStates. A member is dropped when its voice
// state shows that it left.
func MembersInVoice() Policy[fluxer.Member] {
	return func(c *Cache, m fluxer.Member) bool {
		return m.User != nil && c.VoiceChannel(m.GuildID, m.User.ID) != ""
	}
}

// MemberOwner keeps the owner of each cached guild.
func MemberOwner() Policy[fluxer.Member] {
	return func(c *Cache, m fluxer.Member) bool {
		rec, ok := c.store.Guilds().Get("", m.GuildID)
		return ok && m.User != nil && rec.Guild.OwnerID == m.User.ID
	}
}

// MembersOnline keeps members with a cached presence that is not offline. Members
// who go offline stay cached until their next member update or SweepMembers.
func MembersOnline() Policy[fluxer.Member] {
	return func(c *Cache, m fluxer.Member) bool {
		if m.User == nil {
			return false
		}
		p, ok := c.Presence(m.GuildID, m.User.ID)
		return ok && p.Status != fluxer.StatusOffline
	}
}

// MembersTimedOut keeps members whose timeout has not ended yet. Members whose
// timeout runs out stay cached until their next member update or SweepMembers.
func MembersTimedOut() Policy[fluxer.Member] {
	return func(_ *Cache, m fluxer.Member) bool { return m.TimedOut() }
}

// ChannelTypes keeps channels whose type is one of types.
func ChannelTypes(types ...int) Policy[fluxer.Channel] {
	return func(_ *Cache, ch fluxer.Channel) bool { return slices.Contains(types, ch.Type) }
}

// WithFlags replaces the cache flags, which default to FlagsAll.
func WithFlags(f Flags) Option { return func(c *Cache) { c.flags = f } }

// WithoutFlags clears f from the flags set so far.
func WithoutFlags(f Flags) Option { return func(c *Cache) { c.flags &^= f } }

// WithMemberPolicy limits which members are cached. The bot's own member is always
// kept, regardless of the policy, FlagMembers and SkipMembers. A cached member that
// stops matching on a later update is removed.
func WithMemberPolicy(p Policy[fluxer.Member]) Option { return func(c *Cache) { c.memberPolicy = p } }

// WithChannelPolicy limits which channels and threads are cached. A cached channel
// that stops matching on a later update is removed.
func WithChannelPolicy(p Policy[fluxer.Channel]) Option {
	return func(c *Cache) { c.channelPolicy = p }
}

// Flags returns the flags the cache was configured with.
func (c *Cache) Flags() Flags { return c.flags }

func (c *Cache) keepMember(m fluxer.Member) bool {
	if m.User != nil && m.User.ID == c.Self().ID {
		return true
	}
	if !c.flags.Has(FlagMembers) || c.SkipMembers {
		return false
	}
	return c.memberPolicy == nil || c.memberPolicy(c, m)
}

func (c *Cache) keepChannel(ch fluxer.Channel) bool {
	flag := FlagChannels
	if ch.IsThread() {
		flag = FlagThreads
	}
	if !c.flags.Has(flag) {
		return false
	}
	return c.channelPolicy == nil || c.channelPolicy(c, ch)
}

// SweepMembers checks every cached member against the flags, SkipMembers and the
// member policy again and removes the ones that no longer pass. It returns the
// number of members removed.
func (c *Cache) SweepMembers() int {
	removed := 0
	for guildID := range c.store.Members().Groups() {
		unlock := c.lock(guildID)
		for userID, m := range c.store.Members().Group(guildID) {
			if !c.keepMember(m) {
				c.store.Members().Delete(guildID, userID)
				removed++
			}
		}
		unlock()
	}
	return removed
}
