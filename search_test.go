package fluxer_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestMessagesBefore(t *testing.T) {
	srv, c := setup(t)
	for i := range 250 {
		srv.AddMessage(fluxer.Message{ID: strconv.Itoa(1000 + i), ChannelID: "1"})
	}
	n, last := 0, ""
	for m, err := range c.MessagesBefore(context.Background(), "1", "") {
		if err != nil {
			t.Fatal(err)
		}
		if last != "" && fluxer.CompareSnowflakes(m.ID, last) >= 0 {
			t.Fatalf("%s came after %s", m.ID, last)
		}
		n, last = n+1, m.ID
	}
	if n != 250 || len(srv.Requests()) != 3 {
		t.Errorf("got %d messages in %d requests", n, len(srv.Requests()))
	}
	n = 0
	for range c.MessagesBefore(context.Background(), "1", "1100") {
		n++
		if n == 10 {
			break
		}
	}
	if n != 10 {
		t.Errorf("stopped at %d", n)
	}
}

func TestAllMembers(t *testing.T) {
	_, c := guildSetup(t)
	var ids []string
	for m, err := range c.AllMembers(context.Background(), "3") {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, m.User.ID)
	}
	if len(ids) != 3 || ids[0] != "7" || ids[2] != "1000" {
		t.Errorf("got %v", ids)
	}
}

func TestSearchMessages(t *testing.T) {
	srv, c := setup(t)
	srv.AddMessage(fluxer.Message{ID: "1", ChannelID: "1", Content: "hello world"})
	srv.AddMessage(fluxer.Message{ID: "2", ChannelID: "2", Content: "Hello again"})
	srv.AddMessage(fluxer.Message{ID: "3", ChannelID: "2", Content: "bye"})
	r, err := c.SearchMessages(context.Background(), fluxer.MessageSearch{Content: "hello", ChannelIDs: []string{"2"}})
	if err != nil || r.Total != 1 || r.Messages[0].ID != "2" {
		t.Errorf("got %+v, %v", r, err)
	}
}

func TestSearchMessagesExcludes(t *testing.T) {
	srv, c := setup(t)
	everyone := false
	_, err := c.SearchMessages(context.Background(), fluxer.MessageSearch{
		Content: "deploy", AuthorTypes: []string{"bot"}, ExcludeChannelIDs: []string{"9"}, MentionEveryone: &everyone,
		ExcludeAttachmentExtensions: []string{"exe"}, IncludeNSFW: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	reqs := srv.Requests()
	want := `{"content":"deploy","exclude_channel_id":["9"],"author_type":["bot"],"mention_everyone":false,"exclude_attachment_extension":["exe"],"include_nsfw":true}`
	if body := reqs[len(reqs)-1].Body; body != want {
		t.Errorf("body %s", body)
	}
}
