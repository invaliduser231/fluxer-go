package fluxer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestReactionsPage(t *testing.T) {
	srv, c := setup(t)
	srv.Handle("GET /v1/channels/{channel}/messages/{message}/reactions/{emoji}/users", func(w http.ResponseWriter, r *http.Request) {
		after := r.URL.Query().Get("after")
		page := fluxer.ReactionPage{Items: []fluxer.User{{ID: "7"}, {ID: "8"}}, HasMore: true, NextAfter: "8"}
		if after == "8" {
			page = fluxer.ReactionPage{Items: []fluxer.User{{ID: "9"}}}
		}
		_ = json.NewEncoder(w).Encode(page)
	})
	ctx := context.Background()
	var ids []string
	after := ""
	for {
		p, err := c.ReactionsPage(ctx, "1", "2", "\U0001FA99", after, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, u := range p.Items {
			ids = append(ids, u.ID)
		}
		if !p.HasMore {
			break
		}
		after = p.NextAfter
	}
	if strings.Join(ids, ",") != "7,8,9" {
		t.Errorf("got %v", ids)
	}
	if path := srv.Requests()[0].Path; path != "/v1/channels/1/messages/2/reactions/%F0%9F%AA%99/users" {
		t.Errorf("path %s", path)
	}
}

func TestChannelExtras(t *testing.T) {
	srv, c := setup(t)
	srv.Handle("GET /v1/channels/{channel}/follower-stats", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(fluxer.FollowerStats{ChannelCount: 4, GuildCount: 3})
	})
	srv.Handle("GET /v1/channels/{channel}/messages/{message}/crosspost-source", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"guild":{"id":"9","name":"News"}}`))
	})
	srv.Handle("POST /v1/channels/{channel}/post-data", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"threads":{"50":{"owner":{"user":{"id":"7"},"roles":[]},"first_message":{"id":"50","channel_id":"50","content":"hi","author":{"id":"7"}}}}}`))
	})
	noContent := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }
	accepted := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusAccepted) }
	srv.Handle("POST /v1/channels/{channel}/pins/ack", noContent)
	srv.Handle("DELETE /v1/channels/{channel}/messages/ack", noContent)
	srv.Handle("POST /v1/channels/{channel}/messages/bulk-delete-mine", accepted)
	srv.Handle("POST /v1/users/@me/guilds/{guild}/messages/bulk-delete-mine", accepted)

	ctx := context.Background()
	if s, err := c.FollowerStats(ctx, "1"); err != nil || s.GuildCount != 3 {
		t.Errorf("got %+v, %v", s, err)
	}
	if s, err := c.CrosspostSource(ctx, "1", "2"); err != nil || s.Guild.Name != "News" {
		t.Errorf("got %+v, %v", s, err)
	}
	posts, err := c.PostData(ctx, "40", []string{"50"})
	if err != nil || posts["50"].FirstMessage.Content != "hi" || posts["50"].Owner.User.ID != "7" {
		t.Errorf("got %+v, %v", posts, err)
	}
	for _, err := range []error{c.AckPins(ctx, "1"), c.ClearReadState(ctx, "1"), c.DeleteOwnMessages(ctx, "1"), c.DeleteOwnGuildMessages(ctx, "3")} {
		if err != nil {
			t.Error(err)
		}
	}
}

func TestSearchThreads(t *testing.T) {
	srv, c := setup(t)
	var query string
	srv.Handle("GET /v1/channels/{channel}/threads/search", func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"threads":[{"id":"51","type":11,"name":"login loop"}],"members":[{"id":"51","user_id":"1000","join_timestamp":"2026-10-24T10:00:00Z","flags":0,"muted":true,"mute_config":{"end_time":null,"selected_time_window":3600}}],
			"has_more":false,"total_results":1,"first_messages":[{"id":"51","channel_id":"51","author":{"id":"7"},"content":"it loops"}]}`))
	})
	r, err := c.SearchThreads(context.Background(), "40", fluxer.ThreadSearch{
		Name: "login", Tags: []string{"1", "2"}, TagSetting: fluxer.TagsMatchAll, SortBy: fluxer.ThreadSortCreated, SortOrder: "asc", AfterID: "50",
	})
	if err != nil {
		t.Fatal(err)
	}
	if query != "min_id=50&name=login&sort_by=creation_time&sort_order=asc&tag=1&tag=2&tag_setting=match_all" {
		t.Errorf("query %s", query)
	}
	if r.TotalResults != 1 || len(r.FirstMessages) != 1 || r.FirstMessages[0].Content != "it loops" || !r.Members[0].Muted || r.Members[0].MuteConfig.SelectedTimeWindow != 3600 {
		t.Errorf("got %+v", r)
	}
}
