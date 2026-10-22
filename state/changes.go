package state

import (
	"slices"
	"sync"

	"github.com/invaliduser231/fluxer-go"
)

type Change struct {
	Event fluxer.Event
	Old   any
}

type listeners struct {
	mu   sync.RWMutex
	next int
	fns  []listener
}

type listener struct {
	id int
	fn func(Change)
}

func (c *Cache) OnChange(fn func(Change)) func() {
	l := &c.listeners
	l.mu.Lock()
	defer l.mu.Unlock()
	l.next++
	id := l.next
	l.fns = append(l.fns, listener{id: id, fn: fn})
	return func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.fns = slices.DeleteFunc(l.fns, func(x listener) bool { return x.id == id })
	}
}

func OnChange[E any, P interface {
	*E
	fluxer.Event
}, O any](c *Cache, fn func(old O, e P)) func() {
	return c.OnChange(func(ch Change) {
		e, ok := ch.Event.(P)
		if !ok {
			return
		}
		old, _ := ch.Old.(O)
		fn(old, e)
	})
}

func (c *Cache) notify(ch Change) {
	l := &c.listeners
	l.mu.RLock()
	fns := slices.Clone(l.fns)
	l.mu.RUnlock()
	for _, x := range fns {
		x.fn(ch)
	}
}

func (c *Cache) hasListeners() bool {
	l := &c.listeners
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.fns) > 0
}

func ptr[T any](v T, ok bool) *T {
	if !ok {
		return nil
	}
	return &v
}

func (c *Cache) before(e fluxer.Event) any {
	s := c.store
	switch e := e.(type) {
	case *fluxer.GuildUpdate:
		g, ok := c.Guild(e.ID)
		return ptr(g, ok)
	case *fluxer.GuildDelete:
		g, ok := c.Guild(e.ID)
		return ptr(g, ok)
	case *fluxer.ChannelUpdate:
		return ptr(c.channel(e.ID))
	case *fluxer.ChannelDelete:
		return ptr(c.channel(e.ID))
	case *fluxer.ThreadUpdate:
		return ptr(c.channel(e.ID))
	case *fluxer.ThreadDelete:
		return ptr(c.channel(e.ID))
	case *fluxer.GuildRoleUpdate:
		return ptr(s.Roles().Get(e.GuildID, e.Role.ID))
	case *fluxer.GuildRoleDelete:
		return ptr(s.Roles().Get(e.GuildID, e.RoleID))
	case *fluxer.GuildMemberUpdate:
		if e.User == nil {
			return (*fluxer.Member)(nil)
		}
		return ptr(c.Member(e.GuildID, e.User.ID))
	case *fluxer.GuildMemberRemove:
		return ptr(c.Member(e.GuildID, e.User.ID))
	case *fluxer.MessageUpdate:
		return ptr(s.Messages().Get(e.ChannelID, e.ID))
	case *fluxer.MessageDelete:
		return ptr(s.Messages().Get(e.ChannelID, e.ID))
	case *fluxer.MessageDeleteBulk:
		var out []fluxer.Message
		for _, id := range e.IDs {
			if m, ok := s.Messages().Get(e.ChannelID, id); ok {
				out = append(out, m)
			}
		}
		return out
	case *fluxer.MessageReactionAdd:
		return ptr(s.Messages().Get(e.ChannelID, e.MessageID))
	case *fluxer.MessageReactionRemove:
		return ptr(s.Messages().Get(e.ChannelID, e.MessageID))
	case *fluxer.MessageReactionRemoveAll:
		return ptr(s.Messages().Get(e.ChannelID, e.MessageID))
	case *fluxer.MessageReactionRemoveEmoji:
		return ptr(s.Messages().Get(e.ChannelID, e.MessageID))
	case *fluxer.VoiceStateUpdate:
		return ptr(s.VoiceStates().Get(e.GuildID, voiceKey(e.VoiceState)))
	case *fluxer.PresenceUpdate:
		return ptr(s.Presences().Get(e.GuildID, e.User.ID))
	case *fluxer.GuildEmojisUpdate:
		return sortedByID(s.Emojis().Group(e.GuildID), func(x fluxer.Emoji) string { return x.ID })
	case *fluxer.GuildStickersUpdate:
		return sortedByID(s.Stickers().Group(e.GuildID), func(x fluxer.Sticker) string { return x.ID })
	case *fluxer.UserUpdate:
		return ptr(s.Users().Get("", e.ID))
	}
	return nil
}
