// Package fluxer is a client for the Fluxer API at fluxer.app and on
// self-hosted instances.
//
// A [Client] sends REST requests and handles rate limits and retries. A
// [Session] keeps a gateway connection open, resumes it after drops and hands
// every event to its Handler. A [Router] decodes those events into types such
// as [MessageCreate] and calls the functions registered with [On].
//
//	client := fluxer.NewClient(os.Getenv("FLUXER_TOKEN"))
//	router := fluxer.NewRouter()
//	fluxer.On(router, func(m *fluxer.MessageCreate) {
//		if m.Content == "!ping" {
//			go client.Reply(ctx, &m.Message, fluxer.MessageSend{Content: "pong"})
//		}
//	})
//	session := &fluxer.Session{Token: client.Token, Handler: router.Handle}
//	log.Fatal(session.Run(ctx))
//
// The state package caches what the gateway reports, the voice module joins
// voice channels and fluxertest provides a fake server for tests. Guides are at
// https://invaliduser231.github.io/fluxer-go/.
package fluxer
