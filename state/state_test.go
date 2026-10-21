package state_test

import (
	"encoding/json"
	"testing"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/state"
)

func feed(t *testing.T, c *state.Cache, event string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	c.Handle(event, data)
}

const guildCreate = `{"id":"3","properties":{"id":"3","name":"Test","owner_id":"1"},
	"roles":[{"id":"3","name":"@everyone","position":0,"permissions":"3072"},{"id":"40","name":"mods","position":1,"permissions":"8192"}],
	"channels":[{"id":"31","type":2,"name":"voice","position":1},{"id":"30","type":0,"name":"general","position":0,
		"permission_overwrites":[{"id":"3","type":0,"allow":"0","deny":"2048"},{"id":"40","type":0,"allow":"2048","deny":"0"}]}],
	"emojis":[{"id":"60","name":"coin"}],"stickers":[],"members":[{"user":{"id":"1000"},"roles":[]},{"user":{"id":"7"},"roles":["40"]}],
	"member_count":2,"online_count":1,"voice_states":[{"guild_id":"3","channel_id":"31","user_id":"7","connection_id":"c1","version":1}],
	"threads":[{"id":"50","type":11,"name":"talk","parent_id":"30"}]}`

func setup(t *testing.T) *state.Cache {
	t.Helper()
	c := state.New()
	c.Handle("READY", json.RawMessage(`{"session_id":"s","user":{"id":"1000","username":"testbot"},"guilds":[{"id":"3","unavailable":true}],"private_channels":[{"id":"90","type":1}]}`))
	if !c.Unavailable("3") {
		t.Fatal("guild not marked unavailable")
	}
	c.Handle("GUILD_CREATE", json.RawMessage(guildCreate))
	return c
}

func TestGuildCreate(t *testing.T) {
	c := setup(t)
	g, ok := c.Guild("3")
	if !ok || g.Name != "Test" || len(g.Roles) != 2 || g.Roles[1].ID != "40" || len(g.Channels) != 2 || g.Channels[0].ID != "30" || g.MemberCount != 2 {
		t.Fatalf("got %+v", g)
	}
	if c.Self().Username != "testbot" {
		t.Errorf("self %+v", c.Self())
	}
	if _, ok := c.Channel("90"); !ok {
		t.Error("private channel missing")
	}
	if th := c.Threads("30"); len(th) != 1 || th[0].GuildID != "3" {
		t.Errorf("threads %+v", th)
	}
	if v := c.VoiceChannel("3", "7"); v != "31" {
		t.Errorf("voice %q", v)
	}
}

func TestPermissions(t *testing.T) {
	c := setup(t)
	if p, ok := c.Permissions("3", "30", "1000"); !ok || p.Has(fluxer.PermSendMessages) || !p.Has(fluxer.PermViewChannel) {
		t.Errorf("bot got %s", p)
	}
	if p, ok := c.Permissions("3", "50", "7"); !ok || !p.Has(fluxer.PermSendMessages|fluxer.PermManageMessages) {
		t.Errorf("mod in thread got %s", p)
	}
	if _, ok := c.Permissions("3", "30", "8"); ok {
		t.Error("unknown member has permissions")
	}
}

func TestUpdates(t *testing.T) {
	c := setup(t)
	feed(t, c, "GUILD_ROLE_UPDATE", fluxer.GuildRoleUpdate{GuildID: "3", Role: fluxer.Role{ID: "40", Name: "moderators", Position: 1}})
	feed(t, c, "GUILD_ROLE_DELETE", fluxer.GuildRoleDelete{GuildID: "3", RoleID: "3"})
	if r, _ := c.Role("3", "40"); r.Name != "moderators" {
		t.Errorf("role %+v", r)
	}
	if _, ok := c.Role("3", "3"); ok {
		t.Error("role not deleted")
	}

	feed(t, c, "CHANNEL_CREATE", fluxer.Channel{ID: "32", GuildID: "3", Type: fluxer.ChannelText, Name: "new", Position: 2})
	feed(t, c, "CHANNEL_UPDATE", fluxer.Channel{ID: "30", GuildID: "3", Name: "lobby"})
	feed(t, c, "MESSAGE_CREATE", fluxer.Message{ID: "99", ChannelID: "32"})
	if ch, _ := c.Channel("32"); ch.LastMessageID != "99" {
		t.Errorf("last message %+v", ch)
	}
	feed(t, c, "CHANNEL_DELETE", fluxer.Channel{ID: "30", GuildID: "3"})
	if _, ok := c.Channel("50"); ok {
		t.Error("thread of a deleted channel kept")
	}
	if chs := c.GuildChannels("3"); len(chs) != 2 {
		t.Errorf("channels %+v", chs)
	}

	feed(t, c, "GUILD_MEMBER_ADD", fluxer.Member{GuildID: "3", User: &fluxer.User{ID: "8", Username: "ben"}})
	feed(t, c, "GUILD_MEMBER_UPDATE", map[string]any{"guild_id": "3", "user": map[string]string{"id": "8"}, "roles": []string{"40"}, "nick": "Benny"})
	m, ok := c.Member("3", "8")
	if !ok || m.User.Username != "ben" || m.DisplayName() != "Benny" || len(m.Roles) != 1 {
		t.Errorf("member %+v %+v", m, m.User)
	}
	feed(t, c, "GUILD_MEMBER_REMOVE", fluxer.GuildMemberRemove{GuildID: "3", User: fluxer.User{ID: "7"}})
	if g, _ := c.Guild("3"); g.MemberCount != 2 || len(c.Members("3")) != 2 {
		t.Errorf("count %d, members %d", g.MemberCount, len(c.Members("3")))
	}

	feed(t, c, "VOICE_STATE_UPDATE", fluxer.VoiceState{GuildID: "3", UserID: "7", ConnectionID: "c1", Version: 2})
	if len(c.VoiceStates("3")) != 0 {
		t.Error("voice state not removed")
	}
	feed(t, c, "GUILD_EMOJIS_UPDATE", fluxer.GuildEmojisUpdate{GuildID: "3"})
	feed(t, c, "GUILD_DELETE", fluxer.GuildDelete{ID: "3", Unavailable: true})
	if _, ok := c.Guild("3"); ok || !c.Unavailable("3") {
		t.Error("guild still available")
	}
	feed(t, c, "GUILD_DELETE", fluxer.GuildDelete{ID: "3"})
	if _, ok := c.Channel("32"); ok || len(c.Guilds()) != 0 {
		t.Error("guild data kept after leaving")
	}
}

func TestSkipMembers(t *testing.T) {
	c := state.New()
	c.SkipMembers = true
	c.Handle("READY", json.RawMessage(`{"user":{"id":"1000"}}`))
	c.Handle("GUILD_CREATE", json.RawMessage(guildCreate))
	if ms := c.Members("3"); len(ms) != 1 || ms[0].User.ID != "1000" {
		t.Errorf("got %+v", ms)
	}
}
