package fluxer_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

func TestCommandsNeedConnection(t *testing.T) {
	s := &fluxer.Session{Token: "x"}
	if err := s.UpdateStatus(context.Background(), fluxer.StatusUpdate{Status: fluxer.StatusIdle}); !errors.Is(err, fluxer.ErrNotConnected) {
		t.Errorf("got %v", err)
	}
}

func TestIdentifyCarriesPresenceAndShard(t *testing.T) {
	srv, _ := setup(t)
	r := fluxer.NewRouter()
	ready := make(chan *fluxer.Ready, 1)
	fluxer.On(r, func(e *fluxer.Ready) { ready <- e })
	runSession(t, &fluxer.Session{
		URL: srv.GatewayURL(), Token: srv.Token, Handler: r.Handle,
		Shard:    fluxer.Shard{ID: 1, Count: 4},
		Flags:    fluxer.FlagChannelThreads,
		Presence: &fluxer.StatusUpdate{Status: fluxer.StatusDND},
	})
	<-ready
	var id struct {
		Shard      []int               `json:"shard"`
		Flags      int                 `json:"flags"`
		Presence   fluxer.StatusUpdate `json:"presence"`
		Properties map[string]string   `json:"properties"`
	}
	if err := json.Unmarshal(srv.Stats().Identify, &id); err != nil {
		t.Fatal(err)
	}
	if len(id.Shard) != 2 || id.Shard[0] != 1 || id.Shard[1] != 4 || id.Flags != 4 || id.Presence.Status != "dnd" || id.Properties["browser"] != "fluxer-go" {
		t.Errorf("got %+v", id)
	}
}

func TestRequestGuildMembers(t *testing.T) {
	srv, _ := guildSetup(t)
	r := fluxer.NewRouter()
	ready := make(chan struct{}, 1)
	chunks := make(chan *fluxer.GuildMembersChunk, 1)
	counts := make(chan *fluxer.GuildCountsUpdate, 1)
	fluxer.On(r, func(*fluxer.Ready) { ready <- struct{}{} })
	fluxer.On(r, func(e *fluxer.GuildMembersChunk) { chunks <- e })
	fluxer.On(r, func(e *fluxer.GuildCountsUpdate) { counts <- e })
	s := &fluxer.Session{URL: srv.GatewayURL(), Token: srv.Token, Handler: r.Handle}
	runSession(t, s)
	<-ready

	ctx := context.Background()
	if err := s.RequestGuildMembers(ctx, fluxer.RequestMembers{GuildID: "3", Query: "an", Limit: 10, Nonce: "n1"}); err != nil {
		t.Fatal(err)
	}
	select {
	case c := <-chunks:
		if c.GuildID != "3" || c.Nonce != "n1" || len(c.Members) != 1 || c.Members[0].User.Username != "anna" {
			t.Errorf("got %+v", c)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no chunk")
	}
	if err := s.RequestGuildCounts(ctx, []string{"3", "99"}, "c1"); err != nil {
		t.Fatal(err)
	}
	select {
	case c := <-counts:
		if c.Nonce != "c1" || len(c.Counts) != 1 || c.Counts[0].MemberCount != 3 {
			t.Errorf("got %+v", c)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no counts")
	}
	if err := s.UpdateStatus(ctx, fluxer.StatusUpdate{Status: fluxer.StatusIdle, AFK: true}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "presence command", func() bool { return len(srv.Commands(3)) == 1 })
	if got := string(srv.Commands(3)[0]); !strings.Contains(got, `"status":"idle"`) || !strings.Contains(got, `"afk":true`) {
		t.Errorf("got %s", got)
	}
}
