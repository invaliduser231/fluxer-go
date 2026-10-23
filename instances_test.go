package fluxer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/fluxertest"
)

type countingTransport struct{ discoveries atomic.Int32 }

func (t *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.HasSuffix(r.URL.Path, "/.well-known/fluxer") {
		t.discoveries.Add(1)
	}
	return http.DefaultTransport.RoundTrip(r)
}

func TestInstancesCacheDiscovery(t *testing.T) {
	a := fluxertest.NewServer("1000.a")
	defer a.Close()
	b := fluxertest.NewServer("2000.b")
	b.SetUser(fluxer.User{ID: "2000", Username: "otherbot", Bot: true})
	tr := &countingTransport{}
	reg := &fluxer.Instances{HTTP: &http.Client{Transport: tr}, TTL: time.Hour, FailureTTL: time.Millisecond}
	ctx := context.Background()

	for range 3 {
		if _, err := reg.Get(ctx, a.URL); err != nil {
			t.Fatal(err)
		}
	}
	if n := tr.discoveries.Load(); n != 1 {
		t.Errorf("%d discoveries", n)
	}
	cb, err := reg.Client(ctx, b.URL+"/", b.Token)
	if err != nil {
		t.Fatal(err)
	}
	if u, err := cb.CurrentUser(ctx); err != nil || u.Username != "otherbot" {
		t.Errorf("got %+v, %v", u, err)
	}
	if got := reg.Origins(); len(got) != 2 {
		t.Errorf("origins %v", got)
	}
	if reg.Pacer(a.URL) == reg.Pacer(b.URL) || reg.Pacer(a.URL) != reg.Pacer(a.URL) {
		t.Error("pacers are not per instance")
	}

	reg.TTL = time.Millisecond
	reg.Forget(b.URL)
	if _, err := reg.Get(ctx, b.URL); err != nil {
		t.Fatal(err)
	}
	b.Close()
	time.Sleep(5 * time.Millisecond)
	if inst, err := reg.Get(ctx, b.URL); err != nil || inst == nil {
		t.Errorf("stale instance not kept: %v", err)
	}
	if _, err := reg.Get(ctx, "ftp://nope"); err == nil {
		t.Error("expected error")
	}
}

func TestSessionsOnSeveralInstances(t *testing.T) {
	a := fluxertest.NewServer("1000.a")
	defer a.Close()
	b := fluxertest.NewServer("2000.b")
	defer b.Close()
	reg := fluxer.NewInstances()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	seen := map[string][]string{}
	var wg sync.WaitGroup
	for _, srv := range []*fluxertest.Server{a, b} {
		s, err := reg.Session(ctx, srv.URL, srv.Token)
		if err != nil {
			t.Fatal(err)
		}
		origin := srv.URL
		s.Handler = func(event string, data json.RawMessage) {
			if event != "MESSAGE_CREATE" {
				return
			}
			var m fluxer.Message
			_ = json.Unmarshal(data, &m)
			mu.Lock()
			seen[origin] = append(seen[origin], m.Content)
			mu.Unlock()
		}
		s.MinBackoff = 10 * time.Millisecond
		wg.Go(func() { _ = s.Run(ctx) })
	}
	waitFor(t, "both sessions", func() bool { return a.Stats().Identifies == 1 && b.Stats().Identifies == 1 })
	a.Dispatch("MESSAGE_CREATE", fluxer.Message{ID: "1", ChannelID: "2", Content: "from a"})
	b.Dispatch("MESSAGE_CREATE", fluxer.Message{ID: "1", ChannelID: "2", Content: "from b"})
	waitFor(t, "messages", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(seen[a.URL]) == 1 && len(seen[b.URL]) == 1
	})
	if seen[a.URL][0] != "from a" || seen[b.URL][0] != "from b" {
		t.Errorf("got %v", seen)
	}
	cancel()
	wg.Wait()
}
