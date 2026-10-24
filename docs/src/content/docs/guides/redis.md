---
title: Redis
description: Sharing one cache between processes.
---

When a bot runs as several processes, for example one per shard group, each process has its own memory cache by default. With Redis they share one.

```sh
go get github.com/invaliduser231/fluxer-go/state/redis
```

```go
import (
	goredis "github.com/redis/go-redis/v9"

	"github.com/invaliduser231/fluxer-go/state"
	"github.com/invaliduser231/fluxer-go/state/redis"
)

client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
store := redis.New(client, "mybot")
store.OnError = func(err error) { log.Printf("cache: %v", err) }

cache := state.New(state.WithStore(store))
```

Everything else works as with the memory store. Policies and flags are applied by each process when it writes, so give all processes the same options.

## Keys

Each table is a set of hashes under the prefix, one hash per guild or channel: `{mybot:members}:g:<guild id>`, `{mybot:messages}:g:<channel id>` and so on. Values are JSON. Pick a prefix per bot, and per instance if the bot runs on several Fluxer instances.

The part in braces is a Redis Cluster hash tag. All keys of one table land in the same slot, so the multi-key transactions the store uses work on a cluster too. Different tables spread across slots.

`store.Flush(ctx)` deletes everything under the prefix. It uses `SCAN`, which on a Redis Cluster only reaches the node the client talks to.

## Errors and timeouts

The store interface has no error returns, so a failed Redis call is reported to `OnError` and the lookup behaves as a cache miss. Each call has a timeout of two seconds unless you set `store.Timeout`.

## Costs

Single lookups are one round trip. Listing a whole group, like `cache.Members(guildID)` or `SweepMembers`, loads the entire hash. For a guild with 100,000 members that is slow and memory hungry, so prefer single lookups or a member policy that keeps the set small.
