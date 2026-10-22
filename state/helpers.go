package state

import (
	"iter"
	"strings"

	"github.com/invaliduser231/fluxer-go"
)

func outranks(a, b fluxer.Role) bool {
	if a.Position != b.Position {
		return a.Position > b.Position
	}
	return fluxer.CompareSnowflakes(a.ID, b.ID) < 0
}

func (c *Cache) HighestRole(guildID, userID string) (fluxer.Role, bool) {
	m, ok := c.Member(guildID, userID)
	if !ok {
		return fluxer.Role{}, false
	}
	var top fluxer.Role
	found := false
	for _, id := range m.Roles {
		r, ok := c.Role(guildID, id)
		if ok && (!found || outranks(r, top)) {
			top, found = r, true
		}
	}
	return top, found
}

func (c *Cache) owner(guildID string) string {
	rec, _ := c.store.Guilds().Get("", guildID)
	return rec.Guild.OwnerID
}

func (c *Cache) CanManageRole(guildID, actorID, roleID string) bool {
	if actorID == c.owner(guildID) {
		return true
	}
	role, ok := c.Role(guildID, roleID)
	if !ok {
		return false
	}
	top, ok := c.HighestRole(guildID, actorID)
	return ok && outranks(top, role)
}

func (c *Cache) CanManageMember(guildID, actorID, targetID string) bool {
	owner := c.owner(guildID)
	if targetID == owner || actorID == targetID {
		return false
	}
	if actorID == owner {
		return true
	}
	actor, ok := c.HighestRole(guildID, actorID)
	if !ok {
		return false
	}
	target, ok := c.HighestRole(guildID, targetID)
	return !ok || outranks(actor, target)
}

func (c *Cache) SelfMember(guildID string) (fluxer.Member, bool) {
	return c.Member(guildID, c.Self().ID)
}

func (c *Cache) VoiceMembers(guildID, channelID string) []fluxer.Member {
	var out []fluxer.Member
	seen := map[string]bool{}
	for _, v := range c.VoiceStates(guildID) {
		if v.ChannelID != channelID || seen[v.UserID] {
			continue
		}
		seen[v.UserID] = true
		if m, ok := c.Member(guildID, v.UserID); ok {
			out = append(out, m)
		}
	}
	return out
}

func (c *Cache) FindMember(guildID, name string) (fluxer.Member, bool) {
	name = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(name)), "@")
	var partial fluxer.Member
	found := false
	for _, m := range c.Members(guildID) {
		if m.User == nil {
			continue
		}
		candidates := []string{strings.ToLower(m.User.Username), strings.ToLower(m.User.Tag()), strings.ToLower(m.DisplayName())}
		for _, cand := range candidates {
			if cand == name {
				return m, true
			}
			if !found && cand != "" && strings.HasPrefix(cand, name) {
				partial, found = m, true
			}
		}
	}
	return partial, found
}

func (c *Cache) RoleByName(guildID, name string) (fluxer.Role, bool) {
	for _, r := range c.roles(guildID) {
		if strings.EqualFold(r.Name, name) {
			return r, true
		}
	}
	return fluxer.Role{}, false
}

func (c *Cache) ChannelByName(guildID, name string) (fluxer.Channel, bool) {
	name = strings.TrimPrefix(name, "#")
	for _, ch := range c.GuildChannels(guildID) {
		if strings.EqualFold(ch.Name, name) {
			return ch, true
		}
	}
	return fluxer.Channel{}, false
}

func (c *Cache) Emoji(guildID, emojiID string) (fluxer.Emoji, bool) {
	return c.store.Emojis().Get(guildID, emojiID)
}

func (c *Cache) EmojiByName(guildID, name string) (fluxer.Emoji, bool) {
	for _, e := range c.store.Emojis().Group(guildID) {
		if e.Name == name {
			return e, true
		}
	}
	return fluxer.Emoji{}, false
}

func (c *Cache) EachGuild() iter.Seq[fluxer.Guild] {
	return func(yield func(fluxer.Guild) bool) {
		for _, rec := range c.store.Guilds().Group("") {
			if rec.Unavailable {
				continue
			}
			if !yield(c.assemble(rec.Guild)) {
				return
			}
		}
	}
}

func (c *Cache) EachMember(guildID string) iter.Seq[fluxer.Member] {
	return func(yield func(fluxer.Member) bool) {
		for _, m := range c.store.Members().Group(guildID) {
			if !yield(c.complete(m)) {
				return
			}
		}
	}
}

func (c *Cache) EachMessage(channelID string) iter.Seq[fluxer.Message] {
	return func(yield func(fluxer.Message) bool) {
		for _, m := range c.store.Messages().Group(channelID) {
			if !yield(m) {
				return
			}
		}
	}
}
