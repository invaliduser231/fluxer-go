---
title: Testing
description: Running your bot against a fake Fluxer server.
---

`fluxertest` starts a fake Fluxer in your test process. It speaks the REST routes and the gateway protocol, keeps guilds, members, messages and webhooks in memory, and lets you inspect what your code sent.

```go
func TestPing(t *testing.T) {
	srv := fluxertest.NewServer("1000.secret")
	defer srv.Close()

	client := srv.Client()
	router := fluxer.NewRouter()
	registerCommands(router, client)

	session := &fluxer.Session{URL: srv.GatewayURL(), Token: srv.Token, Handler: router.Handle}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go session.Run(ctx)

	srv.Dispatch("MESSAGE_CREATE", fluxer.Message{ID: "1", ChannelID: "2", Content: "!ping", Author: fluxer.User{ID: "7"}})

	deadline := time.Now().Add(time.Second)
	for len(srv.Messages("2")) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := srv.Messages("2"); len(got) != 2 || got[1].Content != "pong" {
		t.Fatalf("got %+v", got)
	}
}
```

`Dispatch` sends an event to every connected session. Messages your bot sends land in `srv.Messages(channelID)`, and `srv.Requests()` lists every request with method, path, body and audit log reason.

## Seeding data

```go
srv.AddGuild(fluxer.Guild{ID: "3", Name: "Test", OwnerID: "1",
	Channels: []fluxer.Channel{{ID: "30", Name: "general"}}})
srv.AddMember("3", fluxer.Member{User: &fluxer.User{ID: "7", Username: "anna"}})
srv.AddMessage(fluxer.Message{ID: "50", ChannelID: "30", Content: "hello"})
```

## Failures

The interesting tests are the ones where Fluxer misbehaves:

| Call | Effect |
| --- | --- |
| `FailOnce(pattern, status, body)` | The next request to that route returns the given error. |
| `RateLimitOnce(pattern, seconds)` | The next request gets a 429. |
| `ExhaustOnce(pattern, seconds)` | The next response says the bucket is empty. |
| `Drop()` | Cuts all gateway connections without a close frame. |
| `RequestReconnect()` | Sends a reconnect request, as during a Fluxer deploy. |
| `InvalidateSessions()` | Forgets all sessions, so the next resume fails. |
| `RefuseNextVoice(code)` | Refuses the next voice join with a gateway error. |
| `Ban(code, until)` | Answers every request as if the IP were banned. |

Patterns are the ones from `net/http`, like `"POST /v1/channels/{channel}/messages"`.

## Routes the fake doesn't know

`Handle` adds a route that checks the token like the others, `HandleRaw` one without:

```go
srv.Handle("GET /v1/guilds/{guild}/audit-logs", func(w http.ResponseWriter, r *http.Request) {
	fluxertest.WriteJSON(w, fluxer.AuditLog{})
})
```
