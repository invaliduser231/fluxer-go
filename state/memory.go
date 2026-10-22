package state

import (
	"hash/maphash"
	"iter"
	"maps"
	"slices"
	"sync"

	"github.com/invaliduser231/fluxer-go"
)

const memShards = 32

var seed = maphash.MakeSeed()

func shardOf(s string) int { return int(maphash.String(seed, s) % memShards) }

type memShard[V any] struct {
	mu     sync.RWMutex
	groups map[string]map[string]V
}

type memIndex struct {
	mu    sync.RWMutex
	where map[string]string
}

type MemoryTable[V any] struct {
	shards  [memShards]memShard[V]
	indexed bool
	index   [memShards]memIndex
}

func NewMemoryTable[V any](indexed bool) *MemoryTable[V] {
	t := &MemoryTable[V]{indexed: indexed}
	for i := range t.shards {
		t.shards[i].groups = map[string]map[string]V{}
		t.index[i].where = map[string]string{}
	}
	return t
}

func (t *MemoryTable[V]) shard(group string) *memShard[V] { return &t.shards[shardOf(group)] }

func (t *MemoryTable[V]) Get(group, key string) (V, bool) {
	s := t.shard(group)
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.groups[group][key]
	return v, ok
}

func (t *MemoryTable[V]) Put(group, key string, v V) {
	s := t.shard(group)
	s.mu.Lock()
	g := s.groups[group]
	if g == nil {
		g = map[string]V{}
		s.groups[group] = g
	}
	g[key] = v
	s.mu.Unlock()
	if t.indexed {
		ix := &t.index[shardOf(key)]
		ix.mu.Lock()
		ix.where[key] = group
		ix.mu.Unlock()
	}
}

func (t *MemoryTable[V]) Delete(group, key string) {
	s := t.shard(group)
	s.mu.Lock()
	if g := s.groups[group]; g != nil {
		delete(g, key)
		if len(g) == 0 {
			delete(s.groups, group)
		}
	}
	s.mu.Unlock()
	t.unindex(group, key)
}

func (t *MemoryTable[V]) unindex(group, key string) {
	if !t.indexed {
		return
	}
	ix := &t.index[shardOf(key)]
	ix.mu.Lock()
	if ix.where[key] == group {
		delete(ix.where, key)
	}
	ix.mu.Unlock()
}

func (t *MemoryTable[V]) DeleteGroup(group string) {
	s := t.shard(group)
	s.mu.Lock()
	keys := slices.Collect(maps.Keys(s.groups[group]))
	delete(s.groups, group)
	s.mu.Unlock()
	for _, k := range keys {
		t.unindex(group, k)
	}
}

func (t *MemoryTable[V]) Group(group string) iter.Seq2[string, V] {
	s := t.shard(group)
	s.mu.RLock()
	snapshot := maps.Clone(s.groups[group])
	s.mu.RUnlock()
	return maps.All(snapshot)
}

func (t *MemoryTable[V]) Len(group string) int {
	s := t.shard(group)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.groups[group])
}

func (t *MemoryTable[V]) Groups() iter.Seq[string] {
	var all []string
	for i := range t.shards {
		s := &t.shards[i]
		s.mu.RLock()
		for g := range s.groups {
			all = append(all, g)
		}
		s.mu.RUnlock()
	}
	return slices.Values(all)
}

func (t *MemoryTable[V]) Locate(key string) (string, bool) {
	if !t.indexed {
		return "", false
	}
	ix := &t.index[shardOf(key)]
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	g, ok := ix.where[key]
	return g, ok
}

type MemoryStore struct {
	guilds    *MemoryTable[GuildRecord]
	channels  *MemoryTable[fluxer.Channel]
	roles     *MemoryTable[fluxer.Role]
	members   *MemoryTable[fluxer.Member]
	voice     *MemoryTable[fluxer.VoiceState]
	emojis    *MemoryTable[fluxer.Emoji]
	stickers  *MemoryTable[fluxer.Sticker]
	users     *MemoryTable[fluxer.User]
	presences *MemoryTable[fluxer.Presence]
	messages  *MemoryTable[fluxer.Message]
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		guilds:    NewMemoryTable[GuildRecord](false),
		channels:  NewMemoryTable[fluxer.Channel](true),
		roles:     NewMemoryTable[fluxer.Role](false),
		members:   NewMemoryTable[fluxer.Member](false),
		voice:     NewMemoryTable[fluxer.VoiceState](false),
		emojis:    NewMemoryTable[fluxer.Emoji](false),
		stickers:  NewMemoryTable[fluxer.Sticker](false),
		users:     NewMemoryTable[fluxer.User](false),
		presences: NewMemoryTable[fluxer.Presence](false),
		messages:  NewMemoryTable[fluxer.Message](true),
	}
}

func (s *MemoryStore) Guilds() Table[GuildRecord]            { return s.guilds }
func (s *MemoryStore) Channels() Table[fluxer.Channel]       { return s.channels }
func (s *MemoryStore) Roles() Table[fluxer.Role]             { return s.roles }
func (s *MemoryStore) Members() Table[fluxer.Member]         { return s.members }
func (s *MemoryStore) VoiceStates() Table[fluxer.VoiceState] { return s.voice }
func (s *MemoryStore) Emojis() Table[fluxer.Emoji]           { return s.emojis }
func (s *MemoryStore) Stickers() Table[fluxer.Sticker]       { return s.stickers }
func (s *MemoryStore) Users() Table[fluxer.User]             { return s.users }
func (s *MemoryStore) Presences() Table[fluxer.Presence]     { return s.presences }
func (s *MemoryStore) Messages() Table[fluxer.Message]       { return s.messages }
