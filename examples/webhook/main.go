package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

func main() {
	hook, err := fluxer.ParseWebhookURL(os.Getenv("FLUXER_WEBHOOK_URL"))
	if err != nil {
		log.Fatal("set FLUXER_WEBHOOK_URL to a webhook url")
	}
	client := fluxer.NewClient("")
	embed := fluxer.Embed{Title: "Deploy finished", Description: "Version 1.4.2 is live."}
	embed.SetColor(0x43b581).SetTimestamp(time.Now())

	msg, err := client.ExecuteWebhook(context.Background(), hook, "", fluxer.WebhookMessage{
		MessageSend: fluxer.MessageSend{Embeds: []fluxer.Embed{embed}},
		Username:    "CI",
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("sent message %s", msg.ID)
}
