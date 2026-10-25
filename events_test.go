package fluxer_test

import (
	"encoding/json"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestDecodeMessageCreate(t *testing.T) {
	raw := `{"id":"5","channel_id":"2","guild_id":"3","channel_type":0,"author":{"id":"7","username":"anna","discriminator":"0420","global_name":"Anna","avatar":null,"flags":0},
		"member":{"roles":["40"],"joined_at":"2026-01-02T03:04:05Z","mute":false,"deaf":false},"type":0,"flags":0,"content":"!ping","timestamp":"2026-10-01T10:00:00.123Z",
		"pinned":false,"mention_everyone":false,"tts":false,"mentions":[],"mention_roles":[],"mention_here":true}`
	e, err := fluxer.Decode("MESSAGE_CREATE", json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	m, ok := e.(*fluxer.MessageCreate)
	if !ok || m.Content != "!ping" || m.Author.DisplayName() != "Anna" || m.Member.Roles[0] != "40" || !m.MentionHere || m.Timestamp.IsZero() {
		t.Errorf("got %#v", e)
	}
	if m.EventName() != "MESSAGE_CREATE" {
		t.Errorf("name %s", m.EventName())
	}
}

func TestDecodeGuildCreate(t *testing.T) {
	raw := `{"id":"3","properties":{"id":"3","name":"Test","owner_id":"1","features":["DISCOVERABLE"]},
		"roles":[{"id":"3","name":"@everyone","color":0,"position":0,"permissions":"1024","hoist":false,"mentionable":false}],
		"channels":[{"id":"30","type":0,"name":"general"}],"emojis":[],"stickers":[],"members":[{"user":{"id":"1000"},"roles":[]}],
		"member_count":12,"online_count":4,"presences":[],"voice_states":[{"guild_id":"3","channel_id":"31","user_id":"7","connection_id":"c1","mute":false,"deaf":false,"self_mute":true,"self_deaf":false,"self_video":false,"self_stream":false,"is_mobile":false,"suppress":false,"version":3}],
		"joined_at":"2026-09-30T19:03:21Z","unavailable":false}`
	e, err := fluxer.Decode("GUILD_CREATE", json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	g := e.(*fluxer.GuildCreate)
	full := g.Guild()
	if full.ID != "3" || full.Name != "Test" || len(full.Roles) != 1 || full.MemberCount != 12 || !full.HasFeature("DISCOVERABLE") {
		t.Errorf("got %+v", full)
	}
	if len(g.VoiceStates) != 1 || !g.VoiceStates[0].SelfMute || g.VoiceStates[0].ConnectionID != "c1" {
		t.Errorf("got %+v", g.VoiceStates)
	}
}

func TestDecodeReactionEmoji(t *testing.T) {
	e, err := fluxer.Decode("MESSAGE_REACTION_ADD", json.RawMessage(`{"user_id":"7","channel_id":"2","message_id":"5","emoji":{"name":"coin","id":"123","animated":true}}`))
	if err != nil {
		t.Fatal(err)
	}
	r := e.(*fluxer.MessageReactionAdd)
	if r.Emoji.String() != "<a:coin:123>" || r.Emoji.APIName() != "coin:123" || r.Emoji.Animated == nil || !*r.Emoji.Animated {
		t.Errorf("got %+v", r.Emoji)
	}
}

func TestDecodeAuditLogEntry(t *testing.T) {
	raw := `{"guild_id":"3","id":"77","action_type":22,"user_id":"1","target_id":"7","reason":"spam","options":{"delete_message_seconds":3600},
		"changes":[{"key":"nick","old_value":"a","new_value":null}]}`
	e, err := fluxer.Decode("GUILD_AUDIT_LOG_ENTRY_CREATE", json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	a := e.(*fluxer.GuildAuditLogEntryCreate)
	if a.GuildID != "3" || a.ActionType != fluxer.AuditMemberBanAdd || a.Options["delete_message_seconds"] != 3600.0 || string(a.Changes[0].OldValue) != `"a"` {
		t.Errorf("got %+v", a)
	}
}

func TestDecodeUnknownAndEmpty(t *testing.T) {
	if e, err := fluxer.Decode("SOMETHING_NEW", json.RawMessage(`{}`)); e != nil || err != nil {
		t.Errorf("got %v, %v", e, err)
	}
	if e, err := fluxer.Decode("RESUMED", nil); err != nil || e.EventName() != "RESUMED" {
		t.Errorf("got %v, %v", e, err)
	}
	if _, err := fluxer.Decode("MESSAGE_DELETE", json.RawMessage(`{"id":5}`)); err == nil {
		t.Error("expected error for a numeric id")
	}
}

func TestDecodeMemberLists(t *testing.T) {
	raw := `{"guild_id":"3","id":"30","member_count":2,"online_count":1,"groups":[{"id":"online","count":1},{"id":"offline","count":1}],
		"ops":[{"op":"SYNC","range":[0,3],"items":[{"group":{"id":"online","count":1}},{"member":{"user":{"id":"7"},"roles":[],"presence":{"status":"online","mobile":false,"afk":false}}},
		{"group":{"id":"offline","count":1}},{"member":{"user":{"id":"8"},"roles":[],"presence":{"status":"offline","mobile":false,"afk":false}}}]}]}`
	e, err := fluxer.Decode("GUILD_MEMBER_LIST_UPDATE", json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	l := e.(*fluxer.GuildMemberListUpdate)
	if len(l.Ops) != 1 || l.Ops[0].Range != [2]int{0, 3} || len(l.Ops[0].Items) != 4 || l.Ops[0].Items[1].Member.User.ID != "7" ||
		l.Ops[0].Items[1].Member.Presence.Status != "online" || l.Ops[0].Items[2].Group.ID != "offline" {
		t.Errorf("got %+v", l)
	}

	raw = `{"guild_id":"3","thread_id":"50","members":[{"user_id":"7","join_timestamp":"2026-10-24T10:00:00Z","flags":0,"member":{"user":{"id":"7"},"roles":[]},"presence":null}]}`
	e, err = fluxer.Decode("THREAD_MEMBER_LIST_UPDATE", json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	if tl := e.(*fluxer.ThreadMemberListUpdate); tl.ThreadID != "50" || len(tl.Members) != 1 || tl.Members[0].UserID != "7" || tl.Members[0].Member == nil {
		t.Errorf("got %+v", tl)
	}
}

func TestDecodeGuildHealth(t *testing.T) {
	e, err := fluxer.Decode("GUILD_HEALTH_UPDATE", json.RawMessage(`{"guild_id":"3","degraded":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if h := e.(*fluxer.GuildHealthUpdate); h.GuildID != "3" || !h.Degraded {
		t.Errorf("got %+v", h)
	}
}

func TestDecodeAttachment(t *testing.T) {
	raw := `{"id":"5","channel_id":"2","author":{"id":"7"},"content":"","attachments":[{"id":"9","filename":"memo.ogg","size":120,"flags":8,
		"content_hash":"h1","placeholder":"cGg=","nsfw":false,"duration":4,"waveform":"AAEC","expires_at":"2026-10-25T10:00:00Z","expired":false}]}`
	e, err := fluxer.Decode("MESSAGE_CREATE", json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	a := e.(*fluxer.MessageCreate).Attachments[0]
	if a.Flags&fluxer.AttachmentSpoiler == 0 || a.ContentHash != "h1" || a.Waveform != "AAEC" || a.Duration != 4 || a.ExpiresAt == nil || a.ExpiresAt.Day() != 25 {
		t.Errorf("got %+v", a)
	}
}

func TestDecodeForumChannel(t *testing.T) {
	raw := `{"id":"40","guild_id":"3","type":15,"name":"help","default_sort_order":1,"default_forum_layout":2,"default_tag_setting":"match_all",
		"default_thread_rate_limit_per_user":30,"nsfw":true,"nsfw_override":null,"content_warning_level":1,"voice_connection_limit":2,"rtc_p2p":true}`
	e, err := fluxer.Decode("CHANNEL_CREATE", json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	c := e.(*fluxer.ChannelCreate)
	if c.DefaultSortOrder == nil || *c.DefaultSortOrder != fluxer.ForumSortCreationTime || c.DefaultForumLayout != fluxer.ForumLayoutGrid ||
		c.DefaultTagSetting != fluxer.TagsMatchAll || c.DefaultThreadRateLimitPerUser != 30 || c.NSFWOverride != nil ||
		c.ContentWarningLevel != fluxer.ContentWarningShow || !c.RTCP2P {
		t.Errorf("got %+v", c.Channel)
	}
}

func TestDecodeGuildUpdate(t *testing.T) {
	raw := `{"id":"3","name":"Test","owner_id":"1","threads_active":true,"content_warning_level":1,"content_warning_text":"Spoilers",
		"message_history_cutoff":"2026-01-01T00:00:00Z","banner_width":960,"splash_card_alignment":2,"disabled_operations":16}`
	e, err := fluxer.Decode("GUILD_UPDATE", json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	g := e.(*fluxer.GuildUpdate)
	if !g.ThreadsActive || g.ContentWarningText != "Spoilers" || g.MessageHistoryCutoff == nil || g.BannerWidth != 960 ||
		g.SplashCardAlignment != fluxer.SplashRight || g.DisabledOperations&fluxer.OperationSendMessage == 0 {
		t.Errorf("got %+v", g.Guild)
	}
}
