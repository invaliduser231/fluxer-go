package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/state"
)

func main() {
	token := os.Getenv("FLUXER_TOKEN")
	if token == "" {
		log.Fatal("set FLUXER_TOKEN")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := fluxer.NewClient(token)
	cache := state.New()
	router := fluxer.NewRouter()
	router.OnError = func(event string, err error) { log.Printf("%s: %v", event, err) }

	fluxer.On(router, func(e *fluxer.Ready) {
		log.Printf("logged in as %s", e.User.Tag())
	})
	fluxer.On(router, func(m *fluxer.MessageCreate) {
		if m.Author.Bot {
			return
		}
		go handle(ctx, client, cache, &m.Message)
	})

	session := &fluxer.Session{
		Token: token,
		Handler: func(event string, data json.RawMessage) {
			cache.Handle(event, data)
			router.Handle(event, data)
		},
	}
	if err := session.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}

func handle(ctx context.Context, client *fluxer.Client, cache *state.Cache, m *fluxer.Message) {
	switch m.Content {
	case "!ping":
		start := time.Now()
		reply, err := client.Reply(ctx, m, fluxer.MessageSend{Content: "pong"})
		if err != nil {
			log.Print(err)
			return
		}
		content := fmt.Sprintf("pong, %d ms", time.Since(start).Milliseconds())
		if _, err := client.EditMessage(ctx, reply.ChannelID, reply.ID, fluxer.MessageEdit{Content: &content}); err != nil {
			log.Print(err)
		}
	case "!server":
		g, ok := cache.Guild(m.GuildID)
		if !ok {
			return
		}
		embed := fluxer.Embed{Title: g.Name}
		embed.AddField("Members", fmt.Sprint(g.MemberCount), true).
			AddField("Channels", fmt.Sprint(len(g.Channels)), true).
			AddField("Roles", fmt.Sprint(len(g.Roles)), true)
		if _, err := client.SendMessage(ctx, m.ChannelID, fluxer.MessageSend{Embeds: []fluxer.Embed{embed}}); err != nil {
			log.Print(err)
		}
	}
}
