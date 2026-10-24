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

// HighestRole returns the member's highest cached role by position, preferring the
// lower ID on a tie. It returns false when none of the member's roles is cached.
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

// CanManageRole reports whether actorID ranks above roleID in the role hierarchy.
// The guild owner can manage every role. Permissions are not checked.
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

// CanManageMember reports whether actorID outranks targetID in the role hierarchy.
// Nobody can manage the owner or themselves, the owner can manage everyone else, and
// a target without cached roles can be managed by any actor that has one.
// Permissions are not checked.
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

// SelfMember returns the bot's own member in a guild.
func (c *Cache) SelfMember(guildID string) (fluxer.Member, bool) {
	return c.Member(guildID, c.Self().ID)
}

// VoiceMembers returns the cached members in a voice channel, sorted by user ID.
// Users in the channel without a cached member are left out.
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

// FindMember looks up a member by username, tag or display name, ignoring case,
// surrounding spaces and a leading @. An exact match wins; otherwise it returns the
// member with the lowest user ID whose name starts with name.
func (c *Cache) FindMember(guildID, name string) (fluxer.Member, bool) {
	name = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(name)), "@")
	if name == "" {
		return fluxer.Member{}, false
	}
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

// RoleByName returns the lowest role by position whose name matches, ignoring case.
func (c *Cache) RoleByName(guildID, name string) (fluxer.Role, bool) {
	for _, r := range c.roles(guildID) {
		if strings.EqualFold(r.Name, name) {
			return r, true
		}
	}
	return fluxer.Role{}, false
}

// ChannelByName returns the first non-thread channel by position whose name matches,
// ignoring case and a leading #.
func (c *Cache) ChannelByName(guildID, name string) (fluxer.Channel, bool) {
	name = strings.TrimPrefix(name, "#")
	for _, ch := range c.GuildChannels(guildID) {
		if strings.EqualFold(ch.Name, name) {
			return ch, true
		}
	}
	return fluxer.Channel{}, false
}

// Emoji returns a cached guild emoji.
func (c *Cache) Emoji(guildID, emojiID string) (fluxer.Emoji, bool) {
	return c.store.Emojis().Get(guildID, emojiID)
}

// EmojiByName returns a guild emoji with exactly that name. If several share the
// name, any one of them is returned.
func (c *Cache) EmojiByName(guildID, name string) (fluxer.Emoji, bool) {
	for _, e := range c.store.Emojis().Group(guildID) {
		if e.Name == name {
			return e, true
		}
	}
	return fluxer.Emoji{}, false
}

// EachGuild yields the available guilds in no particular order, assembled like
// Guild, without building a slice first.
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

// EachMember yields the guild's cached members in no particular order.
func (c *Cache) EachMember(guildID string) iter.Seq[fluxer.Member] {
	return func(yield func(fluxer.Member) bool) {
		for _, m := range c.store.Members().Group(guildID) {
			if !yield(c.complete(m)) {
				return
			}
		}
	}
}

// EachMessage yields a channel's cached messages in no particular order.
func (c *Cache) EachMessage(channelID string) iter.Seq[fluxer.Message] {
	return func(yield func(fluxer.Message) bool) {
		for _, m := range c.store.Messages().Group(channelID) {
			if !yield(m) {
				return
			}
		}
	}
}
