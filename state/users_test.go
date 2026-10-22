package state_test

import (
	"encoding/json"
	"testing"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/state"
)

func TestPartialMembersAreCompleted(t *testing.T) {
	c := setup(t)
	if m, _ := c.Member("3", "1000"); m.User.Username != "testbot" || c.MemberPartial("3", "1000") {
		t.Errorf("own member %+v", m.User)
	}
	if !c.MemberPartial("3", "7") {
		t.Error("member 7 should be partial")
	}
	feed(t, c, "MESSAGE_CREATE", fluxer.MessageCreate{Message: fluxer.Message{
		ID: "99", ChannelID: "30", GuildID: "3", Author: fluxer.User{ID: "7", Username: "anna"},
		Member: &fluxer.Member{Roles: []string{"40"}},
	}})
	if m, _ := c.Member("3", "7"); m.User.Username != "anna" || c.MemberPartial("3", "7") {
		t.Errorf("member 7 %+v", m.User)
	}
	if u, ok := c.User("7"); !ok || u.Username != "anna" {
		t.Errorf("user %+v", u)
	}
}

func TestPresences(t *testing.T) {
	c := state.New(state.WithMemberPolicy(state.MembersOnline()))
	ready(c)
	c.Handle("GUILD_CREATE", json.RawMessage(guildCreate))
	feed(t, c, "PRESENCE_UPDATE_BULK", fluxer.PresenceUpdateBulk{GuildID: "3", Presences: []fluxer.Presence{
		{User: fluxer.User{ID: "8", Username: "ben"}, Status: fluxer.StatusOnline},
		{User: fluxer.User{ID: "9"}, Status: fluxer.StatusIdle},
	}})
	feed(t, c, "GUILD_MEMBER_ADD", fluxer.Member{GuildID: "3", User: &fluxer.User{ID: "8"}})
	if m, ok := c.Member("3", "8"); !ok || m.User.Username != "ben" {
		t.Errorf("online member %+v, %v", m, ok)
	}
	feed(t, c, "PRESENCE_UPDATE", fluxer.Presence{GuildID: "3", User: fluxer.User{ID: "9"}, Status: fluxer.StatusOffline})
	if ps := c.Presences("3"); len(ps) != 1 || ps[0].User.ID != "8" {
		t.Errorf("presences %+v", ps)
	}
	feed(t, c, "GUILD_COUNTS_UPDATE", fluxer.GuildCountsUpdate{Counts: []fluxer.GuildCount{{GuildID: "3", MemberCount: 40, OnlineCount: 12}}})
	if g, _ := c.Guild("3"); g.MemberCount != 40 || g.OnlineCount != 12 {
		t.Errorf("counts %d %d", g.MemberCount, g.OnlineCount)
	}
}
