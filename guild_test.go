package fluxer_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/fluxertest"
)

func guildSetup(t *testing.T) (*fluxertest.Server, *fluxer.Client) {
	t.Helper()
	srv, c := setup(t)
	srv.AddGuild(fluxer.Guild{
		ID: "3", Name: "Test", OwnerID: "1",
		Roles:    []fluxer.Role{{ID: "3", Name: "@everyone", Permissions: 0}},
		Channels: []fluxer.Channel{{ID: "30", Name: "general", Type: fluxer.ChannelText}},
	})
	srv.AddMember("3", fluxer.Member{User: &fluxer.User{ID: "1000", Username: "testbot"}})
	srv.AddMember("3", fluxer.Member{User: &fluxer.User{ID: "7", Username: "anna"}})
	srv.AddMember("3", fluxer.Member{User: &fluxer.User{ID: "8", Username: "ben"}})
	return srv, c
}

func TestGuild(t *testing.T) {
	_, c := guildSetup(t)
	ctx := context.Background()
	g, err := c.Guild(ctx, "3")
	if err != nil || g.Name != "Test" || len(g.Roles) != 1 {
		t.Fatalf("got %+v, %v", g, err)
	}
	gs, err := c.Guilds(ctx, fluxer.GuildQuery{Limit: 10})
	if err != nil || len(gs) != 1 || gs[0].ID != "3" {
		t.Errorf("got %+v, %v", gs, err)
	}
	if _, err := c.Guild(ctx, "4"); fluxer.ErrorCode(err) != "UNKNOWN_GUILD" {
		t.Errorf("got %v", err)
	}
	chs, err := c.GuildChannels(ctx, "3")
	if err != nil || len(chs) != 1 || chs[0].GuildID != "3" {
		t.Errorf("got %+v, %v", chs, err)
	}
}

func TestRoles(t *testing.T) {
	_, c := guildSetup(t)
	ctx := context.Background()
	role, err := c.CreateRole(ctx, "3", fluxer.NewRole{Name: "mods", Color: 0xff0000})
	if err != nil || role.ID == "" || role.Name != "mods" {
		t.Fatalf("got %+v, %v", role, err)
	}
	hoist := true
	if role, err = c.ModifyRole(ctx, "3", role.ID, fluxer.RoleEdit{Hoist: &hoist}); err != nil || !role.Hoist {
		t.Errorf("got %+v, %v", role, err)
	}
	if err := c.DeleteRole(ctx, "3", role.ID); err != nil {
		t.Fatal(err)
	}
	roles, err := c.Roles(ctx, "3")
	if err != nil || len(roles) != 1 {
		t.Errorf("got %+v, %v", roles, err)
	}
}

func TestMembers(t *testing.T) {
	srv, c := guildSetup(t)
	ctx := context.Background()
	ms, err := c.Members(ctx, "3", "7", 100)
	if err != nil || len(ms) != 2 || ms[0].User.ID != "8" || ms[1].User.ID != "1000" {
		t.Fatalf("got %+v, %v", ms, err)
	}
	if ms, err = c.Members(ctx, "3", "", 0); err != nil || len(ms) != 3 {
		t.Errorf("without a limit got %d members, %v", len(ms), err)
	}
	if err := c.AddMemberRole(ctx, "3", "7", "40"); err != nil {
		t.Fatal(err)
	}
	m, err := c.SetNickname(ctx, "3", "7", "Anna")
	if err != nil || m.DisplayName() != "Anna" || len(m.Roles) != 1 {
		t.Errorf("got %+v, %v", m, err)
	}
	until := time.Now().Add(time.Hour)
	if m, err = c.SetTimeout(ctx, "3", "7", until); err != nil || !m.TimedOut() {
		t.Errorf("got %+v, %v", m, err)
	}
	if m, err = c.SetTimeout(ctx, "3", "7", time.Time{}); err != nil || m.TimedOut() {
		t.Errorf("got %+v, %v", m, err)
	}
	if body := srv.Requests()[len(srv.Requests())-1].Body; strings.TrimSpace(body) != `{"communication_disabled_until":null}` {
		t.Errorf("body %s", body)
	}
	me, err := c.CurrentMember(ctx, "3")
	if err != nil || me.User.ID != "1000" {
		t.Errorf("got %+v, %v", me, err)
	}
	if err := c.RemoveMember(fluxer.WithReason(ctx, "bye"), "3", "8"); err != nil {
		t.Fatal(err)
	}
	if _, ok := srv.GuildMember("3", "8"); ok {
		t.Error("member still there")
	}
}

func TestBans(t *testing.T) {
	srv, c := guildSetup(t)
	ctx := context.Background()
	if err := c.Ban(ctx, "3", "7", fluxer.BanOptions{DeleteMessageSeconds: 3600, Reason: "spam"}); err != nil {
		t.Fatal(err)
	}
	bans, err := c.Bans(ctx, "3")
	if err != nil || len(bans) != 1 || bans[0].User.ID != "7" || bans[0].Reason != "spam" || !srv.Banned("3", "7") {
		t.Errorf("got %+v, %v", bans, err)
	}
	if err := c.Unban(ctx, "3", "7"); err != nil {
		t.Fatal(err)
	}
	var apiErr *fluxer.APIError
	if err := c.Unban(ctx, "3", "7"); !errors.As(err, &apiErr) || apiErr.Code != "USER_IS_NOT_BANNED" {
		t.Errorf("got %v", err)
	}
}

func TestWebhooks(t *testing.T) {
	srv, c := guildSetup(t)
	ctx := context.Background()
	hook, err := c.CreateWebhook(ctx, "30", "relay", "")
	if err != nil || hook.Token == "" || hook.GuildID != "3" {
		t.Fatalf("got %+v, %v", hook, err)
	}
	parsed, err := fluxer.ParseWebhookURL("https://web.fluxer.app/api/v1/webhooks/" + hook.ID + "/" + hook.Token)
	if err != nil || parsed.ID != hook.ID || parsed.Token != hook.Token {
		t.Fatalf("got %+v, %v", parsed, err)
	}
	anon := srv.Client()
	anon.Token = ""
	msg, err := anon.ExecuteWebhook(ctx, parsed, "", fluxer.WebhookMessage{MessageSend: fluxer.MessageSend{Content: "hi"}, Username: "Relay"}, nil)
	if err != nil || msg.WebhookID != hook.ID || msg.Author.Username != "Relay" {
		t.Fatalf("got %+v, %v", msg, err)
	}
	if got := srv.Messages("30"); len(got) != 1 || got[0].Content != "hi" {
		t.Errorf("got %+v", got)
	}
	if err := anon.DeleteWebhookWithToken(ctx, parsed); err != nil {
		t.Fatal(err)
	}
	if _, err := anon.WebhookWithToken(ctx, parsed); fluxer.ErrorCode(err) != "UNKNOWN_WEBHOOK" {
		t.Errorf("got %v", err)
	}
	if _, err := fluxer.ParseWebhookURL("https://web.fluxer.app/channels/1/2"); err == nil {
		t.Error("expected error")
	}
}

func TestCreateForum(t *testing.T) {
	_, c := guildSetup(t)
	order := fluxer.ForumSortCreationTime
	ch, err := c.CreateChannel(context.Background(), "3", fluxer.NewChannel{
		Name: "help", Type: fluxer.ChannelForum, Flags: fluxer.ChannelFlagRequireTag,
		AvailableTags:        []fluxer.ForumTag{{Name: "bug"}, {Name: "solved", Moderated: true}},
		DefaultReactionEmoji: &fluxer.DefaultReaction{EmojiName: "\U0001F44D"},
		DefaultSortOrder:     &order, DefaultForumLayout: fluxer.ForumLayoutList, DefaultTagSetting: fluxer.TagsMatchAll,
		DefaultThreadRateLimitPerUser: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ch.AvailableTags) != 2 || ch.AvailableTags[0].ID == "" || ch.DefaultSortOrder == nil || *ch.DefaultSortOrder != order ||
		ch.DefaultForumLayout != fluxer.ForumLayoutList || ch.DefaultTagSetting != fluxer.TagsMatchAll || ch.DefaultThreadRateLimitPerUser != 10 ||
		ch.Flags != fluxer.ChannelFlagRequireTag {
		t.Errorf("got %+v", ch)
	}
}

func TestSearchMembersByAccountAge(t *testing.T) {
	srv, c := setup(t)
	var body string
	srv.Handle("POST /v1/guilds/{guild}/members-search", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		_, _ = w.Write([]byte(`{"guild_id":"3","members":[{"id":"3:7","guild_id":"3","user_id":"7","username":"anna","discriminator":"0001","global_name":null,"nickname":null,
			"role_ids":[],"joined_at":1761300000,"is_bot":false,"supplemental":{"join_source_type":1,"source_invite_code":"abc","inviter_id":"9"}}],"page_result_count":1,"total_result_count":1,"indexing":false}`))
	})
	p, err := c.SearchMembers(context.Background(), "3", fluxer.MemberSearch{UserCreatedAfter: 1761200000, JoinSourceTypes: []int{fluxer.JoinInvite}})
	if err != nil {
		t.Fatal(err)
	}
	if body != `{"user_created_at_gte":1761200000,"join_source_type":[1]}` {
		t.Errorf("body %s", body)
	}
	if s := p.Members[0].Supplemental; s.JoinSourceType == nil || *s.JoinSourceType != fluxer.JoinInvite || *s.SourceInviteCode != "abc" || *s.InviterID != "9" {
		t.Errorf("got %+v", s)
	}
}
