package state

import (
	"slices"

	"github.com/invaliduser231/fluxer-go"
)

type Flags uint32

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

	FlagsAll = FlagGuilds | FlagChannels | FlagThreads | FlagRoles | FlagMembers | FlagVoiceStates | FlagEmojis | FlagStickers |
		FlagUsers | FlagPresences
)

func (f Flags) Has(flag Flags) bool { return f&flag == flag }

type Policy[T any] func(c *Cache, v T) bool

func (p Policy[T]) And(other Policy[T]) Policy[T] {
	return func(c *Cache, v T) bool { return p(c, v) && other(c, v) }
}

func (p Policy[T]) Or(other Policy[T]) Policy[T] {
	return func(c *Cache, v T) bool { return p(c, v) || other(c, v) }
}

func (p Policy[T]) Not() Policy[T] { return func(c *Cache, v T) bool { return !p(c, v) } }

func All[T any]() Policy[T] { return func(*Cache, T) bool { return true } }

func None[T any]() Policy[T] { return func(*Cache, T) bool { return false } }

func MembersInGuilds(guildIDs ...string) Policy[fluxer.Member] {
	return func(_ *Cache, m fluxer.Member) bool { return slices.Contains(guildIDs, m.GuildID) }
}

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

func MembersInVoice() Policy[fluxer.Member] {
	return func(c *Cache, m fluxer.Member) bool {
		return m.User != nil && c.VoiceChannel(m.GuildID, m.User.ID) != ""
	}
}

func MemberOwner() Policy[fluxer.Member] {
	return func(c *Cache, m fluxer.Member) bool {
		rec, ok := c.store.Guilds().Get("", m.GuildID)
		return ok && m.User != nil && rec.Guild.OwnerID == m.User.ID
	}
}

func MembersOnline() Policy[fluxer.Member] {
	return func(c *Cache, m fluxer.Member) bool {
		if m.User == nil {
			return false
		}
		p, ok := c.Presence(m.GuildID, m.User.ID)
		return ok && p.Status != fluxer.StatusOffline
	}
}

func MembersTimedOut() Policy[fluxer.Member] {
	return func(_ *Cache, m fluxer.Member) bool { return m.TimedOut() }
}

func ChannelTypes(types ...int) Policy[fluxer.Channel] {
	return func(_ *Cache, ch fluxer.Channel) bool { return slices.Contains(types, ch.Type) }
}

func WithFlags(f Flags) Option { return func(c *Cache) { c.flags = f } }

func WithoutFlags(f Flags) Option { return func(c *Cache) { c.flags &^= f } }

func WithMemberPolicy(p Policy[fluxer.Member]) Option { return func(c *Cache) { c.memberPolicy = p } }

func WithChannelPolicy(p Policy[fluxer.Channel]) Option {
	return func(c *Cache) { c.channelPolicy = p }
}

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
