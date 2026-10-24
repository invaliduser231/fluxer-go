package fluxer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/invaliduser231/fluxer-go"
)

func ExampleOn() {
	client := fluxer.NewClient(os.Getenv("FLUXER_TOKEN"))
	router := fluxer.NewRouter()
	fluxer.On(router, func(m *fluxer.MessageCreate) {
		if m.Content == "!ping" {
			go client.Reply(context.Background(), &m.Message, fluxer.MessageSend{Content: "pong"})
		}
	})
	session := &fluxer.Session{Token: client.Token, Handler: router.Handle}
	log.Fatal(session.Run(context.Background()))
}

func ExampleClient_SendMessage() {
	client := fluxer.NewClient(os.Getenv("FLUXER_TOKEN"))
	embed := fluxer.Embed{Title: "Status"}
	embed.SetColor(0x43b581).AddField("API", "up", true).AddField("Gateway", "up", true)
	msg, err := client.SendMessage(context.Background(), "1189375284394692610", fluxer.MessageSend{
		Content: "Hourly check",
		Embeds:  []fluxer.Embed{embed},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(msg.ID)
}

func ExampleShards() {
	token := os.Getenv("FLUXER_TOKEN")
	router := fluxer.NewRouter()
	shards := &fluxer.Shards{
		Token:  token,
		Client: fluxer.NewClient(token),
		Handler: func(shard int, event string, data json.RawMessage) {
			router.Handle(event, data)
		},
	}
	log.Fatal(shards.Run(context.Background()))
}

func ExampleChannelPermissions() {
	g := fluxer.Guild{ID: "1", OwnerID: "9", Roles: []fluxer.Role{
		{ID: "1", Permissions: fluxer.PermViewChannel | fluxer.PermSendMessages},
	}}
	ch := fluxer.Channel{ID: "5", PermissionOverwrites: []fluxer.Overwrite{
		{ID: "1", Type: fluxer.OverwriteRole, Deny: fluxer.PermSendMessages},
	}}
	perms := fluxer.ChannelPermissions(g, "7", nil, ch)
	fmt.Println(perms.Has(fluxer.PermSendMessages), perms)
}

func ExampleParseWebhookURL() {
	hook, err := fluxer.ParseWebhookURL(os.Getenv("FLUXER_WEBHOOK_URL"))
	if err != nil {
		log.Fatal(err)
	}
	client := fluxer.NewClient("")
	_, err = client.ExecuteWebhook(context.Background(), hook, "", fluxer.WebhookMessage{
		MessageSend: fluxer.MessageSend{Content: "Build passed"},
		Username:    "CI",
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
}
