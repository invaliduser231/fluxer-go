---
title: Instances
description: Using a self-hosted Fluxer and running one bot on several instances.
---

Fluxer can be self-hosted. Every instance publishes its endpoints at `/.well-known/fluxer`, and `fluxer.Discover` reads them:

```go
inst, err := fluxer.Discover(ctx, "https://chat.example.org")
if err != nil {
	log.Fatal(err)
}
client := inst.NewClient(token)
session := inst.NewSession(token)
session.Handler = router.Handle
```

That is all it takes to run only on your own instance. `inst.Features` tells you what the instance supports, for example whether voice is enabled, and `inst.Media` is the base for avatar and icon URLs.

If you know the URLs already, skip discovery and set them yourself:

```go
client := fluxer.NewClient(token)
client.BaseURL = "https://api.chat.example.org/v1"
session := &fluxer.Session{URL: "wss://gateway.chat.example.org", Token: token}
```

Without either, everything goes to fluxer.app.

## One bot on several instances

A bot account exists per instance, so you have one token for each. `fluxer.Instances` keeps the discovery documents, refreshes them after an hour and keeps using the last good one when an instance is briefly down. It also paces identifies per instance.

IDs are only unique within an instance. Guild `1189375284394692608` on one instance has nothing to do with the same number elsewhere, which is why `state.Instances` keeps one cache per origin.

```go
instances := fluxer.NewInstances()
caches := state.NewInstances()

for origin, token := range tokens {
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
	registerCommands(router, client, cache)

	session.Handler = func(event string, data json.RawMessage) {
		cache.Handle(event, data)
		router.Handle(event, data)
	}
	go session.Run(ctx)
}
```

One router per instance keeps the client, the cache and the events of an instance together, so a handler can't accidentally answer on the wrong one.

`caches.Options` gives instances different cache settings, for example their own Redis prefix:

```go
caches.Options = func(origin string) []state.Option {
	return []state.Option{state.WithStore(redis.New(rdb, "mybot:"+origin))}
}
```

A complete program is in [examples/multi](https://github.com/invaliduser231/fluxer-go/tree/main/examples/multi).
