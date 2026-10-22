package state

import (
	"encoding/json"
	"hash/maphash"
	"maps"
	"slices"
	"sync"

	"github.com/invaliduser231/fluxer-go"
)

const lockStripes = 64

type Cache struct {
	SkipMembers bool
	OnError     func(event string, err error)

	store Store
	locks [lockStripes]sync.Mutex

	selfMu sync.RWMutex
	self   fluxer.User
}

type Option func(*Cache)

func WithStore(s Store) Option { return func(c *Cache) { c.store = s } }

func New(opts ...Option) *Cache {
	c := &Cache{}
	for _, opt := range opts {
		opt(c)
	}
	if c.store == nil {
		c.store = NewMemoryStore()
	}
	return c
}

func (c *Cache) Store() Store { return c.store }

func (c *Cache) lock(guildID string) func() {
	mu := &c.locks[maphash.String(seed, guildID)%lockStripes]
	mu.Lock()
	return mu.Unlock
}

func (c *Cache) Handle(event string, data json.RawMessage) {
	e, err := fluxer.Decode(event, data)
	if err != nil {
		if c.OnError != nil {
			c.OnError(event, err)
		}
		return
	}
	if e != nil {
		c.Apply(e)
	}
}

func eventGuild(e fluxer.Event) string {
	switch e := e.(type) {
	case *fluxer.GuildCreate:
		return e.ID
	case *fluxer.GuildSync:
		return e.ID
	case *fluxer.GuildUpdate:
		return e.ID
	case *fluxer.GuildDelete:
		return e.ID
	case *fluxer.GuildRoleCreate:
		return e.GuildID
	case *fluxer.GuildRoleUpdate:
		return e.GuildID
	case *fluxer.GuildRoleUpdateBulk:
		return e.GuildID
	case *fluxer.GuildRoleDelete:
		return e.GuildID
	case *fluxer.GuildEmojisUpdate:
		return e.GuildID
	case *fluxer.GuildStickersUpdate:
		return e.GuildID
	case *fluxer.ChannelCreate:
		return e.GuildID
	case *fluxer.ChannelUpdate:
		return e.GuildID
	case *fluxer.ChannelUpdateBulk:
		return e.GuildID
	case *fluxer.ChannelDelete:
		return e.GuildID
	case *fluxer.ThreadCreate:
		return e.GuildID
	case *fluxer.ThreadUpdate:
		return e.GuildID
	case *fluxer.ThreadDelete:
		return e.GuildID
	case *fluxer.ThreadListSync:
		return e.GuildID
	case *fluxer.MessageCreate:
		return e.GuildID
	case *fluxer.PassiveUpdates:
		return e.GuildID
	case *fluxer.GuildMemberAdd:
		return e.GuildID
	case *fluxer.GuildMemberUpdate:
		return e.GuildID
	case *fluxer.GuildMemberRemove:
		return e.GuildID
	case *fluxer.GuildMembersChunk:
		return e.GuildID
	case *fluxer.VoiceStateUpdate:
		return e.GuildID
	}
	return ""
}

func (c *Cache) Apply(e fluxer.Event) {
	defer c.lock(eventGuild(e))()
	c.apply(e)
}

func (c *Cache) apply(e fluxer.Event) {
	s := c.store
	switch e := e.(type) {
	case *fluxer.Ready:
		c.selfMu.Lock()
		c.self = e.User
		c.selfMu.Unlock()
		for _, ch := range e.PrivateChannels {
			c.putChannel(ch)
		}
		for _, g := range e.Guilds {
			rec, _ := s.Guilds().Get("", g.ID)
			rec.Guild.ID, rec.Unavailable = g.ID, g.Unavailable
			s.Guilds().Put("", g.ID, rec)
		}
	case *fluxer.UserUpdate:
		c.selfMu.Lock()
		if e.ID == c.self.ID {
			c.self = e.User
		}
		c.selfMu.Unlock()
	case *fluxer.GuildCreate:
		c.putGuild(e.GuildState)
	case *fluxer.GuildSync:
		c.putGuild(e.GuildState)
	case *fluxer.GuildUpdate:
		rec := c.record(e.ID)
		info := e.Guild
		info.Roles, info.Channels, info.Emojis, info.Stickers = nil, nil, nil, nil
		if info.MemberCount == 0 {
			info.MemberCount = rec.Guild.MemberCount
		}
		if info.OnlineCount == 0 {
			info.OnlineCount = rec.Guild.OnlineCount
		}
		rec.Guild = info
		s.Guilds().Put("", e.ID, rec)
	case *fluxer.GuildDelete:
		if e.Unavailable {
			rec := c.record(e.ID)
			rec.Unavailable = true
			s.Guilds().Put("", e.ID, rec)
			return
		}
		c.dropGuild(e.ID)
	case *fluxer.GuildRoleCreate:
		s.Roles().Put(e.GuildID, e.Role.ID, e.Role)
	case *fluxer.GuildRoleUpdate:
		s.Roles().Put(e.GuildID, e.Role.ID, e.Role)
	case *fluxer.GuildRoleUpdateBulk:
		for _, r := range e.Roles {
			s.Roles().Put(e.GuildID, r.ID, r)
		}
	case *fluxer.GuildRoleDelete:
		s.Roles().Delete(e.GuildID, e.RoleID)
	case *fluxer.GuildEmojisUpdate:
		s.Emojis().DeleteGroup(e.GuildID)
		for _, em := range e.Emojis {
			s.Emojis().Put(e.GuildID, em.ID, em)
		}
	case *fluxer.GuildStickersUpdate:
		s.Stickers().DeleteGroup(e.GuildID)
		for _, st := range e.Stickers {
			s.Stickers().Put(e.GuildID, st.ID, st)
		}
	case *fluxer.ChannelCreate:
		c.putChannel(e.Channel)
	case *fluxer.ChannelUpdate:
		c.putChannel(e.Channel)
	case *fluxer.ChannelUpdateBulk:
		for _, ch := range e.Channels {
			if ch.GuildID == "" {
				ch.GuildID = e.GuildID
			}
			c.putChannel(ch)
		}
	case *fluxer.ChannelDelete:
		group, _ := s.Channels().Locate(e.ID)
		c.deleteChannel(e.ID)
		for id, ch := range s.Channels().Group(group) {
			if ch.ParentID == e.ID && ch.IsThread() {
				c.deleteChannel(id)
			}
		}
	case *fluxer.ThreadCreate:
		c.putChannel(e.Channel)
	case *fluxer.ThreadUpdate:
		c.putChannel(e.Channel)
	case *fluxer.ThreadDelete:
		c.deleteChannel(e.ID)
	case *fluxer.ThreadListSync:
		for id, ch := range s.Channels().Group(e.GuildID) {
			if ch.IsThread() && (len(e.ChannelIDs) == 0 || slices.Contains(e.ChannelIDs, ch.ParentID)) {
				c.deleteChannel(id)
			}
		}
		for _, ch := range e.Threads {
			if ch.GuildID == "" {
				ch.GuildID = e.GuildID
			}
			c.putChannel(ch)
		}
	case *fluxer.MessageCreate:
		if ch, ok := c.channel(e.ChannelID); ok {
			ch.LastMessageID = e.ID
			c.putChannel(ch)
		}
	case *fluxer.PassiveUpdates:
		for id, last := range e.Channels {
			if ch, ok := c.channel(id); ok {
				ch.LastMessageID = last
				c.putChannel(ch)
			}
		}
		for _, v := range e.VoiceStates {
			c.putVoice(e.GuildID, v)
		}
	case *fluxer.GuildMemberAdd:
		c.putMember(e.GuildID, e.Member)
		rec := c.record(e.GuildID)
		rec.Guild.MemberCount++
		s.Guilds().Put("", e.GuildID, rec)
	case *fluxer.GuildMemberUpdate:
		c.putMember(e.GuildID, e.Member)
	case *fluxer.GuildMemberRemove:
		s.Members().Delete(e.GuildID, e.User.ID)
		rec := c.record(e.GuildID)
		rec.Guild.MemberCount = max(rec.Guild.MemberCount-1, 0)
		s.Guilds().Put("", e.GuildID, rec)
	case *fluxer.GuildMembersChunk:
		for _, m := range e.Members {
			c.putMember(e.GuildID, m)
		}
	case *fluxer.VoiceStateUpdate:
		c.putVoice(e.GuildID, e.VoiceState)
		if e.Member != nil && e.GuildID != "" {
			c.putMember(e.GuildID, *e.Member)
		}
	}
}

func (c *Cache) record(id string) GuildRecord {
	rec, ok := c.store.Guilds().Get("", id)
	if !ok {
		rec.Guild.ID = id
	}
	return rec
}

func (c *Cache) dropGuild(id string) {
	s := c.store
	for chID := range s.Channels().Group(id) {
		s.Channels().Delete(id, chID)
	}
	s.Roles().DeleteGroup(id)
	s.Members().DeleteGroup(id)
	s.VoiceStates().DeleteGroup(id)
	s.Emojis().DeleteGroup(id)
	s.Stickers().DeleteGroup(id)
	s.Guilds().Delete("", id)
}

func (c *Cache) putGuild(gs fluxer.GuildState) {
	s := c.store
	rec := c.record(gs.ID)
	if gs.Unavailable {
		rec.Unavailable = true
		s.Guilds().Put("", gs.ID, rec)
		return
	}
	info := gs.Properties
	info.ID = gs.ID
	info.MemberCount, info.OnlineCount = gs.MemberCount, gs.OnlineCount
	info.Roles, info.Channels, info.Emojis, info.Stickers = nil, nil, nil, nil
	rec.Guild, rec.Unavailable = info, false
	if gs.JoinedAt != nil {
		rec.JoinedAt = gs.JoinedAt
	}
	s.Guilds().Put("", gs.ID, rec)
	s.Roles().DeleteGroup(gs.ID)
	for _, r := range gs.Roles {
		s.Roles().Put(gs.ID, r.ID, r)
	}
	s.Emojis().DeleteGroup(gs.ID)
	for _, em := range gs.Emojis {
		s.Emojis().Put(gs.ID, em.ID, em)
	}
	s.Stickers().DeleteGroup(gs.ID)
	for _, st := range gs.Stickers {
		s.Stickers().Put(gs.ID, st.ID, st)
	}
	for chID := range s.Channels().Group(gs.ID) {
		s.Channels().Delete(gs.ID, chID)
	}
	for _, ch := range append(slices.Clone(gs.Channels), gs.Threads...) {
		ch.GuildID = gs.ID
		c.putChannel(ch)
	}
	for _, m := range gs.Members {
		c.putMember(gs.ID, m)
	}
	s.VoiceStates().DeleteGroup(gs.ID)
	for _, v := range gs.VoiceStates {
		c.putVoice(gs.ID, v)
	}
}

func (c *Cache) channel(id string) (fluxer.Channel, bool) {
	group, ok := c.store.Channels().Locate(id)
	if !ok {
		return fluxer.Channel{}, false
	}
	return c.store.Channels().Get(group, id)
}

func (c *Cache) putChannel(ch fluxer.Channel) {
	if old, ok := c.store.Channels().Locate(ch.ID); ok && old != ch.GuildID {
		c.store.Channels().Delete(old, ch.ID)
	}
	c.store.Channels().Put(ch.GuildID, ch.ID, ch)
}

func (c *Cache) deleteChannel(id string) {
	if group, ok := c.store.Channels().Locate(id); ok {
		c.store.Channels().Delete(group, id)
	}
}

func (c *Cache) putMember(guildID string, m fluxer.Member) {
	if m.User == nil || guildID == "" {
		return
	}
	if c.SkipMembers && m.User.ID != c.Self().ID {
		return
	}
	if old, ok := c.store.Members().Get(guildID, m.User.ID); ok && old.User != nil && m.User.Username == "" {
		m.User = old.User
	}
	m.GuildID = guildID
	c.store.Members().Put(guildID, m.User.ID, m)
}

func voiceKey(v fluxer.VoiceState) string {
	if v.ConnectionID != "" {
		return v.ConnectionID
	}
	return v.UserID
}

func (c *Cache) putVoice(guildID string, v fluxer.VoiceState) {
	if guildID == "" {
		guildID = v.GuildID
	}
	if guildID == "" {
		return
	}
	key := voiceKey(v)
	if v.ChannelID == "" {
		c.store.VoiceStates().Delete(guildID, key)
		return
	}
	if old, ok := c.store.VoiceStates().Get(guildID, key); ok && old.Version > v.Version {
		return
	}
	v.GuildID, v.Member = guildID, nil
	c.store.VoiceStates().Put(guildID, key, v)
}

func (c *Cache) Self() fluxer.User {
	c.selfMu.RLock()
	defer c.selfMu.RUnlock()
	return c.self
}

func (c *Cache) Guild(id string) (fluxer.Guild, bool) {
	rec, ok := c.store.Guilds().Get("", id)
	if !ok || rec.Unavailable {
		return fluxer.Guild{}, false
	}
	return c.assemble(rec.Guild), true
}

func (c *Cache) Guilds() []fluxer.Guild {
	var out []fluxer.Guild
	for _, rec := range c.store.Guilds().Group("") {
		if !rec.Unavailable {
			out = append(out, c.assemble(rec.Guild))
		}
	}
	slices.SortFunc(out, func(a, b fluxer.Guild) int { return fluxer.CompareSnowflakes(a.ID, b.ID) })
	return out
}

func (c *Cache) Unavailable(id string) bool {
	rec, ok := c.store.Guilds().Get("", id)
	return ok && rec.Unavailable
}

func (c *Cache) assemble(g fluxer.Guild) fluxer.Guild {
	g.Roles = c.roles(g.ID)
	g.Emojis = sortedByID(c.store.Emojis().Group(g.ID), func(e fluxer.Emoji) string { return e.ID })
	g.Stickers = sortedByID(c.store.Stickers().Group(g.ID), func(s fluxer.Sticker) string { return s.ID })
	g.Channels = c.GuildChannels(g.ID)
	return g
}

func sortedByID[V any](seq func(func(string, V) bool), id func(V) string) []V {
	var out []V
	for _, v := range seq {
		out = append(out, v)
	}
	slices.SortFunc(out, func(a, b V) int { return fluxer.CompareSnowflakes(id(a), id(b)) })
	return out
}

func (c *Cache) roles(guildID string) []fluxer.Role {
	out := slices.Collect(func(yield func(fluxer.Role) bool) {
		for _, r := range c.store.Roles().Group(guildID) {
			if !yield(r) {
				return
			}
		}
	})
	slices.SortFunc(out, func(a, b fluxer.Role) int {
		if a.Position != b.Position {
			return a.Position - b.Position
		}
		return fluxer.CompareSnowflakes(a.ID, b.ID)
	})
	return out
}

func (c *Cache) Channel(id string) (fluxer.Channel, bool) { return c.channel(id) }

func (c *Cache) GuildChannels(guildID string) []fluxer.Channel {
	var out []fluxer.Channel
	for _, ch := range c.store.Channels().Group(guildID) {
		if !ch.IsThread() {
			out = append(out, ch)
		}
	}
	slices.SortFunc(out, func(a, b fluxer.Channel) int {
		if a.Position != b.Position {
			return a.Position - b.Position
		}
		return fluxer.CompareSnowflakes(a.ID, b.ID)
	})
	return out
}

func (c *Cache) Threads(parentID string) []fluxer.Channel {
	parent, ok := c.channel(parentID)
	if !ok {
		return nil
	}
	var out []fluxer.Channel
	for _, ch := range c.store.Channels().Group(parent.GuildID) {
		if ch.IsThread() && ch.ParentID == parentID {
			out = append(out, ch)
		}
	}
	slices.SortFunc(out, func(a, b fluxer.Channel) int { return fluxer.CompareSnowflakes(a.ID, b.ID) })
	return out
}

func (c *Cache) Role(guildID, roleID string) (fluxer.Role, bool) {
	return c.store.Roles().Get(guildID, roleID)
}

func (c *Cache) Member(guildID, userID string) (fluxer.Member, bool) {
	return c.store.Members().Get(guildID, userID)
}

func (c *Cache) Members(guildID string) []fluxer.Member {
	out := slices.Collect(maps.Values(maps.Collect(c.store.Members().Group(guildID))))
	slices.SortFunc(out, func(a, b fluxer.Member) int { return fluxer.CompareSnowflakes(a.User.ID, b.User.ID) })
	return out
}

func (c *Cache) VoiceStates(guildID string) []fluxer.VoiceState {
	out := slices.Collect(maps.Values(maps.Collect(c.store.VoiceStates().Group(guildID))))
	slices.SortFunc(out, func(a, b fluxer.VoiceState) int { return fluxer.CompareSnowflakes(a.UserID, b.UserID) })
	return out
}

func (c *Cache) VoiceChannel(guildID, userID string) string {
	for _, v := range c.store.VoiceStates().Group(guildID) {
		if v.UserID == userID {
			return v.ChannelID
		}
	}
	return ""
}

func (c *Cache) Permissions(guildID, channelID, userID string) (fluxer.Permissions, bool) {
	rec, ok := c.store.Guilds().Get("", guildID)
	if !ok {
		return 0, false
	}
	m, ok := c.Member(guildID, userID)
	if !ok {
		return 0, false
	}
	info := rec.Guild
	info.Roles = c.roles(guildID)
	if channelID == "" {
		return fluxer.BasePermissions(info, userID, m.Roles), true
	}
	ch, ok := c.channel(channelID)
	if ok && ch.IsThread() {
		ch, ok = c.channel(ch.ParentID)
	}
	if !ok {
		return 0, false
	}
	return fluxer.ChannelPermissions(info, userID, m.Roles, ch), true
}
