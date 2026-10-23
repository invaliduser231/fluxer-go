package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/state"
)

func main() {
	pairs := strings.Fields(os.Getenv("FLUXER_INSTANCES"))
	if len(pairs) == 0 {
		log.Fatal("set FLUXER_INSTANCES to origin=token pairs separated by spaces")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	instances := fluxer.NewInstances()
	caches := state.NewInstances()
	var wg sync.WaitGroup
	for _, pair := range pairs {
		origin, token, ok := strings.Cut(pair, "=")
		if !ok {
			log.Fatalf("%q is not origin=token", pair)
		}
		client, err := instances.Client(ctx, origin, token)
		if err != nil {
			log.Fatal(err)
		}
		session, err := instances.Session(ctx, origin, token)
		if err != nil {
			log.Fatal(err)
		}
		cache := caches.For(origin)
		router := fluxer.NewRouter()
		fluxer.On(router, func(m *fluxer.MessageCreate) {
			if m.Author.Bot || m.Content != "!where" {
				return
			}
			g, _ := cache.Guild(m.GuildID)
			go client.Reply(ctx, &m.Message, fluxer.MessageSend{Content: g.Name + " on " + origin})
		})
		session.Handler = func(event string, data json.RawMessage) {
			cache.Handle(event, data)
			router.Handle(event, data)
		}
		wg.Go(func() {
			if err := session.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("%s: %v", origin, err)
			}
		})
	}
	wg.Wait()
}
