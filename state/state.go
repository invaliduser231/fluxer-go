package state

import (
	"encoding/json"
	"hash/maphash"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

const lockStripes = 64

// Cache keeps gateway state in a Store. Pass every dispatch to Handle or Apply; the
// read methods are safe to call from any goroutine.
type Cache struct {
	// SkipMembers stops the cache from storing members other than the bot's own member.
	SkipMembers bool
	// OnError receives events that Handle could not decode.
	OnError func(event string, err error)

	store         Store
	flags         Flags
	memberPolicy  Policy[fluxer.Member]
	channelPolicy Policy[fluxer.Channel]
	messagePolicy Policy[fluxer.Message]
	messageLimit  int
	messageTTL    time.Duration
	onEvict       func(fluxer.Message)
	locks         [lockStripes]sync.Mutex
	listeners     listeners
	ready         readyState
	counters      counters

	selfMu sync.RWMutex
	self   fluxer.User
}

// Option configures a Cache created by New.
type Option func(*Cache)

// WithStore sets the backing store. The default is a new MemoryStore.
func WithStore(s Store) Option { return func(c *Cache) { c.store = s } }

// New returns a cache that stores everything in FlagsAll, with opts applied in order.
func New(opts ...Option) *Cache {
	c := &Cache{flags: FlagsAll}
	for _, opt := range opts {
		opt(c)
	}
	if c.store == nil {
		c.store = NewMemoryStore()
	}
	return c
}

// Store returns the backing store.
func (c *Cache) Store() Store { return c.store }

func (c *Cache) lock(guildID string) func() {
	mu := &c.locks[maphash.String(seed, guildID)%lockStripes]
	mu.Lock()
	return mu.Unlock
}

// Handle decodes a raw gateway dispatch and applies it. Decode errors go to OnError
// and event names the library does not know are ignored.
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
	case *fluxer.ThreadMemberUpdate:
		return e.GuildID
	case *fluxer.ThreadMembersUpdate:
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
	case *fluxer.PresenceUpdate:
		return e.GuildID
	case *fluxer.PresenceUpdateBulk:
		return e.GuildID
	case *fluxer.MessageUpdate:
		return e.GuildID
	case *fluxer.MessageDelete:
		return e.GuildID
	case *fluxer.MessageDeleteBulk:
		return e.GuildID
	case *fluxer.MessageReactionAdd:
		return e.GuildID
	case *fluxer.MessageReactionAddMany:
		return e.GuildID
	case *fluxer.MessageReactionRemove:
		return e.GuildID
	case *fluxer.MessageReactionRemoveAll:
		return e.GuildID
	case *fluxer.MessageReactionRemoveEmoji:
		return e.GuildID
	case *fluxer.ChannelPinsUpdate:
		return e.GuildID
	}
	return ""
}

// Apply updates the cache from a decoded event and then runs the OnChange handlers.
// Events for the same guild are applied one at a time.
func (c *Cache) Apply(e fluxer.Event) {
	unlock := c.lock(eventGuild(e))
	var old any
	notify := c.hasListeners()
	if notify {
		old = c.before(e)
	}
	c.apply(e)
	unlock()
	if notify {
		c.notify(Change{Event: e, Old: old})
	}
}

func (c *Cache) apply(e fluxer.Event) {
	s := c.store
	c.applyMessage(e)
	switch e := e.(type) {
	case *fluxer.Ready:
		c.selfMu.Lock()
		c.self = e.User
		c.selfMu.Unlock()
		c.putUser(e.User)
		for _, ch := range e.PrivateChannels {
			c.putChannel(ch)
		}
		c.ready.start(e.Guilds)
		for _, g := range e.Guilds {
			rec, _ := s.Guilds().Get("", g.ID)
			rec.Guild.ID, rec.Unavailable = g.ID, g.Unavailable
			c.putRecord(g.ID, rec)
		}
	case *fluxer.UserUpdate:
		c.selfMu.Lock()
		if e.ID == c.self.ID {
			c.self = e.User
		}
		c.selfMu.Unlock()
		c.putUser(e.User)
	case *fluxer.GuildCreate:
		c.putGuild(e.GuildState)
		c.ready.resolve(e.ID)
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
		c.putRecord(e.ID, rec)
	case *fluxer.GuildDelete:
		c.ready.resolve(e.ID)
		if e.Unavailable {
			rec := c.record(e.ID)
			rec.Unavailable = true
			c.putRecord(e.ID, rec)
			return
		}
		c.dropGuild(e.ID)
	case *fluxer.GuildRoleCreate:
		c.putRole(e.GuildID, e.Role)
	case *fluxer.GuildRoleUpdate:
		c.putRole(e.GuildID, e.Role)
	case *fluxer.GuildRoleUpdateBulk:
		for _, r := range e.Roles {
			c.putRole(e.GuildID, r)
		}
	case *fluxer.GuildRoleDelete:
		s.Roles().Delete(e.GuildID, e.RoleID)
	case *fluxer.GuildEmojisUpdate:
		c.putEmojis(e.GuildID, e.Emojis)
	case *fluxer.GuildStickersUpdate:
		c.putStickers(e.GuildID, e.Stickers)
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
		s.Messages().DeleteGroup(e.ID)
		for id, ch := range s.Channels().Group(group) {
			if ch.ParentID == e.ID && ch.IsThread() {
				c.deleteChannel(id)
				s.Messages().DeleteGroup(id)
			}
		}
	case *fluxer.ThreadCreate:
		c.putChannel(e.Channel)
	case *fluxer.ThreadUpdate:
		c.putChannel(e.Channel)
	case *fluxer.ThreadDelete:
		c.deleteChannel(e.ID)
		s.Messages().DeleteGroup(e.ID)
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
	case *fluxer.ThreadMemberUpdate:
		if ch, ok := c.channel(e.ID); ok {
			tm := e.ThreadMember
			ch.Member = &tm
			c.putChannel(ch)
		}
	case *fluxer.ThreadMembersUpdate:
		self := c.Self().ID
		for _, tm := range e.AddedMembers {
			if tm.Member != nil && e.GuildID != "" {
				c.putMember(e.GuildID, *tm.Member)
			}
		}
		if ch, ok := c.channel(e.ID); ok {
			ch.MemberCount = e.MemberCount
			for _, tm := range e.AddedMembers {
				if tm.UserID == self {
					tm.Member = nil
					ch.Member = &tm
				}
			}
			if slices.Contains(e.RemovedMemberIDs, self) {
				ch.Member = nil
			}
			c.putChannel(ch)
		}
	case *fluxer.MessageCreate:
		c.putUser(e.Author)
		if e.Member != nil && e.GuildID != "" {
			m := *e.Member
			m.User = &e.Author
			c.putMember(e.GuildID, m)
		}
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
		if rec, ok := s.Guilds().Get("", e.GuildID); ok {
			rec.Guild.MemberCount++
			c.putRecord(e.GuildID, rec)
		}
	case *fluxer.GuildMemberUpdate:
		c.putMember(e.GuildID, e.Member)
	case *fluxer.GuildMemberRemove:
		s.Members().Delete(e.GuildID, e.User.ID)
		if rec, ok := s.Guilds().Get("", e.GuildID); ok {
			rec.Guild.MemberCount = max(rec.Guild.MemberCount-1, 0)
			c.putRecord(e.GuildID, rec)
		}
	case *fluxer.GuildMembersChunk:
		for _, p := range e.Presences {
			c.putPresence(e.GuildID, p)
		}
		for _, m := range e.Members {
			c.putMember(e.GuildID, m)
		}
	case *fluxer.PresenceUpdate:
		c.putPresence(e.GuildID, e.Presence)
	case *fluxer.PresenceUpdateBulk:
		for _, p := range e.Presences {
			c.putPresence(e.GuildID, p)
		}
	case *fluxer.GuildCountsUpdate:
		for _, n := range e.Counts {
			if rec, ok := s.Guilds().Get("", n.GuildID); ok {
				rec.Guild.MemberCount, rec.Guild.OnlineCount = n.MemberCount, n.OnlineCount
				c.putRecord(n.GuildID, rec)
			}
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
		s.Messages().DeleteGroup(chID)
	}
	s.Roles().DeleteGroup(id)
	s.Members().DeleteGroup(id)
	s.VoiceStates().DeleteGroup(id)
	s.Emojis().DeleteGroup(id)
	s.Stickers().DeleteGroup(id)
	s.Presences().DeleteGroup(id)
	s.Guilds().Delete("", id)
}

func (c *Cache) putGuild(gs fluxer.GuildState) {
	s := c.store
	rec := c.record(gs.ID)
	if gs.Unavailable {
		rec.Unavailable = true
		c.putRecord(gs.ID, rec)
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
	c.putRecord(gs.ID, rec)
	s.Roles().DeleteGroup(gs.ID)
	for _, r := range gs.Roles {
		c.putRole(gs.ID, r)
	}
	c.putEmojis(gs.ID, gs.Emojis)
	c.putStickers(gs.ID, gs.Stickers)
	for chID := range s.Channels().Group(gs.ID) {
		s.Channels().Delete(gs.ID, chID)
	}
	for _, ch := range append(slices.Clone(gs.Channels), gs.Threads...) {
		ch.GuildID = gs.ID
		c.putChannel(ch)
	}
	s.VoiceStates().DeleteGroup(gs.ID)
	for _, v := range gs.VoiceStates {
		c.putVoice(gs.ID, v)
	}
	for _, m := range gs.Members {
		c.putMember(gs.ID, m)
	}
}

func (c *Cache) putRecord(id string, rec GuildRecord) {
	if c.flags.Has(FlagGuilds) {
		c.store.Guilds().Put("", id, rec)
	}
}

func (c *Cache) putRole(guildID string, r fluxer.Role) {
	if c.flags.Has(FlagRoles) {
		c.store.Roles().Put(guildID, r.ID, r)
	}
}

func (c *Cache) putEmojis(guildID string, emojis []fluxer.Emoji) {
	c.store.Emojis().DeleteGroup(guildID)
	if !c.flags.Has(FlagEmojis) {
		return
	}
	for _, em := range emojis {
		c.store.Emojis().Put(guildID, em.ID, em)
	}
}

func (c *Cache) putStickers(guildID string, stickers []fluxer.Sticker) {
	c.store.Stickers().DeleteGroup(guildID)
	if !c.flags.Has(FlagStickers) {
		return
	}
	for _, st := range stickers {
		c.store.Stickers().Put(guildID, st.ID, st)
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
	if !c.keepChannel(ch) {
		c.deleteChannel(ch.ID)
		return
	}
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
	c.putUser(*m.User)
	if old, ok := c.store.Members().Get(guildID, m.User.ID); ok && old.User != nil && m.User.Username == "" {
		m.User = old.User
	}
	m.GuildID = guildID
	if !c.keepMember(m) {
		c.store.Members().Delete(guildID, m.User.ID)
		return
	}
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
	if !c.flags.Has(FlagVoiceStates) {
		return
	}
	key := voiceKey(v)
	if v.ChannelID == "" {
		c.store.VoiceStates().Delete(guildID, key)
		if m, ok := c.store.Members().Get(guildID, v.UserID); ok && !c.keepMember(m) {
			c.store.Members().Delete(guildID, v.UserID)
		}
		return
	}
	if old, ok := c.store.VoiceStates().Get(guildID, key); ok && old.Version > v.Version {
		return
	}
	v.GuildID, v.Member = guildID, nil
	c.store.VoiceStates().Put(guildID, key, v)
}

// Self returns the bot user from the last READY, kept current by USER_UPDATE.
func (c *Cache) Self() fluxer.User {
	c.selfMu.RLock()
	defer c.selfMu.RUnlock()
	return c.self
}

// Guild returns a cached guild with its roles, emojis, stickers and non-thread
// channels filled in. Unavailable guilds are reported as missing.
func (c *Cache) Guild(id string) (fluxer.Guild, bool) {
	rec, ok := c.store.Guilds().Get("", id)
	if !ok || rec.Unavailable {
		return fluxer.Guild{}, false
	}
	return c.assemble(rec.Guild), true
}

// Guilds returns all available guilds sorted by ID, assembled like Guild.
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

// Unavailable reports whether the guild is known but currently unavailable.
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

// Channel returns a cached guild channel, thread or DM channel by ID.
func (c *Cache) Channel(id string) (fluxer.Channel, bool) { return c.channel(id) }

// GuildChannels returns the guild's channels without threads, sorted by position and
// then ID.
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

// Threads returns the cached threads of parentID sorted by ID. It returns nil when
// the parent channel is not cached.
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

// Role returns a cached role.
func (c *Cache) Role(guildID, roleID string) (fluxer.Role, bool) {
	return c.store.Roles().Get(guildID, roleID)
}

// Member returns a cached member. When the stored member carries only a user ID, the
// user is filled in from the user table.
func (c *Cache) Member(guildID, userID string) (fluxer.Member, bool) {
	m, ok := c.store.Members().Get(guildID, userID)
	if ok {
		m = c.complete(m)
	}
	return m, ok
}

func (c *Cache) complete(m fluxer.Member) fluxer.Member {
	if m.User != nil && m.User.Username == "" {
		if u, ok := c.store.Users().Get("", m.User.ID); ok {
			m.User = &u
		}
	}
	return m
}

// MemberPartial reports whether a member is cached without user details such as the
// username.
func (c *Cache) MemberPartial(guildID, userID string) bool {
	m, ok := c.Member(guildID, userID)
	return ok && (m.User == nil || m.User.Username == "")
}

// User returns a cached user. Users are collected from READY, members, presences and
// messages while FlagUsers is set.
func (c *Cache) User(id string) (fluxer.User, bool) { return c.store.Users().Get("", id) }

// Presence returns a user's presence in a guild. Offline users have no cached
// presence.
func (c *Cache) Presence(guildID, userID string) (fluxer.Presence, bool) {
	return c.store.Presences().Get(guildID, userID)
}

// Presences returns the guild's cached presences sorted by user ID.
func (c *Cache) Presences(guildID string) []fluxer.Presence {
	out := slices.Collect(maps.Values(maps.Collect(c.store.Presences().Group(guildID))))
	slices.SortFunc(out, func(a, b fluxer.Presence) int { return fluxer.CompareSnowflakes(a.User.ID, b.User.ID) })
	return out
}

func (c *Cache) putUser(u fluxer.User) {
	if u.ID == "" || u.Username == "" || !c.flags.Has(FlagUsers) {
		return
	}
	c.store.Users().Put("", u.ID, u)
}

func (c *Cache) putPresence(guildID string, p fluxer.Presence) {
	if guildID == "" {
		guildID = p.GuildID
	}
	if guildID == "" || p.User.ID == "" || !c.flags.Has(FlagPresences) {
		return
	}
	c.putUser(p.User)
	if p.Status == fluxer.StatusOffline || p.Status == "" {
		c.store.Presences().Delete(guildID, p.User.ID)
		return
	}
	p.GuildID = guildID
	c.store.Presences().Put(guildID, p.User.ID, p)
}

// Members returns the guild's cached members sorted by user ID.
func (c *Cache) Members(guildID string) []fluxer.Member {
	var out []fluxer.Member
	for _, m := range c.store.Members().Group(guildID) {
		out = append(out, c.complete(m))
	}
	slices.SortFunc(out, func(a, b fluxer.Member) int { return fluxer.CompareSnowflakes(a.User.ID, b.User.ID) })
	return out
}

// VoiceStates returns the guild's voice states sorted by user ID. A user connected
// from several sessions appears once per connection.
func (c *Cache) VoiceStates(guildID string) []fluxer.VoiceState {
	out := slices.Collect(maps.Values(maps.Collect(c.store.VoiceStates().Group(guildID))))
	slices.SortFunc(out, func(a, b fluxer.VoiceState) int { return fluxer.CompareSnowflakes(a.UserID, b.UserID) })
	return out
}

// VoiceChannel returns the voice channel a user is in, or an empty string. For a
// user with several connections in the guild it returns any one of them.
func (c *Cache) VoiceChannel(guildID, userID string) string {
	for _, v := range c.store.VoiceStates().Group(guildID) {
		if v.UserID == userID {
			return v.ChannelID
		}
	}
	return ""
}

// Permissions computes a member's permissions from cached data. An empty channelID
// gives guild-level permissions, and a thread uses its parent's overwrites. It
// returns false when the guild, member or channel is not cached.
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
