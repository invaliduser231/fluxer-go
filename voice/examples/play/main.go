package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/voice"
)

func main() {
	guild := flag.String("guild", "", "guild id")
	channel := flag.String("channel", "", "voice channel id")
	file := flag.String("file", "", "ogg opus file to play")
	flag.Parse()
	token := os.Getenv("FLUXER_TOKEN")
	if token == "" || *guild == "" || *channel == "" || *file == "" {
		log.Fatal("usage: FLUXER_TOKEN=... play -guild ID -channel ID -file song.ogg")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	router := fluxer.NewRouter()
	ready := make(chan struct{}, 1)
	remove := fluxer.On(router, func(*fluxer.Ready) {
		select {
		case ready <- struct{}{}:
		default:
		}
	})
	session := &fluxer.Session{Token: token, Handler: router.Handle}
	go func() {
		if err := session.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Print(err)
			stop()
		}
	}()
	select {
	case <-ready:
		remove()
	case <-ctx.Done():
		return
	}

	manager := voice.NewManager(session, router)
	defer manager.Close()
	conn, err := manager.Join(ctx, *guild, *channel, voice.Options{SelfDeaf: true})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Leave(context.WithoutCancel(ctx))
	log.Printf("playing %s", *file)
	if err := conn.PlayFile(ctx, *file); err != nil && !errors.Is(err, context.Canceled) {
		log.Print(err)
	}
}
