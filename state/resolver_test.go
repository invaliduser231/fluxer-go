package state_test

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/fluxertest"
	"github.com/invaliduser231/fluxer-go/state"
)

func resolverSetup(t *testing.T) (*fluxertest.Server, *state.Cache, *state.Resolver) {
	t.Helper()
	srv := fluxertest.NewServer("1000.secret")
	t.Cleanup(srv.Close)
	srv.AddGuild(fluxer.Guild{
		ID: "3", Name: "Test", OwnerID: "1",
		Roles: []fluxer.Role{{ID: "3", Name: "@everyone", Permissions: fluxer.PermViewChannel | fluxer.PermSendMessages}, {ID: "40", Name: "mods", Permissions: fluxer.PermManageMessages}},
		Channels: []fluxer.Channel{{ID: "30", Name: "general", Type: fluxer.ChannelText, PermissionOverwrites: []fluxer.Overwrite{
			{ID: "3", Type: fluxer.OverwriteRole, Deny: fluxer.PermSendMessages},
		}}},
	})
	srv.AddMember("3", fluxer.Member{User: &fluxer.User{ID: "7", Username: "anna"}, Roles: []string{"40"}})
	srv.AddMessage(fluxer.Message{ID: "50", ChannelID: "30", Content: "hi", Author: fluxer.User{ID: "7", Username: "anna"}})
	c := state.New()
	return srv, c, state.NewResolver(c, srv.Client())
}

func TestResolverFetchesOnceAndCaches(t *testing.T) {
	srv, c, r := resolverSetup(t)
	ctx := context.Background()
	g, err := r.Guild(ctx, "3")
	if err != nil || g.Name != "Test" {
		t.Fatalf("got %+v, %v", g, err)
	}
	if _, ok := c.Guild("3"); !ok {
		t.Fatal("guild not written back")
	}
	if _, err := r.Guild(ctx, "3"); err != nil {
		t.Fatal(err)
	}
	m, err := r.Member(ctx, "3", "7")
	if err != nil || m.User.Username != "anna" {
		t.Fatalf("got %+v, %v", m, err)
	}
	msg, err := r.Message(ctx, "30", "50")
	if err != nil || msg.Content != "hi" {
		t.Fatalf("got %+v, %v", msg, err)
	}
	if _, err := r.Message(ctx, "30", "50"); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, req := range srv.Requests() {
		if strings.HasPrefix(req.Path, "/v1/guilds/3") && !strings.Contains(req.Path, "members") || strings.Contains(req.Path, "messages/50") {
			n++
		}
	}
	if n != 2 {
		t.Errorf("%d fetches for guild and message, want 2", n)
	}
	st := c.Stats()
	if st.Hits != 2 || st.Misses != 3 || st.Guilds != 1 || st.Roles != 2 || st.Messages != 1 {
		t.Errorf("stats %+v", st)
	}
}

func TestResolverCollapsesConcurrentFetches(t *testing.T) {
	srv, _, r := resolverSetup(t)
	srv.Handle("GET /v1/users/{user}", func(w http.ResponseWriter, r *http.Request) {
		fluxertest.WriteJSON(w, fluxer.User{ID: r.PathValue("user"), Username: "testbot"})
	})
	srv.SetDelay(50 * time.Millisecond)
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if u, err := r.User(context.Background(), "1000"); err != nil || u.ID != "1000" {
				t.Errorf("got %+v, %v", u, err)
			}
		})
	}
	wg.Wait()
	if n := len(srv.Requests()); n != 1 {
		t.Errorf("%d requests", n)
	}
}

func TestResolverPermissions(t *testing.T) {
	_, _, r := resolverSetup(t)
	p, err := r.Permissions(context.Background(), "3", "30", "7")
	if err != nil {
		t.Fatal(err)
	}
	if p.Has(fluxer.PermSendMessages) || !p.Has(fluxer.PermManageMessages|fluxer.PermViewChannel) {
		t.Errorf("got %s", p)
	}
	if _, err := r.Permissions(context.Background(), "3", "30", "8"); fluxer.ErrorCode(err) != "UNKNOWN_MEMBER" {
		t.Errorf("got %v", err)
	}
}
