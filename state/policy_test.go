package state_test

import (
	"encoding/json"
	"testing"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/state"
)

func ready(c *state.Cache) {
	c.Handle("READY", json.RawMessage(`{"user":{"id":"1000","username":"testbot"}}`))
}

func TestMembersInVoicePolicy(t *testing.T) {
	c := state.New(state.WithMemberPolicy(state.MembersInVoice()))
	ready(c)
	c.Handle("GUILD_CREATE", json.RawMessage(guildCreate))
	if ms := c.Members("3"); len(ms) != 2 {
		t.Fatalf("got %d members", len(ms))
	}
	feed(t, c, "GUILD_MEMBER_ADD", fluxer.Member{GuildID: "3", User: &fluxer.User{ID: "8", Username: "ben"}})
	if _, ok := c.Member("3", "8"); ok {
		t.Error("member outside voice cached")
	}
	feed(t, c, "VOICE_STATE_UPDATE", fluxer.VoiceState{GuildID: "3", UserID: "7", ConnectionID: "c1", Version: 2})
	if _, ok := c.Member("3", "7"); ok {
		t.Error("member kept after leaving voice")
	}
	if _, ok := c.Member("3", "1000"); !ok {
		t.Error("own member dropped")
	}
}

func TestFlagsAndChannelPolicy(t *testing.T) {
	c := state.New(
		state.WithoutFlags(state.FlagEmojis|state.FlagThreads),
		state.WithChannelPolicy(state.ChannelTypes(fluxer.ChannelText)),
	)
	ready(c)
	c.Handle("GUILD_CREATE", json.RawMessage(guildCreate))
	g, _ := c.Guild("3")
	if len(g.Emojis) != 0 || len(g.Channels) != 1 || g.Channels[0].ID != "30" {
		t.Errorf("got emojis %v, channels %v", g.Emojis, g.Channels)
	}
	if len(c.Threads("30")) != 0 {
		t.Error("threads cached")
	}
	if c.Flags().Has(state.FlagEmojis) || !c.Flags().Has(state.FlagRoles) {
		t.Errorf("flags %b", c.Flags())
	}
}

func TestPolicyCombinators(t *testing.T) {
	inGuild := state.MembersInGuilds("3")
	mod := state.MembersWithRole("40")
	m := fluxer.Member{GuildID: "3", Roles: []string{"40"}, User: &fluxer.User{ID: "7"}}
	other := fluxer.Member{GuildID: "4", User: &fluxer.User{ID: "8"}}
	c := state.New()
	if !inGuild.And(mod)(c, m) || inGuild.And(mod)(c, other) || !inGuild.Or(mod)(c, m) || !mod.Not()(c, other) {
		t.Error("combinators")
	}
	if !state.All[fluxer.Member]()(c, other) || state.None[fluxer.Member]()(c, m) {
		t.Error("all or none")
	}
}

func TestSweepMembers(t *testing.T) {
	policy := state.All[fluxer.Member]()
	c := state.New(state.WithMemberPolicy(func(c *state.Cache, m fluxer.Member) bool { return policy(c, m) }))
	ready(c)
	c.Handle("GUILD_CREATE", json.RawMessage(guildCreate))
	policy = state.None[fluxer.Member]()
	if n := c.SweepMembers(); n != 1 || len(c.Members("3")) != 1 {
		t.Errorf("swept %d, left %d", n, len(c.Members("3")))
	}
}
