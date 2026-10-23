package redis_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/state"
	"github.com/invaliduser231/fluxer-go/state/redis"
	goredis "github.com/redis/go-redis/v9"
)

func newStore(t *testing.T) (*miniredis.Miniredis, *redis.Store) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	s := redis.New(client, "test")
	s.OnError = func(err error) { t.Errorf("redis: %v", err) }
	return mr, s
}

func TestTable(t *testing.T) {
	_, s := newStore(t)
	ch := s.Channels()
	ch.Put("3", "30", fluxer.Channel{ID: "30", GuildID: "3", Name: "general"})
	ch.Put("3", "31", fluxer.Channel{ID: "31", GuildID: "3", Name: "voice"})
	ch.Put("", "90", fluxer.Channel{ID: "90", Type: fluxer.ChannelDM})
	if c, ok := ch.Get("3", "30"); !ok || c.Name != "general" {
		t.Errorf("got %+v, %v", c, ok)
	}
	if g, ok := ch.Locate("90"); !ok || g != "" {
		t.Errorf("locate %q, %v", g, ok)
	}
	if ch.Len("3") != 2 || !slices.Contains(slices.Collect(ch.Groups()), "") {
		t.Errorf("len %d, groups %v", ch.Len("3"), slices.Collect(ch.Groups()))
	}
	ch.Delete("3", "31")
	n := 0
	for range ch.Group("3") {
		n++
	}
	if n != 1 {
		t.Errorf("group has %d", n)
	}
	ch.DeleteGroup("3")
	if _, ok := ch.Locate("30"); ok || ch.Len("3") != 0 {
		t.Error("group not deleted")
	}
	if _, ok := ch.Get("3", "nope"); ok {
		t.Error("found a missing key")
	}
	if slices.Contains(slices.Collect(ch.Groups()), "3") {
		t.Error("empty group still listed")
	}
}

const guildCreate = `{"id":"3","properties":{"id":"3","name":"Test","owner_id":"1"},
	"roles":[{"id":"3","name":"@everyone","position":0,"permissions":"3072"}],
	"channels":[{"id":"30","type":0,"name":"general"}],"members":[{"user":{"id":"7","username":"anna"},"roles":[]}],
	"member_count":2,"voice_states":[]}`

func TestCachesShareRedis(t *testing.T) {
	mr, s := newStore(t)
	writer := state.New(state.WithStore(s))
	writer.Handle("READY", json.RawMessage(`{"user":{"id":"1000","username":"testbot"}}`))
	writer.Handle("GUILD_CREATE", json.RawMessage(guildCreate))
	writer.Handle("MESSAGE_CREATE", json.RawMessage(`{"id":"50","channel_id":"30","guild_id":"3","content":"hi","author":{"id":"7","username":"anna"}}`))

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer client.Close()
	reader := state.New(state.WithStore(redis.New(client, "test")))
	g, ok := reader.Guild("3")
	if !ok || g.Name != "Test" || len(g.Channels) != 1 || len(g.Roles) != 1 {
		t.Fatalf("got %+v, %v", g, ok)
	}
	if m, ok := reader.Member("3", "7"); !ok || m.User.Username != "anna" {
		t.Errorf("member %+v", m)
	}
	if m, ok := reader.MessageByID("50"); !ok || m.Content != "hi" {
		t.Errorf("message %+v", m)
	}
	if p, ok := reader.Permissions("3", "30", "7"); !ok || !p.Has(fluxer.PermSendMessages) {
		t.Errorf("permissions %s", p)
	}

	writer.Handle("GUILD_DELETE", json.RawMessage(`{"id":"3"}`))
	if _, ok := reader.Guild("3"); ok {
		t.Error("guild still visible after delete")
	}
	if err := s.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if keys := mr.Keys(); len(keys) != 0 {
		t.Errorf("keys left: %v", keys)
	}
}

func TestErrorsAreReported(t *testing.T) {
	mr, s := newStore(t)
	var got error
	s.OnError = func(err error) { got = err }
	mr.Close()
	s.Users().Put("", "1", fluxer.User{ID: "1", Username: "x"})
	if got == nil || errors.Is(got, goredis.Nil) {
		t.Errorf("got %v", got)
	}
}
