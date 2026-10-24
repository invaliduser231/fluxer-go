package state

import (
	"context"
	"sync"

	"github.com/invaliduser231/fluxer-go"
)

// Resolver reads from a Cache and falls back to the REST API on a miss. Fetched
// values are written back to the cache, subject to its flags and policies, and
// concurrent fetches of the same value share one request.
type Resolver struct {
	Cache  *Cache
	Client *fluxer.Client

	mu     sync.Mutex
	flight map[string]*call
}

type call struct {
	done chan struct{}
	val  any
	err  error
}

// NewResolver returns a resolver for c that fetches missing data with client.
func NewResolver(c *Cache, client *fluxer.Client) *Resolver {
	return &Resolver{Cache: c, Client: client}
}

func once[T any](ctx context.Context, r *Resolver, key string, fetch func() (T, error)) (T, error) {
	r.mu.Lock()
	if r.flight == nil {
		r.flight = map[string]*call{}
	}
	if c, ok := r.flight[key]; ok {
		r.mu.Unlock()
		select {
		case <-c.done:
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		}
		v, _ := c.val.(T)
		return v, c.err
	}
	c := &call{done: make(chan struct{})}
	r.flight[key] = c
	r.mu.Unlock()
	v, err := fetch()
	c.val, c.err = v, err
	r.mu.Lock()
	delete(r.flight, key)
	r.mu.Unlock()
	close(c.done)
	return v, err
}

// Guild returns a guild from the cache or fetches it.
func (r *Resolver) Guild(ctx context.Context, id string) (fluxer.Guild, error) {
	if g, ok := r.Cache.Guild(id); ok {
		r.Cache.hit()
		return g, nil
	}
	r.Cache.miss()
	return once(ctx, r, "guild:"+id, func() (fluxer.Guild, error) {
		g, err := r.Client.Guild(ctx, id)
		if err != nil {
			return fluxer.Guild{}, err
		}
		r.Cache.storeGuild(*g)
		return *g, nil
	})
}

// Channel returns a channel from the cache or fetches it.
func (r *Resolver) Channel(ctx context.Context, id string) (fluxer.Channel, error) {
	if ch, ok := r.Cache.Channel(id); ok {
		r.Cache.hit()
		return ch, nil
	}
	r.Cache.miss()
	return once(ctx, r, "channel:"+id, func() (fluxer.Channel, error) {
		ch, err := r.Client.Channel(ctx, id)
		if err != nil {
			return fluxer.Channel{}, err
		}
		unlock := r.Cache.lock(ch.GuildID)
		r.Cache.putChannel(*ch)
		unlock()
		return *ch, nil
	})
}

// Member returns a member from the cache, or fetches it when it is missing or
// partial.
func (r *Resolver) Member(ctx context.Context, guildID, userID string) (fluxer.Member, error) {
	if m, ok := r.Cache.Member(guildID, userID); ok && !r.Cache.MemberPartial(guildID, userID) {
		r.Cache.hit()
		return m, nil
	}
	r.Cache.miss()
	return once(ctx, r, "member:"+guildID+":"+userID, func() (fluxer.Member, error) {
		m, err := r.Client.Member(ctx, guildID, userID)
		if err != nil {
			return fluxer.Member{}, err
		}
		m.GuildID = guildID
		unlock := r.Cache.lock(guildID)
		r.Cache.putMember(guildID, *m)
		unlock()
		return *m, nil
	})
}

// User returns a user from the cache or fetches it.
func (r *Resolver) User(ctx context.Context, id string) (fluxer.User, error) {
	if u, ok := r.Cache.User(id); ok {
		r.Cache.hit()
		return u, nil
	}
	r.Cache.miss()
	return once(ctx, r, "user:"+id, func() (fluxer.User, error) {
		u, err := r.Client.User(ctx, id)
		if err != nil {
			return fluxer.User{}, err
		}
		r.Cache.putUser(*u)
		return *u, nil
	})
}

// Message returns a message from the cache or fetches it.
func (r *Resolver) Message(ctx context.Context, channelID, messageID string) (fluxer.Message, error) {
	if m, ok := r.Cache.Message(channelID, messageID); ok {
		r.Cache.hit()
		return m, nil
	}
	r.Cache.miss()
	return once(ctx, r, "message:"+channelID+":"+messageID, func() (fluxer.Message, error) {
		m, err := r.Client.Message(ctx, channelID, messageID)
		if err != nil {
			return fluxer.Message{}, err
		}
		unlock := r.Cache.lock(m.GuildID)
		r.Cache.putMessage(*m)
		unlock()
		return *m, nil
	})
}

// Roles returns the guild's roles. Any cached role for the guild counts as a hit,
// and only a guild without cached roles is fetched.
func (r *Resolver) Roles(ctx context.Context, guildID string) ([]fluxer.Role, error) {
	if r.Cache.store.Roles().Len(guildID) > 0 {
		r.Cache.hit()
		return r.Cache.roles(guildID), nil
	}
	r.Cache.miss()
	return once(ctx, r, "roles:"+guildID, func() ([]fluxer.Role, error) {
		roles, err := r.Client.Roles(ctx, guildID)
		if err != nil {
			return nil, err
		}
		unlock := r.Cache.lock(guildID)
		for _, role := range roles {
			r.Cache.putRole(guildID, role)
		}
		unlock()
		return roles, nil
	})
}

// Permissions fetches whatever is missing of the guild, its roles, the member, the
// channel and a thread's parent, and then computes the permissions like
// Cache.Permissions. An empty channelID gives guild-level permissions.
func (r *Resolver) Permissions(ctx context.Context, guildID, channelID, userID string) (fluxer.Permissions, error) {
	g, err := r.Guild(ctx, guildID)
	if err != nil {
		return 0, err
	}
	if g.Roles, err = r.Roles(ctx, guildID); err != nil {
		return 0, err
	}
	m, err := r.Member(ctx, guildID, userID)
	if err != nil {
		return 0, err
	}
	if channelID == "" {
		return fluxer.BasePermissions(g, userID, m.Roles), nil
	}
	ch, err := r.Channel(ctx, channelID)
	if err != nil {
		return 0, err
	}
	if ch.IsThread() {
		if ch, err = r.Channel(ctx, ch.ParentID); err != nil {
			return 0, err
		}
	}
	return fluxer.ChannelPermissions(g, userID, m.Roles, ch), nil
}

func (c *Cache) storeGuild(g fluxer.Guild) {
	unlock := c.lock(g.ID)
	defer unlock()
	rec := c.record(g.ID)
	info := g
	info.Roles, info.Channels, info.Emojis, info.Stickers = nil, nil, nil, nil
	if info.MemberCount == 0 {
		info.MemberCount = rec.Guild.MemberCount
	}
	rec.Guild, rec.Unavailable = info, false
	c.putRecord(g.ID, rec)
	for _, role := range g.Roles {
		c.putRole(g.ID, role)
	}
	for _, ch := range g.Channels {
		ch.GuildID = g.ID
		c.putChannel(ch)
	}
	if g.Emojis != nil {
		c.putEmojis(g.ID, g.Emojis)
	}
	if g.Stickers != nil {
		c.putStickers(g.ID, g.Stickers)
	}
}
