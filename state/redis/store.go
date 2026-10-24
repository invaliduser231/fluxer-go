package redis

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"slices"
	"time"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/state"
	goredis "github.com/redis/go-redis/v9"
)

// Store is a state.Store backed by Redis, so several processes can share one cache.
// Each group is a Redis hash, and all keys start with the store's prefix.
type Store struct {
	// Timeout bounds each Redis operation. The default is 2 seconds.
	Timeout time.Duration
	// OnError receives Redis and decoding errors. Table methods have no error results,
	// so a failed read looks like a miss and a failed write is dropped; OnError is the
	// only place these errors show up.
	OnError func(error)

	client goredis.UniversalClient
	prefix string

	guilds    *table[state.GuildRecord]
	channels  *table[fluxer.Channel]
	roles     *table[fluxer.Role]
	members   *table[fluxer.Member]
	voice     *table[fluxer.VoiceState]
	emojis    *table[fluxer.Emoji]
	stickers  *table[fluxer.Sticker]
	users     *table[fluxer.User]
	presences *table[fluxer.Presence]
	messages  *table[fluxer.Message]
}

// New returns a store that uses client and starts every key with prefix, or with
// "fluxer" when prefix is empty.
func New(client goredis.UniversalClient, prefix string) *Store {
	s := &Store{client: client, prefix: cmp.Or(prefix, "fluxer")}
	s.guilds = newTable[state.GuildRecord](s, "guilds", false)
	s.channels = newTable[fluxer.Channel](s, "channels", true)
	s.roles = newTable[fluxer.Role](s, "roles", false)
	s.members = newTable[fluxer.Member](s, "members", false)
	s.voice = newTable[fluxer.VoiceState](s, "voice", false)
	s.emojis = newTable[fluxer.Emoji](s, "emojis", false)
	s.stickers = newTable[fluxer.Sticker](s, "stickers", false)
	s.users = newTable[fluxer.User](s, "users", false)
	s.presences = newTable[fluxer.Presence](s, "presences", false)
	s.messages = newTable[fluxer.Message](s, "messages", true)
	return s
}

// Guilds returns the guild table.
func (s *Store) Guilds() state.Table[state.GuildRecord] { return s.guilds }

// Channels returns the channel table, which is indexed.
func (s *Store) Channels() state.Table[fluxer.Channel] { return s.channels }

// Roles returns the role table.
func (s *Store) Roles() state.Table[fluxer.Role] { return s.roles }

// Members returns the member table.
func (s *Store) Members() state.Table[fluxer.Member] { return s.members }

// VoiceStates returns the voice state table.
func (s *Store) VoiceStates() state.Table[fluxer.VoiceState] { return s.voice }

// Emojis returns the emoji table.
func (s *Store) Emojis() state.Table[fluxer.Emoji] { return s.emojis }

// Stickers returns the sticker table.
func (s *Store) Stickers() state.Table[fluxer.Sticker] { return s.stickers }

// Users returns the user table.
func (s *Store) Users() state.Table[fluxer.User] { return s.users }

// Presences returns the presence table.
func (s *Store) Presences() state.Table[fluxer.Presence] { return s.presences }

// Messages returns the message table, which is indexed.
func (s *Store) Messages() state.Table[fluxer.Message] { return s.messages }

func (s *Store) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cmp.Or(s.Timeout, 2*time.Second))
}

func (s *Store) fail(err error) {
	if err != nil && !errors.Is(err, goredis.Nil) && s.OnError != nil {
		s.OnError(err)
	}
}

type table[V any] struct {
	s       *Store
	name    string
	indexed bool
}

func newTable[V any](s *Store, name string, indexed bool) *table[V] {
	return &table[V]{s: s, name: name, indexed: indexed}
}

func (t *table[V]) groupKey(group string) string { return t.s.prefix + ":" + t.name + ":g:" + group }
func (t *table[V]) groupsKey() string            { return t.s.prefix + ":" + t.name + ":groups" }
func (t *table[V]) indexKey() string             { return t.s.prefix + ":" + t.name + ":idx" }

func (t *table[V]) Get(group, key string) (V, bool) {
	var v V
	ctx, cancel := t.s.ctx()
	defer cancel()
	data, err := t.s.client.HGet(ctx, t.groupKey(group), key).Bytes()
	if err != nil {
		t.s.fail(err)
		return v, false
	}
	if err := json.Unmarshal(data, &v); err != nil {
		t.s.fail(err)
		return v, false
	}
	return v, true
}

func (t *table[V]) Put(group, key string, v V) {
	data, err := json.Marshal(v)
	if err != nil {
		t.s.fail(err)
		return
	}
	ctx, cancel := t.s.ctx()
	defer cancel()
	_, err = t.s.client.TxPipelined(ctx, func(p goredis.Pipeliner) error {
		p.HSet(ctx, t.groupKey(group), key, data)
		p.SAdd(ctx, t.groupsKey(), group)
		if t.indexed {
			p.HSet(ctx, t.indexKey(), key, group)
		}
		return nil
	})
	t.s.fail(err)
}

func (t *table[V]) Delete(group, key string) {
	ctx, cancel := t.s.ctx()
	defer cancel()
	var left *goredis.IntCmd
	_, err := t.s.client.TxPipelined(ctx, func(p goredis.Pipeliner) error {
		p.HDel(ctx, t.groupKey(group), key)
		left = p.HLen(ctx, t.groupKey(group))
		return nil
	})
	if err != nil {
		t.s.fail(err)
		return
	}
	if left.Val() == 0 {
		t.s.fail(t.s.client.SRem(ctx, t.groupsKey(), group).Err())
	}
	t.unindex(ctx, group, key)
}

func (t *table[V]) unindex(ctx context.Context, group string, keys ...string) {
	if !t.indexed || len(keys) == 0 {
		return
	}
	where, err := t.s.client.HMGet(ctx, t.indexKey(), keys...).Result()
	if err != nil {
		t.s.fail(err)
		return
	}
	var stale []string
	for i, w := range where {
		if g, ok := w.(string); ok && g == group {
			stale = append(stale, keys[i])
		}
	}
	if len(stale) > 0 {
		t.s.fail(t.s.client.HDel(ctx, t.indexKey(), stale...).Err())
	}
}

func (t *table[V]) DeleteGroup(group string) {
	ctx, cancel := t.s.ctx()
	defer cancel()
	var keys []string
	if t.indexed {
		var err error
		if keys, err = t.s.client.HKeys(ctx, t.groupKey(group)).Result(); err != nil {
			t.s.fail(err)
			return
		}
	}
	_, err := t.s.client.TxPipelined(ctx, func(p goredis.Pipeliner) error {
		p.Del(ctx, t.groupKey(group))
		p.SRem(ctx, t.groupsKey(), group)
		return nil
	})
	t.s.fail(err)
	t.unindex(ctx, group, keys...)
}

func (t *table[V]) Group(group string) iter.Seq2[string, V] {
	ctx, cancel := t.s.ctx()
	defer cancel()
	raw, err := t.s.client.HGetAll(ctx, t.groupKey(group)).Result()
	if err != nil {
		t.s.fail(err)
	}
	return func(yield func(string, V) bool) {
		for key, data := range raw {
			var v V
			if err := json.Unmarshal([]byte(data), &v); err != nil {
				t.s.fail(err)
				continue
			}
			if !yield(key, v) {
				return
			}
		}
	}
}

func (t *table[V]) Len(group string) int {
	ctx, cancel := t.s.ctx()
	defer cancel()
	n, err := t.s.client.HLen(ctx, t.groupKey(group)).Result()
	t.s.fail(err)
	return int(n)
}

func (t *table[V]) Groups() iter.Seq[string] {
	ctx, cancel := t.s.ctx()
	defer cancel()
	groups, err := t.s.client.SMembers(ctx, t.groupsKey()).Result()
	t.s.fail(err)
	return slices.Values(groups)
}

func (t *table[V]) Locate(key string) (string, bool) {
	if !t.indexed {
		return "", false
	}
	ctx, cancel := t.s.ctx()
	defer cancel()
	group, err := t.s.client.HGet(ctx, t.indexKey(), key).Result()
	if err != nil {
		t.s.fail(err)
		return "", false
	}
	return group, true
}

// Flush deletes every key that starts with the store's prefix.
func (s *Store) Flush(ctx context.Context) error {
	var cursor uint64
	for {
		keys, next, err := s.client.Scan(ctx, cursor, s.prefix+":*", 500).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := s.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		if next == 0 {
			return nil
		}
		cursor = next
	}
}
