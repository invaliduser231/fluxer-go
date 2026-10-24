---
title: Events
description: Typed gateway events and the router.
---

The gateway sends events as a name and a JSON payload. `Session.Handler` gets both. You can work with that directly, but usually you want types.

## The router

```go
router := fluxer.NewRouter()

fluxer.On(router, func(e *fluxer.GuildMemberAdd) {
	log.Printf("%s joined %s", e.User.Tag(), e.GuildID)
})
fluxer.On(router, func(e *fluxer.MessageReactionAdd) {
	if e.Emoji.Name == "\u2b50" {
		go addToStarboard(e.ChannelID, e.MessageID)
	}
})

session.Handler = router.Handle
```

The type of the parameter decides which event a function receives. Each event is decoded once, no matter how many handlers it has.

`fluxer.On` returns a function that removes the handler again, which is handy for waiting on a single event:

```go
done := make(chan *fluxer.MessageCreate, 1)
var remove func()
remove = fluxer.On(router, func(m *fluxer.MessageCreate) {
	if m.Author.ID == userID && m.ChannelID == channelID {
		remove()
		done <- m
	}
})
```

## Everything else

`OnAny` sees every event, including ones fluxer-go has no type for yet:

```go
router.OnAny(func(event string, data json.RawMessage) {
	metrics.Inc(event)
})
```

A handler that panics doesn't take the gateway down. The router recovers and reports the panic to `OnError`, together with payloads it couldn't decode:

```go
router.OnError = func(event string, err error) {
	log.Printf("%s: %v", event, err)
}
```

## Without the router

`fluxer.Decode` turns a raw event into its type, or returns `nil` for names it doesn't know:

```go
e, err := fluxer.Decode(event, data)
switch e := e.(type) {
case *fluxer.MessageCreate:
	...
case *fluxer.GuildDelete:
	...
}
```

## Things Fluxer does differently

- There are no intents. Every event a session may see is sent unless you list it in `Session.IgnoredEvents`.
- After READY a bot gets one `GuildCreate` per guild. The members in these carry only a user ID; the [cache](/fluxer-go/guides/cache/) fills them in where it can.
- `GuildDelete` with `Unavailable` set means a guild is briefly unreachable. Without it, the bot was removed.
- Gateway refusals that are not dispatches, like a full voice channel, arrive as `*fluxer.GatewayError` under the name `GATEWAY_ERROR`.
