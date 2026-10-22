package state_test

import (
	"testing"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/state"
)

func TestOldValues(t *testing.T) {
	c := setup(t)
	var gotRole, gotNick, gotContent, gotChannel string
	var deleted []string
	state.OnChange(c, func(old *fluxer.Role, e *fluxer.GuildRoleUpdate) {
		if old != nil {
			gotRole = old.Name + " -> " + e.Role.Name
		}
	})
	state.OnChange(c, func(old *fluxer.Member, e *fluxer.GuildMemberUpdate) {
		if old != nil && old.Nick == nil {
			gotNick = e.DisplayName()
		}
	})
	state.OnChange(c, func(old *fluxer.Message, e *fluxer.MessageUpdate) {
		if old != nil {
			gotContent = old.Content + " -> " + e.Content
		}
	})
	state.OnChange(c, func(old *fluxer.Channel, e *fluxer.ChannelDelete) {
		if old != nil {
			gotChannel = old.Name
		}
	})
	remove := state.OnChange(c, func(old []fluxer.Message, e *fluxer.MessageDeleteBulk) {
		for _, m := range old {
			deleted = append(deleted, m.Content)
		}
	})

	feed(t, c, "GUILD_ROLE_UPDATE", fluxer.GuildRoleUpdate{GuildID: "3", Role: fluxer.Role{ID: "40", Name: "moderators"}})
	nick := "Anna"
	feed(t, c, "GUILD_MEMBER_UPDATE", fluxer.Member{GuildID: "3", User: &fluxer.User{ID: "7", Username: "anna"}, Nick: &nick})
	feed(t, c, "MESSAGE_CREATE", message("20", "30", "draft"))
	feed(t, c, "MESSAGE_UPDATE", fluxer.MessageUpdate{Message: fluxer.Message{ID: "20", ChannelID: "30", GuildID: "3", Content: "final", Author: fluxer.User{ID: "7"}}})
	feed(t, c, "MESSAGE_CREATE", message("21", "30", "a"))
	feed(t, c, "MESSAGE_CREATE", message("22", "30", "b"))
	feed(t, c, "MESSAGE_DELETE_BULK", fluxer.MessageDeleteBulk{ChannelID: "30", IDs: []string{"21", "22", "99"}})
	remove()
	feed(t, c, "MESSAGE_DELETE_BULK", fluxer.MessageDeleteBulk{ChannelID: "30", IDs: []string{"20"}})
	feed(t, c, "CHANNEL_DELETE", fluxer.Channel{ID: "30", GuildID: "3"})

	if gotRole != "mods -> moderators" || gotNick != "Anna" {
		t.Errorf("role %q, nick %q", gotRole, gotNick)
	}
	if gotContent != "draft -> final" || gotChannel != "general" || len(deleted) != 2 {
		t.Errorf("content %q, channel %q, deleted %v", gotContent, gotChannel, deleted)
	}
}

func TestOldValueMissing(t *testing.T) {
	c := state.New()
	called := false
	state.OnChange(c, func(old *fluxer.Message, e *fluxer.MessageDelete) {
		called = true
		if old != nil {
			t.Errorf("got %+v", old)
		}
	})
	feed(t, c, "MESSAGE_DELETE", fluxer.MessageDelete{ID: "1", ChannelID: "2"})
	if !called {
		t.Error("handler not called")
	}
}
