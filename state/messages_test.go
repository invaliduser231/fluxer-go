package state_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/state"
)

func message(id, channel, content string) fluxer.MessageCreate {
	return fluxer.MessageCreate{Message: fluxer.Message{ID: id, ChannelID: channel, GuildID: "3", Content: content, Author: fluxer.User{ID: "7", Username: "anna"}}}
}

func TestMessageCacheLimitAndHistory(t *testing.T) {
	var evicted []string
	c := state.New(state.WithMessages(3, 0), state.WithMessageEvict(func(m fluxer.Message) { evicted = append(evicted, m.ID) }))
	ready(c)
	for i := 10; i < 15; i++ {
		feed(t, c, "MESSAGE_CREATE", message(strconv.Itoa(i), "30", "m"+strconv.Itoa(i)))
	}
	got := c.Messages("30", fluxer.MessageQuery{})
	if len(got) != 3 || got[0].ID != "14" || got[2].ID != "12" {
		t.Fatalf("got %v", ids(got))
	}
	if len(evicted) != 2 || evicted[0] != "10" {
		t.Errorf("evicted %v", evicted)
	}
	if got := c.Messages("30", fluxer.MessageQuery{Before: "14", Limit: 1}); len(got) != 1 || got[0].ID != "13" {
		t.Errorf("before got %v", ids(got))
	}
	if got := c.Messages("30", fluxer.MessageQuery{After: "12"}); len(got) != 2 || got[0].ID != "13" {
		t.Errorf("after got %v", ids(got))
	}
	if m, ok := c.MessageByID("13"); !ok || m.ChannelID != "30" {
		t.Errorf("by id %+v", m)
	}
}

func TestMessageUpdatesAndReactions(t *testing.T) {
	c := setup(t)
	feed(t, c, "MESSAGE_CREATE", message("20", "30", "hello"))
	feed(t, c, "MESSAGE_UPDATE", fluxer.MessageUpdate{Message: fluxer.Message{ID: "20", ChannelID: "30", GuildID: "3", Content: "hello!", Author: fluxer.User{ID: "7"}}})
	coin := fluxer.ReactionEmoji{Name: "coin", ID: "60"}
	feed(t, c, "MESSAGE_REACTION_ADD", fluxer.MessageReactionAdd{ChannelID: "30", MessageID: "20", UserID: "7", Emoji: coin})
	feed(t, c, "MESSAGE_REACTION_ADD", fluxer.MessageReactionAdd{ChannelID: "30", MessageID: "20", UserID: "1000", Emoji: coin})
	feed(t, c, "MESSAGE_REACTION_ADD_MANY", fluxer.MessageReactionAddMany{ChannelID: "30", MessageID: "20", Reactions: []fluxer.ReactionAddition{
		{UserID: "8", Emoji: fluxer.ReactionEmoji{Name: "\U0001F44D"}},
	}})
	m, _ := c.Message("30", "20")
	if m.Content != "hello!" || len(m.Reactions) != 2 || m.Reactions[0].Count != 2 || !m.Reactions[0].Me {
		t.Fatalf("got %+v", m)
	}
	feed(t, c, "MESSAGE_REACTION_REMOVE", fluxer.MessageReactionRemove{ChannelID: "30", MessageID: "20", UserID: "1000", Emoji: coin})
	feed(t, c, "MESSAGE_REACTION_REMOVE_EMOJI", fluxer.MessageReactionRemoveEmoji{ChannelID: "30", MessageID: "20", Emoji: fluxer.ReactionEmoji{Name: "\U0001F44D"}})
	m, _ = c.Message("30", "20")
	if len(m.Reactions) != 1 || m.Reactions[0].Count != 1 || m.Reactions[0].Me {
		t.Errorf("got %+v", m.Reactions)
	}
	feed(t, c, "MESSAGE_DELETE", fluxer.MessageDelete{ID: "20", ChannelID: "30"})
	if _, ok := c.Message("30", "20"); ok {
		t.Error("deleted message kept")
	}
	feed(t, c, "MESSAGE_CREATE", message("21", "30", "x"))
	feed(t, c, "CHANNEL_DELETE", fluxer.Channel{ID: "30", GuildID: "3"})
	if len(c.Messages("30", fluxer.MessageQuery{})) != 0 {
		t.Error("messages of a deleted channel kept")
	}
}

func TestMessageTTLAndPolicy(t *testing.T) {
	c := state.New(state.WithMessages(10, time.Hour), state.WithMessagePolicy(state.MessagesFromHumans()))
	ready(c)
	old := message(fluxer.SnowflakeAt(time.Now().Add(-2*time.Hour)), "30", "old")
	old.Timestamp = time.Now().Add(-2 * time.Hour)
	fresh := message(fluxer.SnowflakeAt(time.Now()), "30", "fresh")
	bot := message(fluxer.SnowflakeAt(time.Now())+"1", "30", "beep")
	bot.Author.Bot = true
	feed(t, c, "MESSAGE_CREATE", old)
	feed(t, c, "MESSAGE_CREATE", fresh)
	feed(t, c, "MESSAGE_CREATE", bot)
	if got := c.Messages("30", fluxer.MessageQuery{}); len(got) != 1 || got[0].Content != "fresh" {
		t.Errorf("got %v", ids(got))
	}
	c2 := state.New(state.WithMessages(0, 0))
	feed(t, c2, "MESSAGE_CREATE", fresh)
	if len(c2.Messages("30", fluxer.MessageQuery{})) != 0 {
		t.Error("disabled message cache stored a message")
	}
}

func ids(ms []fluxer.Message) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.ID
	}
	return out
}
