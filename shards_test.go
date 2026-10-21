package fluxer_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

func TestShards(t *testing.T) {
	srv, c := setup(t)
	var mu sync.Mutex
	ready := map[int]bool{}
	m := &fluxer.Shards{
		Token:  srv.Token,
		Count:  3,
		Client: c,
		Pacer:  fluxer.NewPacer(6000),
		Handler: func(shard int, event string, data json.RawMessage) {
			if event == "READY" {
				mu.Lock()
				ready[shard] = true
				mu.Unlock()
			}
		},
		Configure: func(s *fluxer.Session) { s.MinBackoff, s.MaxBackoff = 10*time.Millisecond, 50*time.Millisecond },
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- m.Run(ctx) }()
	waitFor(t, "all shards ready", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(ready) == 3
	})
	st := srv.Stats()
	var ids []int
	for _, pair := range st.Shards {
		if pair[1] != 3 {
			t.Errorf("shard count %v", pair)
		}
		ids = append(ids, pair[0])
	}
	slices.Sort(ids)
	if !slices.Equal(ids, []int{0, 1, 2}) {
		t.Errorf("shards %v", ids)
	}
	guild := "1189375284394692608"
	if s := m.ForGuild(guild); s == nil || s.Shard.ID != fluxer.ShardFor(guild, 3) {
		t.Errorf("got %+v", s)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
}

func TestShardsStopOnInvalidToken(t *testing.T) {
	srv, c := setup(t)
	m := &fluxer.Shards{Token: "wrong", Client: c, Count: 2, URL: srv.GatewayURL()}
	done := make(chan error, 1)
	go func() { done <- m.Run(context.Background()) }()
	select {
	case err := <-done:
		if !errors.Is(err, fluxer.ErrInvalidToken) {
			t.Errorf("got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop")
	}
}
