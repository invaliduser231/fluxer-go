---
title: Your first bot
description: A small bot from the first line to a clean shutdown.
---

This bot replies to `!ping` with the time the reply took, and to `!server` with a few numbers about the guild.

```go
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	token := os.Getenv("FLUXER_TOKEN")
	client := fluxer.NewClient(token)
	cache := state.New()
	router := fluxer.NewRouter()

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
		client.EditMessage(ctx, reply.ChannelID, reply.ID, fluxer.MessageEdit{Content: &content})
	case "!server":
		g, ok := cache.Guild(m.GuildID)
		if !ok {
			return
		}
		embed := fluxer.Embed{Title: g.Name}
		embed.AddField("Members", fmt.Sprint(g.MemberCount), true).
			AddField("Channels", fmt.Sprint(len(g.Channels)), true)
		client.SendMessage(ctx, m.ChannelID, fluxer.MessageSend{Embeds: []fluxer.Embed{embed}})
	}
}
```

## The parts

**Client** sends REST requests. It waits when Fluxer says a rate limit is exhausted and retries requests that are safe to repeat. `Reply`, `SendMessage` and `EditMessage` are three of about 160 calls.

**Session** holds the gateway connection. `Run` blocks until the context ends. When the connection drops, it reconnects and resumes, so you get the events you missed. It only gives up when retrying can't help, for example with `fluxer.ErrInvalidToken`.

**Handler** receives every event as a name and raw JSON. Here it feeds the cache first and then the router, so a handler can already read the guild from the cache.

**Router** decodes events into types like `*fluxer.MessageCreate` and calls the functions registered with `fluxer.On`.

## Don't block the gateway

Handlers run on the goroutine that reads from the gateway. While a handler waits for an HTTP response, no other event is processed and heartbeats can be delayed. That's why the bot starts `handle` in its own goroutine.

## Next

- [Messages and files](/fluxer-go/guides/messages/) for more of the REST side
- [The cache](/fluxer-go/guides/cache/) if you want members, roles and permissions without extra requests
- [Testing](/fluxer-go/guides/testing/) to run this bot against a fake server
