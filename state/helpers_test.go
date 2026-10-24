package state_test

import (
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestHierarchyAndLookups(t *testing.T) {
	c := setup(t)
	feed(t, c, "GUILD_ROLE_CREATE", fluxer.GuildRoleCreate{GuildID: "3", Role: fluxer.Role{ID: "41", Name: "Admins", Position: 2}})
	feed(t, c, "GUILD_MEMBER_ADD", fluxer.Member{GuildID: "3", User: &fluxer.User{ID: "8", Username: "ben", Discriminator: "0042"}, Roles: []string{"41"}})
	feed(t, c, "GUILD_MEMBER_UPDATE", fluxer.Member{GuildID: "3", User: &fluxer.User{ID: "7", Username: "anna"}, Roles: []string{"40"}})

	if r, ok := c.HighestRole("3", "8"); !ok || r.ID != "41" {
		t.Errorf("highest %+v", r)
	}
	if !c.CanManageMember("3", "8", "7") || c.CanManageMember("3", "7", "8") || c.CanManageMember("3", "8", "1") {
		t.Error("member hierarchy")
	}
	if !c.CanManageMember("3", "1", "8") || !c.CanManageRole("3", "8", "40") || c.CanManageRole("3", "7", "41") {
		t.Error("owner or role hierarchy")
	}
	if m, ok := c.FindMember("3", "@BEN#0042"); !ok || m.User.ID != "8" {
		t.Errorf("by tag %+v", m)
	}
	if _, ok := c.FindMember("3", " "); ok {
		t.Error("empty name matched")
	}
	if m, ok := c.FindMember("3", "an"); !ok || m.User.ID != "7" {
		t.Errorf("by prefix %+v", m)
	}
	if r, ok := c.RoleByName("3", "admins"); !ok || r.ID != "41" {
		t.Errorf("role by name %+v", r)
	}
	if ch, ok := c.ChannelByName("3", "#General"); !ok || ch.ID != "30" {
		t.Errorf("channel by name %+v", ch)
	}
	if e, ok := c.EmojiByName("3", "coin"); !ok || e.ID != "60" {
		t.Errorf("emoji %+v", e)
	}
	if ms := c.VoiceMembers("3", "31"); len(ms) != 1 || ms[0].User.ID != "7" {
		t.Errorf("voice members %+v", ms)
	}
	if m, ok := c.SelfMember("3"); !ok || m.User.ID != "1000" {
		t.Errorf("self %+v", m)
	}
	n := 0
	for range c.EachMember("3") {
		n++
	}
	if n != 3 {
		t.Errorf("iterated %d members", n)
	}
	for g := range c.EachGuild() {
		if g.ID != "3" {
			t.Errorf("guild %s", g.ID)
		}
	}
}
