---
title: Sharding
description: Splitting a large bot across several gateway sessions.
---

One gateway session can hold up to 2,500 guilds. Above that Fluxer closes the connection with 4011 and the bot needs shards: several sessions, each responsible for part of the guilds.

```go
shards := &fluxer.Shards{
	Token:  token,
	Count:  4,
	Client: client,
	Handler: func(shard int, event string, data json.RawMessage) {
		cache.Handle(event, data)
		router.Handle(event, data)
	},
}
log.Fatal(shards.Run(ctx))
```

`Run` starts one session per shard and stops all of them when one fails for good. Without `Count` it asks `GET /gateway/bot`, which currently always recommends one shard, so set the count yourself once you grow.

A guild belongs to shard `(guild_id >> 22) % count`. `fluxer.ShardFor` computes that, and `shards.ForGuild(guildID)` gives you the session to send commands for a guild:

```go
shards.ForGuild(guildID).RequestGuildMembers(ctx, fluxer.RequestMembers{GuildID: guildID})
```

Identifies are paced at one per second by default. Pass your own `Pacer` if you run shards in several processes and want them to share one budget, for example with `Pacer.Reserve` backed by Redis.

Account events that belong to no guild only arrive on shard 0.

`Configure` runs for every session before it starts, which is where per-session settings like `Compress` or `IgnoredEvents` go.
