---
title: Resuming
description: What happens between a dropped connection and the next event.
---

A gateway connection ends sooner or later: a network hiccup, a Fluxer deploy, your own restart. Fluxer keeps the session alive for 60 seconds after the socket closes and buffers the events it would have sent. `Session.Run` uses that.

## The normal case

1. The connection drops.
2. `Run` waits a moment (a random part of `MinBackoff`, so a thousand bots don't reconnect in the same millisecond) and opens a new socket.
3. It sends a resume with the session ID and the last sequence number it processed.
4. Fluxer replays the missed events and sends `RESUMED`.

Your handler sees the replayed events like any others. Nothing else in your code changes.

## When resuming fails

Fluxer answers with an invalid session when the session has expired or the gap is larger than its buffer of 4,096 events. `Run` then identifies again, which means a fresh `READY` and, for bots, one `GuildCreate` per guild. The cache rebuilds itself from those. Events from the gap are gone; if they matter, read the history with `Client.Messages`.

The same happens when Fluxer doesn't answer a resume at all within `ReadyTimeout`.

## Deploys without a gap

A new process can take over the session of an old one:

```go
cp := oldSession.Checkpoint()
newSession.Restore(cp)
go newSession.Run(ctx)
stopOldProcess()
```

Fluxer moves the session to the new socket and tells the old one to reconnect. Close the old process right after, otherwise it would take the session back.

## Heartbeats

Fluxer closes a connection when no heartbeat arrived for 45 seconds. The session sends them on its own goroutine, but acknowledgements are read in the same loop that calls your handler. A handler that blocks for longer than about half a minute makes the session think the connection is dead, and it reconnects. Start a goroutine for slow work.
