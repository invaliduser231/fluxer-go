package fluxer

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"sync"
)

type Shards struct {
	URL       string
	Token     string
	Count     int
	Client    *Client
	Handler   func(shard int, event string, data json.RawMessage)
	Configure func(s *Session)
	Pacer     *Pacer

	mu       sync.Mutex
	sessions []*Session
}

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

func (m *Shards) Sessions() []*Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*Session(nil), m.sessions...)
}

func (m *Shards) ForGuild(guildID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sessions) == 0 {
		return nil
	}
	return m.sessions[ShardFor(guildID, len(m.sessions))]
}
