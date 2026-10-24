---
title: The cache
description: Keeping guilds, members, messages and voice states in memory, with policies, old values, chunking and REST fallback.
---

The `state` package builds a picture of everything your bot can see from the gateway events: guilds, channels, threads, roles, members, users, presences, voice states, emojis, stickers and the last messages of each channel.

```go
cache := state.New()
session.Handler = func(event string, data json.RawMessage) {
	cache.Handle(event, data)
	router.Handle(event, data)
}
```

Feed the cache before the router. Then handlers already see the state after the event.

```go
g, ok := cache.Guild(guildID)
m, ok := cache.Member(guildID, userID)
ch, ok := cache.Channel(channelID)
perms, ok := cache.Permissions(guildID, channelID, userID)
```

`Permissions` applies roles, channel overwrites and the owner rule, and uses the parent channel for threads.

## Choosing what to keep

Every kind of entity can be switched off:

```go
cache := state.New(state.WithoutFlags(state.FlagPresences | state.FlagStickers))
```

Members, channels and messages can also be filtered. A policy is a function that decides per entity, and policies combine with `And`, `Or` and `Not`:

```go
cache := state.New(
	state.WithMemberPolicy(state.MembersInVoice().Or(state.MemberOwner())),
	state.WithChannelPolicy(state.ChannelTypes(fluxer.ChannelText, fluxer.ChannelVoice)),
)
```

Built in are `MembersInVoice`, `MembersOnline`, `MemberOwner`, `MembersWithRole`, `MembersInGuilds`, `MembersTimedOut`, `ChannelTypes`, `MessagesInGuilds` and `MessagesFromHumans`. Your own policy is a plain function:

```go
regulars := func(c *state.Cache, m fluxer.Member) bool {
	return time.Since(m.JoinedAt) > 30*24*time.Hour
}
cache := state.New(state.WithMemberPolicy(regulars))
```

Two rules apply on top. The bot's own member is always kept, because permission checks need it. And a member is dropped again as soon as it stops matching, for example when it leaves voice under `MembersInVoice`. `SweepMembers` re-checks everything that is already stored.

## Messages

The last 100 messages per channel are kept by default.

```go
cache := state.New(
	state.WithMessages(500, 6*time.Hour),
	state.WithMessagePolicy(state.MessagesFromHumans()),
	state.WithMessageEvict(func(m fluxer.Message) { archive(m) }),
)
```

Edits replace the stored message, reactions update its counts, and deletes remove it. `Messages` reads the history back:

```go
recent := cache.Messages(channelID, fluxer.MessageQuery{Limit: 20})
older := cache.Messages(channelID, fluxer.MessageQuery{Before: recent[len(recent)-1].ID, Limit: 20})
```

`WithMessages(0, 0)` turns the message cache off. `SweepMessages` drops what is older than the TTL.

## Old values

Many bots care about what changed: the old nick, the content before an edit, the message that was deleted. `state.OnChange` passes the cached value from before the event:

```go
state.OnChange(cache, func(old *fluxer.Message, e *fluxer.MessageDelete) {
	if old != nil {
		modlog.Printf("deleted in <#%s>: %s", e.ChannelID, old.Content)
	}
})
state.OnChange(cache, func(old *fluxer.Member, e *fluxer.GuildMemberUpdate) {
	if old != nil && old.DisplayName() != e.DisplayName() {
		modlog.Printf("%s is now %s", old.DisplayName(), e.DisplayName())
	}
})
```

`old` is `nil` when the cache didn't have the entity, for example a message older than the message limit. Bulk deletes pass `[]fluxer.Message`, emoji updates `[]fluxer.Emoji`.

## Members that only have an ID

Right after READY, Fluxer sends bots one `GuildCreate` per guild. The members in it are reduced to the user ID. The cache fills in the user from anything else it has seen, a message, a presence or a member update. `MemberPartial` tells you which members are still incomplete.

## Chunking

To have all members of a guild, request them. `Chunker` sends the request and waits until every chunk has arrived:

```go
chunker := state.NewChunker(session)
session.Handler = func(event string, data json.RawMessage) {
	cache.Handle(event, data)
	chunker.Handle(event, data)
	router.Handle(event, data)
}

members, err := chunker.RequestAll(ctx, guildID)
```

With a filter it loads guilds by itself as they become available:

```go
chunker.Filter = state.ChunkGuilds(mainGuildID)
```

Fluxer lets a bot request the full member list of a guild once every 30 seconds. A request inside that window returns a `*state.RateLimitedError` with the remaining time. Requests by name or by user ID are not limited this way:

```go
found, err := chunker.Request(ctx, fluxer.RequestMembers{GuildID: guildID, Query: "ann", Limit: 10})
```

`cache.WaitGuilds(ctx)` blocks until every guild from READY has arrived, which is the moment to start work that needs all guilds.

## Falling back to REST

`Resolver` checks the cache first and asks the API when something is missing. The answer goes into the cache, and when ten goroutines ask for the same member at once, only one request goes out.

```go
resolver := state.NewResolver(cache, client)
m, err := resolver.Member(ctx, guildID, userID)
perms, err := resolver.Permissions(ctx, guildID, channelID, userID)
```

`Resolver.Permissions` fetches the guild, roles, member and channel if needed, which makes it safe to call from a command handler without knowing what is cached.

## Helpers

| | |
| --- | --- |
| `HighestRole`, `CanManageMember`, `CanManageRole` | Role hierarchy as Fluxer applies it: higher position wins, on equal positions the older role. |
| `FindMember`, `RoleByName`, `ChannelByName`, `EmojiByName` | Lookups for commands that take names. `FindMember` matches username, `name#0042`, nick and display name, and falls back to a prefix. |
| `VoiceMembers`, `VoiceChannel`, `VoiceStates` | Who is in which voice channel. |
| `SelfMember`, `Self` | The bot itself. |
| `EachGuild`, `EachMember`, `EachMessage` | Iterators. |
| `Stats` | Counts per kind and resolver hits and misses. |

## Storage

The cache keeps its data in a `state.Store`. The default is in memory and sharded, so many guilds don't fight over one lock. [Redis](/fluxer-go/guides/redis/) is the other built-in option.
