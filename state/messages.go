package state

import (
	"slices"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

// DefaultMessagesPerChannel is the number of messages kept per channel when
// WithMessages is not used.
const DefaultMessagesPerChannel = 100

// WithMessages sets how many messages are kept per channel and for how long. When a
// channel goes over perChannel, its oldest messages are evicted. A ttl of zero keeps
// messages until the limit evicts them; otherwise expired messages are removed when
// the channel gets a new message or by SweepMessages. A perChannel of zero or less
// disables the message cache.
func WithMessages(perChannel int, ttl time.Duration) Option {
	return func(c *Cache) {
		c.messageLimit, c.messageTTL = perChannel, ttl
		if perChannel <= 0 {
			c.flags &^= FlagMessages
		}
	}
}

// WithMessagePolicy limits which messages are cached. SweepMessages removes cached
// messages that no longer match.
func WithMessagePolicy(p Policy[fluxer.Message]) Option {
	return func(c *Cache) { c.messagePolicy = p }
}

// WithMessageEvict sets a function that is called for every message removed by the
// limit, the TTL or SweepMessages. Messages deleted by events are not reported.
func WithMessageEvict(fn func(fluxer.Message)) Option { return func(c *Cache) { c.onEvict = fn } }

// MessagesInGuilds keeps messages sent in the listed guilds.
func MessagesInGuilds(guildIDs ...string) Policy[fluxer.Message] {
	return func(_ *Cache, m fluxer.Message) bool { return slices.Contains(guildIDs, m.GuildID) }
}

// MessagesFromHumans keeps messages that were not sent by a bot or a webhook.
func MessagesFromHumans() Policy[fluxer.Message] {
	return func(_ *Cache, m fluxer.Message) bool { return !m.Author.Bot && m.WebhookID == "" }
}

func (c *Cache) expired(m fluxer.Message, now time.Time) bool {
	if c.messageTTL <= 0 {
		return false
	}
	at := m.Timestamp
	if at.IsZero() {
		at, _ = fluxer.SnowflakeTime(m.ID)
	}
	return now.Sub(at) > c.messageTTL
}

func (c *Cache) putMessage(m fluxer.Message) {
	if !c.flags.Has(FlagMessages) || m.ID == "" || m.ChannelID == "" {
		return
	}
	if c.messagePolicy != nil && !c.messagePolicy(c, m) {
		c.store.Messages().Delete(m.ChannelID, m.ID)
		return
	}
	if c.expired(m, time.Now()) {
		return
	}
	t := c.store.Messages()
	t.Put(m.ChannelID, m.ID, m)
	limit := c.messageLimit
	if limit <= 0 {
		limit = DefaultMessagesPerChannel
	}
	if t.Len(m.ChannelID) <= limit && c.messageTTL <= 0 {
		return
	}
	var all []fluxer.Message
	for _, v := range t.Group(m.ChannelID) {
		all = append(all, v)
	}
	slices.SortFunc(all, func(a, b fluxer.Message) int { return fluxer.CompareSnowflakes(a.ID, b.ID) })
	now := time.Now()
	for i, v := range all {
		if len(all)-i <= limit && !c.expired(v, now) {
			break
		}
		c.evict(v)
	}
}

func (c *Cache) evict(m fluxer.Message) {
	c.store.Messages().Delete(m.ChannelID, m.ID)
	if c.onEvict != nil {
		c.onEvict(m)
	}
}

func (c *Cache) editMessage(channelID, id string, fn func(m *fluxer.Message)) {
	m, ok := c.store.Messages().Get(channelID, id)
	if !ok {
		return
	}
	fn(&m)
	c.store.Messages().Put(channelID, id, m)
}

func sameEmoji(a, b fluxer.ReactionEmoji) bool {
	if a.ID != "" || b.ID != "" {
		return a.ID == b.ID
	}
	return a.Name == b.Name
}

func (c *Cache) addReaction(channelID, messageID, userID string, emoji fluxer.ReactionEmoji) {
	me := userID == c.Self().ID
	c.editMessage(channelID, messageID, func(m *fluxer.Message) {
		for i := range m.Reactions {
			if sameEmoji(m.Reactions[i].Emoji, emoji) {
				m.Reactions[i].Count++
				m.Reactions[i].Me = m.Reactions[i].Me || me
				return
			}
		}
		m.Reactions = append(m.Reactions, fluxer.MessageReaction{Emoji: emoji, Count: 1, Me: me})
	})
}

func (c *Cache) removeReaction(channelID, messageID, userID string, emoji fluxer.ReactionEmoji) {
	me := userID == c.Self().ID
	c.editMessage(channelID, messageID, func(m *fluxer.Message) {
		for i := range m.Reactions {
			if !sameEmoji(m.Reactions[i].Emoji, emoji) {
				continue
			}
			m.Reactions[i].Count--
			if me {
				m.Reactions[i].Me = false
			}
			if m.Reactions[i].Count <= 0 {
				m.Reactions = slices.Delete(m.Reactions, i, i+1)
			}
			return
		}
	})
}

func (c *Cache) applyMessage(e fluxer.Event) {
	switch e := e.(type) {
	case *fluxer.MessageCreate:
		c.putMessage(e.Message)
	case *fluxer.MessageUpdate:
		if _, ok := c.store.Messages().Get(e.ChannelID, e.ID); ok || e.Author.ID != "" {
			c.putMessage(e.Message)
		}
	case *fluxer.MessageDelete:
		c.store.Messages().Delete(e.ChannelID, e.ID)
	case *fluxer.MessageDeleteBulk:
		for _, id := range e.IDs {
			c.store.Messages().Delete(e.ChannelID, id)
		}
	case *fluxer.MessageReactionAdd:
		c.addReaction(e.ChannelID, e.MessageID, e.UserID, e.Emoji)
	case *fluxer.MessageReactionAddMany:
		for _, r := range e.Reactions {
			c.addReaction(e.ChannelID, e.MessageID, r.UserID, r.Emoji)
		}
	case *fluxer.MessageReactionRemove:
		c.removeReaction(e.ChannelID, e.MessageID, e.UserID, e.Emoji)
	case *fluxer.MessageReactionRemoveAll:
		c.editMessage(e.ChannelID, e.MessageID, func(m *fluxer.Message) { m.Reactions = nil })
	case *fluxer.MessageReactionRemoveEmoji:
		c.editMessage(e.ChannelID, e.MessageID, func(m *fluxer.Message) {
			m.Reactions = slices.DeleteFunc(m.Reactions, func(r fluxer.MessageReaction) bool { return sameEmoji(r.Emoji, e.Emoji) })
		})
	case *fluxer.ChannelPinsUpdate:
		if ch, ok := c.channel(e.ChannelID); ok {
			ch.LastPinTimestamp = e.LastPinTimestamp
			c.putChannel(ch)
		}
	}
}

// Message returns a cached message.
func (c *Cache) Message(channelID, messageID string) (fluxer.Message, bool) {
	return c.store.Messages().Get(channelID, messageID)
}

// MessageByID returns a cached message without knowing its channel. It needs a
// store whose message table supports Locate.
func (c *Cache) MessageByID(messageID string) (fluxer.Message, bool) {
	channelID, ok := c.store.Messages().Locate(messageID)
	if !ok {
		return fluxer.Message{}, false
	}
	return c.store.Messages().Get(channelID, messageID)
}

// Messages returns cached messages of a channel between q.After and q.Before, newest
// first. When only q.After is set, it returns the messages right after it, oldest
// first. q.Limit caps the number of results.
func (c *Cache) Messages(channelID string, q fluxer.MessageQuery) []fluxer.Message {
	var out []fluxer.Message
	for _, m := range c.store.Messages().Group(channelID) {
		if q.Before != "" && fluxer.CompareSnowflakes(m.ID, q.Before) >= 0 {
			continue
		}
		if q.After != "" && fluxer.CompareSnowflakes(m.ID, q.After) <= 0 {
			continue
		}
		out = append(out, m)
	}
	slices.SortFunc(out, func(a, b fluxer.Message) int { return fluxer.CompareSnowflakes(b.ID, a.ID) })
	if q.After != "" && q.Before == "" {
		slices.Reverse(out)
	}
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out
}

// SweepMessages removes expired messages and messages that fail the message policy,
// calling the evict function for each one. It returns the number removed.
func (c *Cache) SweepMessages() int {
	now := time.Now()
	removed := 0
	for channelID := range c.store.Messages().Groups() {
		for _, m := range c.store.Messages().Group(channelID) {
			if c.expired(m, now) || c.messagePolicy != nil && !c.messagePolicy(c, m) {
				c.evict(m)
				removed++
			}
		}
	}
	return removed
}
