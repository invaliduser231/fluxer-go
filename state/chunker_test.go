package state_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/fluxertest"
	"github.com/invaliduser231/fluxer-go/state"
)

func TestChunkerAgainstGateway(t *testing.T) {
	srv := fluxertest.NewServer("1000.secret")
	defer srv.Close()
	srv.AddGuild(fluxer.Guild{ID: "3", Name: "Test"})
	for _, u := range []fluxer.User{{ID: "7", Username: "anna"}, {ID: "8", Username: "ben"}, {ID: "9", Username: "annika"}} {
		srv.AddMember("3", fluxer.Member{User: &u})
	}
	cache := state.New()
	s := &fluxer.Session{URL: srv.GatewayURL(), Token: srv.Token, MinBackoff: 10 * time.Millisecond}
	chunker := state.NewChunker(s)
	ready := make(chan struct{}, 1)
	s.Handler = func(event string, data json.RawMessage) {
		cache.Handle(event, data)
		chunker.Handle(event, data)
		if event == "READY" {
			ready <- struct{}{}
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() { _ = s.Run(ctx) })
	defer func() {
		cancel()
		wg.Wait()
	}()
	<-ready

	members, err := chunker.Request(ctx, fluxer.RequestMembers{GuildID: "3", Query: "ann", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 || members[0].GuildID != "3" {
		t.Errorf("got %+v", members)
	}
	if all, err := chunker.RequestAll(ctx, "3"); err != nil || len(all) != 3 {
		t.Errorf("got %d, %v", len(all), err)
	}
	if m, ok := cache.Member("3", "8"); !ok || m.User.Username != "ben" {
		t.Errorf("cache did not take the chunk: %+v", m)
	}
}

type silentRequester struct {
	mu   sync.Mutex
	sent []fluxer.RequestMembers
}

func (r *silentRequester) RequestGuildMembers(_ context.Context, req fluxer.RequestMembers) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, req)
	return nil
}

func (r *silentRequester) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.sent)
}

func (r *silentRequester) nonce(i int) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sent[i].Nonce
}

func TestChunkerMultipleChunksAndRateLimit(t *testing.T) {
	r := &silentRequester{}
	k := state.NewChunker(r)
	k.Timeout = time.Second
	result := make(chan []fluxer.Member, 1)
	go func() {
		ms, _ := k.RequestAll(context.Background(), "3")
		result <- ms
	}()
	waitFor(t, "request", func() bool { return r.count() == 1 })
	nonce := r.nonce(0)
	for i := range 2 {
		chunk, _ := json.Marshal(fluxer.GuildMembersChunk{GuildID: "3", Nonce: nonce, ChunkIndex: i, ChunkCount: 2,
			Members: []fluxer.Member{{User: &fluxer.User{ID: string(rune('7' + i))}}}})
		k.Handle("GUILD_MEMBERS_CHUNK", chunk)
	}
	if ms := <-result; len(ms) != 2 {
		t.Errorf("got %d members", len(ms))
	}

	errc := make(chan error, 1)
	go func() {
		_, err := k.RequestAll(context.Background(), "4")
		errc <- err
	}()
	waitFor(t, "second request", func() bool { return r.count() == 2 })
	k.Handle("RATE_LIMITED", json.RawMessage(`{"opcode":8,"retry_after":12.5,"meta":{"guild_id":"4"}}`))
	var rl *state.RateLimitedError
	if err := <-errc; !errors.As(err, &rl) || rl.RetryAfter != 12500*time.Millisecond {
		t.Errorf("got %v", err)
	}
}

func TestWaitGuilds(t *testing.T) {
	c := state.New()
	c.Handle("READY", json.RawMessage(`{"user":{"id":"1000"},"guilds":[{"id":"3","unavailable":true},{"id":"4","unavailable":true}]}`))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := c.WaitGuilds(ctx); !errors.Is(err, context.DeadlineExceeded) || c.PendingGuilds() != 2 {
		t.Fatalf("got %v with %d pending", err, c.PendingGuilds())
	}
	c.Handle("GUILD_CREATE", json.RawMessage(guildCreate))
	c.Handle("GUILD_DELETE", json.RawMessage(`{"id":"4","unavailable":true}`))
	if err := c.WaitGuilds(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestWaitGuildsCalledBeforeReady(t *testing.T) {
	c := state.New()
	done := make(chan error, 1)
	go func() { done <- c.WaitGuilds(context.Background()) }()
	time.Sleep(20 * time.Millisecond)
	c.Handle("READY", json.RawMessage(`{"user":{"id":"1000"},"guilds":[{"id":"3","unavailable":true}]}`))
	c.Handle("GUILD_CREATE", json.RawMessage(guildCreate))
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WaitGuilds kept waiting")
	}
}

func TestChunkerIgnoresExtraChunks(t *testing.T) {
	r := &silentRequester{}
	k := state.NewChunker(r)
	go func() { _, _ = k.RequestAll(context.Background(), "3") }()
	waitFor(t, "request", func() bool { return r.count() == 1 })
	for _, i := range []int{0, 1} {
		chunk, _ := json.Marshal(fluxer.GuildMembersChunk{GuildID: "3", Nonce: r.nonce(0), ChunkIndex: i, ChunkCount: 1})
		k.Handle("GUILD_MEMBERS_CHUNK", chunk)
	}
	k.Handle("RATE_LIMITED", json.RawMessage(`{"opcode":8,"retry_after":1,"meta":{"guild_id":"3"}}`))
}
