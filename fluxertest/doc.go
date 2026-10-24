// Package fluxertest runs a fake Fluxer server for tests.
//
// The [Server] answers the common REST routes, keeps guilds, members, messages
// and webhooks in memory and speaks the gateway protocol, including resume,
// zstd compression and voice grants. Tests dispatch events to connected
// sessions and inspect the requests a bot sent:
//
//	srv := fluxertest.NewServer("1000.secret")
//	defer srv.Close()
//	session := &fluxer.Session{URL: srv.GatewayURL(), Token: srv.Token, Handler: router.Handle}
//	go session.Run(ctx)
//	srv.Dispatch("MESSAGE_CREATE", fluxer.Message{ID: "1", ChannelID: "2", Content: "!ping"})
//
// Helpers such as [Server.FailOnce], [Server.Drop] and
// [Server.InvalidateSessions] simulate failures.
package fluxertest
