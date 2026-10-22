package state

import "sync/atomic"

type Stats struct {
	Guilds      int
	Channels    int
	Roles       int
	Members     int
	Users       int
	Presences   int
	VoiceStates int
	Messages    int
	Emojis      int
	Stickers    int
	Hits        uint64
	Misses      uint64
}

type counters struct {
	hits   atomic.Uint64
	misses atomic.Uint64
}

func (c *Cache) hit()  { c.counters.hits.Add(1) }
func (c *Cache) miss() { c.counters.misses.Add(1) }

func count[V any](t Table[V]) int {
	n := 0
	for g := range t.Groups() {
		n += t.Len(g)
	}
	return n
}

func (c *Cache) Stats() Stats {
	s := c.store
	return Stats{
		Guilds:      count(s.Guilds()),
		Channels:    count(s.Channels()),
		Roles:       count(s.Roles()),
		Members:     count(s.Members()),
		Users:       count(s.Users()),
		Presences:   count(s.Presences()),
		VoiceStates: count(s.VoiceStates()),
		Messages:    count(s.Messages()),
		Emojis:      count(s.Emojis()),
		Stickers:    count(s.Stickers()),
		Hits:        c.counters.hits.Load(),
		Misses:      c.counters.misses.Load(),
	}
}
