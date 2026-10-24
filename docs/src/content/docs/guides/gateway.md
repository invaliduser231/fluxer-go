---
title: Gateway
description: Session options, commands, compression and keeping a session across restarts.
---

```go
session := &fluxer.Session{
	Token:         token,
	Handler:       router.Handle,
	Compress:      true,
	IgnoredEvents: []string{"TYPING_START", "PRESENCE_UPDATE"},
	Presence:      &fluxer.StatusUpdate{Status: fluxer.StatusIdle},
	Logger:        slog.Default(),
}
err := session.Run(ctx)
```

`Run` connects, identifies and then keeps the connection alive until `ctx` ends. It reconnects with backoff between `MinBackoff` and `MaxBackoff` (one second and one minute by default) and resumes the session, so events sent while you were away arrive after the reconnect.

It returns early only when another attempt can't succeed:

| Error | Meaning |
| --- | --- |
| `fluxer.ErrInvalidToken` | Fluxer rejected the token. |
| `*fluxer.BanError` | The IP address is banned. `Until` says for how long. |
| `*fluxer.CloseError` with 4002, 4010, 4011 or 4012 | The payload, shard or API version is wrong. 4011 means the bot needs more shards. |

## Options

- `Compress` asks for `zstd-stream` compression. It saves a lot of bandwidth for bots in big guilds and costs a little CPU.
- `IgnoredEvents` tells Fluxer not to send those events at all. That is cheaper than dropping them yourself.
- `Presence` is the status the session starts with.
- `Shard` and `Pacer` are set by [`fluxer.Shards`](/fluxer-go/guides/sharding/); you rarely touch them yourself.
- `Flags` takes `fluxer.FlagDebounceMessageReactions` and `fluxer.FlagChannelThreads`.

## Commands

Commands need a ready connection and return `fluxer.ErrNotConnected` before that.

```go
session.UpdateStatus(ctx, fluxer.StatusUpdate{Status: fluxer.StatusDND})

session.RequestGuildMembers(ctx, fluxer.RequestMembers{GuildID: guildID, Query: "ann", Limit: 10})

session.RequestGuildCounts(ctx, []string{guildID}, "")
```

The answers come back as events (`GuildMembersChunk`, `GuildCountsUpdate`). If you want to wait for all chunks of a member request, use [`state.Chunker`](/fluxer-go/guides/cache/#chunking).

Fluxer allows five presence updates per 20 seconds on one connection and drops the rest without telling you.

## Surviving a restart

Fluxer keeps a disconnected session for 60 seconds. If your process restarts within that window, it can resume instead of starting over:

```go
cp := session.Checkpoint()
saveSomewhere(cp.SessionID, cp.Seq)

next := &fluxer.Session{Token: token, Handler: router.Handle}
next.Restore(fluxer.Checkpoint{SessionID: id, Seq: seq})
next.Run(ctx)
```

This works for rolling deploys: start the new process with the checkpoint of the old one, then stop the old one. [Resuming](/fluxer-go/background/resuming/) explains what Fluxer replays and when it refuses.

## Latency and the logged-in user

`session.Latency()` is the time between the last heartbeat and its acknowledgement. `session.User()` is the bot user from the last READY.
