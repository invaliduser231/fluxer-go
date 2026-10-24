package fluxer

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"sync"
)

// Shards runs one Session per shard with a shared identify Pacer.
type Shards struct {
	// URL is the gateway URL for every shard.
	URL   string
	Token string
	// Count is the number of shards. When Count is zero or URL is empty, Run fetches
	// Client.GatewayBot and fills in what is missing, with at least one shard.
	Count int
	// Client is only needed when Count or URL is unset.
	Client *Client
	// Handler receives the dispatches of all shards together with the shard ID. It runs on the
	// gateway read goroutine of that shard.
	Handler func(shard int, event string, data json.RawMessage)
	// Configure is called for each Session before it starts, for options such as Compress or
	// IgnoredEvents. It may also replace the Handler and Pacer set by Shards.
	Configure func(s *Session)
	// Pacer spaces out identifies across all shards. Nil uses NewPacer(60).
	Pacer *Pacer

	mu       sync.Mutex
	sessions []*Session
}

// Run starts every shard and blocks until ctx ends or a shard's Session.Run returns an error. That
// error stops the other shards and is returned. Without Count and URL, Run fails if Client is nil
// or GatewayBot fails.
func (m *Shards) Run(ctx context.Context) error {
	count := m.Count
	url := m.URL
	if count <= 0 || url == "" {
		if m.Client == nil {
			return errors.New("fluxer: shards need a count and a url or a client to ask for them")
		}
		g, err := m.Client.GatewayBot(ctx)
		if err != nil {
			return err
		}
		count = cmp.Or(count, max(g.Shards, 1))
		url = cmp.Or(url, g.URL)
	}
	pacer := m.Pacer
	if pacer == nil {
		pacer = NewPacer(60)
	}
	sessions := make([]*Session, count)
	for i := range sessions {
		s := &Session{URL: url, Token: m.Token, Shard: Shard{ID: i, Count: count}, Pacer: pacer}
		if m.Handler != nil {
			id := i
			s.Handler = func(event string, data json.RawMessage) { m.Handler(id, event, data) }
		}
		if m.Configure != nil {
			m.Configure(s)
		}
		sessions[i] = s
	}
	m.mu.Lock()
	m.sessions = sessions
	m.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errs := make(chan error, count)
	var wg sync.WaitGroup
	for _, s := range sessions {
		wg.Go(func() {
			if err := s.Run(ctx); err != nil && ctx.Err() == nil {
				errs <- err
				cancel()
			}
		})
	}
	wg.Wait()
	close(errs)
	if err := <-errs; err != nil {
		return err
	}
	return ctx.Err()
}

// Sessions returns a copy of the shard sessions indexed by shard ID. It is empty until Run has
// started.
func (m *Shards) Sessions() []*Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*Session(nil), m.sessions...)
}

// ForGuild returns the session of the shard that owns guildID, or nil before Run has started.
func (m *Shards) ForGuild(guildID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sessions) == 0 {
		return nil
	}
	return m.sessions[ShardFor(guildID, len(m.sessions))]
}
