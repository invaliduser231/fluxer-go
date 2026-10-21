package state

import (
	"encoding/json"
	"maps"
	"slices"
	"sync"

	"github.com/invaliduser231/fluxer-go"
)

type Cache struct {
	SkipMembers bool
	OnError     func(event string, err error)

	mu       sync.RWMutex
	self     fluxer.User
	guilds   map[string]*guild
	channels map[string]fluxer.Channel
}

type guild struct {
	info        fluxer.Guild
	unavailable bool
	roles       map[string]fluxer.Role
	emojis      []fluxer.Emoji
	stickers    []fluxer.Sticker
	channels    map[string]bool
	members     map[string]fluxer.Member
	voice       map[string]fluxer.VoiceState
}

func New() *Cache {
	return &Cache{guilds: map[string]*guild{}, channels: map[string]fluxer.Channel{}}
}

func (c *Cache) Handle(event string, data json.RawMessage) {
	e, err := fluxer.Decode(event, data)
	if err != nil {
		if c.OnError != nil {
			c.OnError(event, err)
		}
		return
	}
	if e == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.apply(e)
}

func (c *Cache) apply(e fluxer.Event) {
	switch e := e.(type) {
	case *fluxer.Ready:
		c.self = e.User
		for _, ch := range e.PrivateChannels {
			c.channels[ch.ID] = ch
		}
		for _, g := range e.Guilds {
			c.guild(g.ID).unavailable = g.Unavailable
		}
	case *fluxer.UserUpdate:
		if e.ID == c.self.ID {
			c.self = e.User
		}
	case *fluxer.GuildCreate:
		c.putGuild(e.GuildState)
	case *fluxer.GuildSync:
		c.putGuild(e.GuildState)
	case *fluxer.GuildUpdate:
		g := c.guild(e.ID)
		info := e.Guild
		info.Roles, info.Channels, info.Emojis, info.Stickers = nil, nil, nil, nil
		if info.MemberCount == 0 {
			info.MemberCount = g.info.MemberCount
		}
		g.info = info
	case *fluxer.GuildDelete:
		if e.Unavailable {
			c.guild(e.ID).unavailable = true
			return
		}
		if g := c.guilds[e.ID]; g != nil {
			for id := range g.channels {
				delete(c.channels, id)
			}
		}
		delete(c.guilds, e.ID)
	case *fluxer.GuildRoleCreate:
		c.guild(e.GuildID).roles[e.Role.ID] = e.Role
	case *fluxer.GuildRoleUpdate:
		c.guild(e.GuildID).roles[e.Role.ID] = e.Role
	case *fluxer.GuildRoleUpdateBulk:
		g := c.guild(e.GuildID)
		for _, r := range e.Roles {
			g.roles[r.ID] = r
		}
	case *fluxer.GuildRoleDelete:
		delete(c.guild(e.GuildID).roles, e.RoleID)
	case *fluxer.GuildEmojisUpdate:
		c.guild(e.GuildID).emojis = e.Emojis
	case *fluxer.GuildStickersUpdate:
		c.guild(e.GuildID).stickers = e.Stickers
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
		c.deleteChannel(e.ID)
		for id, ch := range c.channels {
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
		for id, ch := range c.channels {
			if ch.GuildID == e.GuildID && ch.IsThread() && (len(e.ChannelIDs) == 0 || slices.Contains(e.ChannelIDs, ch.ParentID)) {
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
		if ch, ok := c.channels[e.ChannelID]; ok {
			ch.LastMessageID = e.ID
			c.channels[ch.ID] = ch
		}
	case *fluxer.PassiveUpdates:
		for id, last := range e.Channels {
			if ch, ok := c.channels[id]; ok {
				ch.LastMessageID = last
				c.channels[id] = ch
			}
		}
		for _, v := range e.VoiceStates {
			c.putVoice(e.GuildID, v)
		}
	case *fluxer.GuildMemberAdd:
		c.putMember(e.GuildID, e.Member)
		c.guild(e.GuildID).info.MemberCount++
	case *fluxer.GuildMemberUpdate:
		c.putMember(e.GuildID, e.Member)
	case *fluxer.GuildMemberRemove:
		g := c.guild(e.GuildID)
		delete(g.members, e.User.ID)
		g.info.MemberCount = max(g.info.MemberCount-1, 0)
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

func (c *Cache) guild(id string) *guild {
	g := c.guilds[id]
	if g == nil {
		g = &guild{
			info:     fluxer.Guild{ID: id},
			roles:    map[string]fluxer.Role{},
			channels: map[string]bool{},
			members:  map[string]fluxer.Member{},
			voice:    map[string]fluxer.VoiceState{},
		}
		c.guilds[id] = g
	}
	return g
}

func (c *Cache) putGuild(s fluxer.GuildState) {
	g := c.guild(s.ID)
	if s.Unavailable {
		g.unavailable = true
		return
	}
	info := s.Properties
	info.ID = s.ID
	info.MemberCount, info.OnlineCount = s.MemberCount, s.OnlineCount
	g.info, g.unavailable = info, false
	g.roles = map[string]fluxer.Role{}
	for _, r := range s.Roles {
		g.roles[r.ID] = r
	}
	g.emojis, g.stickers = s.Emojis, s.Stickers
	for id := range g.channels {
		delete(c.channels, id)
	}
	g.channels = map[string]bool{}
	for _, ch := range append(slices.Clone(s.Channels), s.Threads...) {
		ch.GuildID = s.ID
		c.putChannel(ch)
	}
	for _, m := range s.Members {
		c.putMember(s.ID, m)
	}
	g.voice = map[string]fluxer.VoiceState{}
	for _, v := range s.VoiceStates {
		c.putVoice(s.ID, v)
	}
}

func (c *Cache) putChannel(ch fluxer.Channel) {
	c.channels[ch.ID] = ch
	if ch.GuildID != "" {
		c.guild(ch.GuildID).channels[ch.ID] = true
	}
}

func (c *Cache) deleteChannel(id string) {
	if ch, ok := c.channels[id]; ok && ch.GuildID != "" {
		if g := c.guilds[ch.GuildID]; g != nil {
			delete(g.channels, id)
		}
	}
	delete(c.channels, id)
}

func (c *Cache) putMember(guildID string, m fluxer.Member) {
	if m.User == nil || guildID == "" {
		return
	}
	if c.SkipMembers && m.User.ID != c.self.ID {
		return
	}
	g := c.guild(guildID)
	if old, ok := g.members[m.User.ID]; ok && old.User != nil && m.User.Username == "" {
		m.User = old.User
	}
	m.GuildID = guildID
	g.members[m.User.ID] = m
}

func (c *Cache) putVoice(guildID string, v fluxer.VoiceState) {
	if guildID == "" {
		guildID = v.GuildID
	}
	if guildID == "" {
		return
	}
	g := c.guild(guildID)
	key := v.ConnectionID
	if key == "" {
		key = v.UserID
	}
	if v.ChannelID == "" {
		delete(g.voice, key)
		return
	}
	if old, ok := g.voice[key]; ok && old.Version > v.Version {
		return
	}
	v.GuildID, v.Member = guildID, nil
	g.voice[key] = v
}

func (c *Cache) Self() fluxer.User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.self
}

func (c *Cache) Guild(id string) (fluxer.Guild, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	g, ok := c.guilds[id]
	if !ok || g.unavailable {
		return fluxer.Guild{}, false
	}
	return c.assemble(g), true
}

func (c *Cache) Guilds() []fluxer.Guild {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]fluxer.Guild, 0, len(c.guilds))
	for _, g := range c.guilds {
		if !g.unavailable {
			out = append(out, c.assemble(g))
		}
	}
	slices.SortFunc(out, func(a, b fluxer.Guild) int { return fluxer.CompareSnowflakes(a.ID, b.ID) })
	return out
}

func (c *Cache) Unavailable(id string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	g, ok := c.guilds[id]
	return ok && g.unavailable
}

func (c *Cache) assemble(g *guild) fluxer.Guild {
	out := g.info
	out.Roles = slices.SortedFunc(maps.Values(g.roles), func(a, b fluxer.Role) int { return a.Position - b.Position })
	out.Emojis, out.Stickers = slices.Clone(g.emojis), slices.Clone(g.stickers)
	out.Channels = c.guildChannels(g)
	return out
}

func (c *Cache) guildChannels(g *guild) []fluxer.Channel {
	out := make([]fluxer.Channel, 0, len(g.channels))
	for id := range g.channels {
		if ch, ok := c.channels[id]; ok && !ch.IsThread() {
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

func (c *Cache) Channel(id string) (fluxer.Channel, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ch, ok := c.channels[id]
	return ch, ok
}

func (c *Cache) GuildChannels(guildID string) []fluxer.Channel {
	c.mu.RLock()
	defer c.mu.RUnlock()
	g, ok := c.guilds[guildID]
	if !ok {
		return nil
	}
	return c.guildChannels(g)
}

func (c *Cache) Threads(parentID string) []fluxer.Channel {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []fluxer.Channel
	for _, ch := range c.channels {
		if ch.IsThread() && ch.ParentID == parentID {
			out = append(out, ch)
		}
	}
	slices.SortFunc(out, func(a, b fluxer.Channel) int { return fluxer.CompareSnowflakes(a.ID, b.ID) })
	return out
}

func (c *Cache) Role(guildID, roleID string) (fluxer.Role, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	g, ok := c.guilds[guildID]
	if !ok {
		return fluxer.Role{}, false
	}
	r, ok := g.roles[roleID]
	return r, ok
}

func (c *Cache) Member(guildID, userID string) (fluxer.Member, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	g, ok := c.guilds[guildID]
	if !ok {
		return fluxer.Member{}, false
	}
	m, ok := g.members[userID]
	return m, ok
}

func (c *Cache) Members(guildID string) []fluxer.Member {
	c.mu.RLock()
	defer c.mu.RUnlock()
	g, ok := c.guilds[guildID]
	if !ok {
		return nil
	}
	out := slices.Collect(maps.Values(g.members))
	slices.SortFunc(out, func(a, b fluxer.Member) int { return fluxer.CompareSnowflakes(a.User.ID, b.User.ID) })
	return out
}

func (c *Cache) VoiceStates(guildID string) []fluxer.VoiceState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	g, ok := c.guilds[guildID]
	if !ok {
		return nil
	}
	out := slices.Collect(maps.Values(g.voice))
	slices.SortFunc(out, func(a, b fluxer.VoiceState) int { return fluxer.CompareSnowflakes(a.UserID, b.UserID) })
	return out
}

func (c *Cache) VoiceChannel(guildID, userID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	g, ok := c.guilds[guildID]
	if !ok {
		return ""
	}
	for _, v := range g.voice {
		if v.UserID == userID {
			return v.ChannelID
		}
	}
	return ""
}

func (c *Cache) Permissions(guildID, channelID, userID string) (fluxer.Permissions, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	g, ok := c.guilds[guildID]
	if !ok {
		return 0, false
	}
	m, ok := g.members[userID]
	if !ok {
		return 0, false
	}
	info := g.info
	info.Roles = slices.Collect(maps.Values(g.roles))
	if channelID == "" {
		return fluxer.BasePermissions(info, userID, m.Roles), true
	}
	ch, ok := c.channels[channelID]
	if ok && ch.IsThread() {
		ch, ok = c.channels[ch.ParentID]
	}
	if !ok {
		return 0, false
	}
	return fluxer.ChannelPermissions(info, userID, m.Roles, ch), true
}
