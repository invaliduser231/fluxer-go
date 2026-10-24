<img src="docs/src/assets/logo.svg" alt="" width="72" height="72">

# fluxer-go

Go library for the [Fluxer](https://fluxer.app) API: REST client, gateway, typed events, a state cache and voice over LiveKit.

```
go get github.com/invaliduser231/fluxer-go
```

The voice support lives in its own module so the WebRTC dependencies stay out of bots that do not need them:

```
go get github.com/invaliduser231/fluxer-go/voice
```

Requires Go 1.26 or newer. The API is not stable before 1.0.

Guides are at [invaliduser231.github.io/fluxer-go](https://invaliduser231.github.io/fluxer-go/), the API reference is on [pkg.go.dev](https://pkg.go.dev/github.com/invaliduser231/fluxer-go).

## Packages

| Package | Contents |
| --- | --- |
| `fluxer` | REST client, gateway session, sharding, events, router, permissions, OAuth2 |
| `fluxer/state` | Cache fed by gateway events, with policies, old values, a message cache, member chunking and REST fallback |
| `fluxer/state/redis` | Redis storage for the cache |
| `fluxer/fluxertest` | Fake Fluxer server with REST routes and a gateway for tests |
| `fluxer/voice` | Joining voice channels, playing Ogg Opus, publishing and receiving tracks |

## A small bot

```go
client := fluxer.NewClient(os.Getenv("FLUXER_TOKEN"))
router := fluxer.NewRouter()

fluxer.On(router, func(m *fluxer.MessageCreate) {
	if m.Content == "!ping" {
		go client.Reply(ctx, &m.Message, fluxer.MessageSend{Content: "pong"})
	}
})

session := &fluxer.Session{Token: client.Token, Handler: router.Handle}
log.Fatal(session.Run(ctx))
```

`Session.Run` keeps the connection alive until the context ends. It resumes after drops, identifies again when a session cannot be resumed and returns only for errors that a reconnect cannot fix, such as `fluxer.ErrInvalidToken`, a `*fluxer.BanError` or the close codes 4002 and 4010 to 4012.

Handlers run on the gateway goroutine. Start a goroutine for anything that waits on the network.

A complete example is in [examples/ping](examples/ping).

## REST

Every call takes a context and returns typed results.

```go
msg, err := client.SendMessage(ctx, channelID, fluxer.MessageSend{
	Content: "Release notes",
	Embeds:  []fluxer.Embed{{Title: "v1.4.2", Description: "Fixes the login loop."}},
})

files := []fluxer.File{{Name: "report.csv", Data: data}}
_, err = client.SendMessageFiles(ctx, channelID, fluxer.MessageSend{Content: "Weekly report"}, files)

for m, err := range client.MessagesBefore(ctx, channelID, "") {
	if err != nil {
		return err
	}
	fmt.Println(m.Author.Tag(), m.Content)
}
```

Files above a few megabytes go through presigned uploads, which send the bytes straight to storage and split large files into parts:

```go
_, err = client.SendMessageUploads(ctx, channelID, fluxer.MessageSend{Content: "Recording"}, []fluxer.File{{Name: "call.mp4", Data: video}})
```

Edits use pointer fields, so only the fields you set are sent:

```go
topic := "Read the rules first"
_, err := client.ModifyChannel(ctx, channelID, fluxer.ChannelEdit{Topic: &topic})
```

Moderation calls accept an audit log reason through the context:

```go
ctx = fluxer.WithReason(ctx, "spam")
err := client.Ban(ctx, guildID, userID, fluxer.BanOptions{DeleteMessageSeconds: 3600})
```

The client follows the rate limit headers per bucket, waits on 429 responses and retries `GET`, `PUT`, `PATCH` and `DELETE` requests as well as message sends that carry a nonce. `SendMessage` sets a nonce when you do not. Errors from the API are `*fluxer.APIError` values:

```go
if fluxer.ErrorCode(err) == "MISSING_PERMISSIONS" {
	...
}
```

A 401 marks the token as rejected and later calls fail without a request. `errors.Is(err, fluxer.ErrTokenRejected)` reports it.

## Events

`fluxer.On` registers a handler for one event type and returns a function that removes it. The router decodes each event once, no matter how many handlers it has.

```go
fluxer.On(router, func(e *fluxer.GuildMemberAdd) {
	log.Printf("%s joined %s", e.User.Tag(), e.GuildID)
})
router.OnAny(func(event string, data json.RawMessage) {
	log.Println(event)
})
router.OnError = func(event string, err error) {
	log.Printf("%s: %v", event, err)
}
```

`fluxer.Decode(event, data)` turns a raw dispatch into its typed event if you prefer your own dispatching.

## State

```go
cache := state.New()
session.Handler = func(event string, data json.RawMessage) {
	cache.Handle(event, data)
	router.Handle(event, data)
}

g, ok := cache.Guild(guildID)
perms, ok := cache.Permissions(guildID, channelID, userID)
if perms.Has(fluxer.PermSendMessages | fluxer.PermEmbedLinks) {
	...
}
```

The cache keeps guilds, channels, threads, roles, members, users, presences, voice states, emojis, stickers and the last messages of each channel. Every entity can be switched off, and members, channels and messages can be filtered with policies:

```go
cache := state.New(
	state.WithoutFlags(state.FlagPresences),
	state.WithMemberPolicy(state.MembersInVoice().Or(state.MemberOwner())),
	state.WithChannelPolicy(state.ChannelTypes(fluxer.ChannelText, fluxer.ChannelVoice)),
	state.WithMessages(200, time.Hour),
	state.WithMessagePolicy(state.MessagesFromHumans()),
)
```

The bot's own member is always kept. A member that stops matching its policy, for example by leaving voice, is dropped. `SweepMembers` and `SweepMessages` apply the policies and the message TTL to everything already stored.

Members from the guild burst after READY carry only a user ID. The cache fills them from the users it has seen, and `MemberPartial` reports the ones it could not complete.

Handlers that need the state before an update register with `state.OnChange`:

```go
state.OnChange(cache, func(old *fluxer.Message, e *fluxer.MessageUpdate) {
	if old != nil && old.Content != e.Content {
		log.Printf("edited: %q -> %q", old.Content, e.Content)
	}
})
state.OnChange(cache, func(old *fluxer.Member, e *fluxer.GuildMemberRemove) {
	if old != nil {
		log.Printf("%s left", old.DisplayName())
	}
})
```

`Chunker` requests members and waits until every chunk has arrived. With a filter it loads the members of each guild as it becomes available. `WaitGuilds` blocks until all guilds from READY have arrived.

```go
chunker := state.NewChunker(session)
chunker.Filter = state.ChunkGuilds(mainGuildID)
session.Handler = func(event string, data json.RawMessage) {
	cache.Handle(event, data)
	chunker.Handle(event, data)
	router.Handle(event, data)
}
members, err := chunker.Request(ctx, fluxer.RequestMembers{GuildID: guildID, Query: "ann", Limit: 10})
```

`Resolver` reads from the cache and falls back to REST. Parallel lookups of the same entity share one request, and the result is written back.

```go
resolver := state.NewResolver(cache, client)
member, err := resolver.Member(ctx, guildID, userID)
perms, err := resolver.Permissions(ctx, guildID, channelID, userID)
```

There are also helpers for the role hierarchy (`HighestRole`, `CanManageMember`, `CanManageRole`), name lookups (`FindMember`, `RoleByName`, `ChannelByName`, `EmojiByName`), voice (`VoiceMembers`, `VoiceChannel`), message history (`Messages` with before and after), iterators (`EachGuild`, `EachMember`, `EachMessage`) and `Stats`.

`fluxer.ChannelPermissions` computes permissions from a guild, roles and a channel without the cache.

### Redis

The storage behind the cache is an interface. `state/redis` keeps it in Redis, so several processes or shards can share one cache:

```
go get github.com/invaliduser231/fluxer-go/state/redis
```

```go
client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
store := redis.New(client, "mybot")
store.OnError = func(err error) { log.Print(err) }
cache := state.New(state.WithStore(store))
```

## Gateway options

```go
session := &fluxer.Session{
	Token:         token,
	Handler:       router.Handle,
	Compress:      true,
	IgnoredEvents: []string{"TYPING_START", "PRESENCE_UPDATE"},
	Presence:      &fluxer.StatusUpdate{Status: fluxer.StatusIdle},
	Logger:        slog.Default(),
}
```

`Compress` negotiates `zstd-stream` compression. `IgnoredEvents` asks Fluxer not to send those dispatches at all. Commands such as `UpdateStatus`, `RequestGuildMembers`, `RequestGuildCounts` and `UpdateVoiceState` are methods on the session.

`Session.User` returns the bot user from the last READY. `Checkpoint` and `Restore` carry a session ID and sequence across a process restart, so a new process can resume within the 60 seconds Fluxer keeps a session.

## Sharding

A bot in more than 2,500 guilds needs several shards. `fluxer.Shards` runs one session per shard and paces the identifies.

```go
shards := &fluxer.Shards{
	Token:  token,
	Client: client,
	Handler: func(shard int, event string, data json.RawMessage) {
		router.Handle(event, data)
	},
}
log.Fatal(shards.Run(ctx))
```

Without `Count` the recommendation from `GET /gateway/bot` is used. `shards.ForGuild(guildID)` returns the session that owns a guild.

## Voice

Fluxer runs voice through LiveKit. The voice package sends the voice state update, waits for the grant and connects to the LiveKit room.

```go
manager := voice.NewManager(session, router)
conn, err := manager.Join(ctx, guildID, channelID, voice.Options{SelfDeaf: true})
if err != nil {
	return err
}
defer conn.Leave(ctx)

err = conn.PlayFile(ctx, "song.ogg")
```

`Play` takes any Ogg Opus stream. Other formats can be converted on the fly:

```go
cmd := exec.CommandContext(ctx, "ffmpeg", "-i", input, "-c:a", "libopus", "-b:a", "96k", "-f", "ogg", "-")
out, _ := cmd.StdoutPipe()
if err := cmd.Start(); err != nil {
	return err
}
err = conn.Play(ctx, out)
```

`conn.OpusTrack` publishes a track that takes Opus frames directly, `voice.Options.OnTrack` receives the tracks of other participants and `voice.Options.OnDisconnect` reports a dropped media connection. `Move`, `SetMute` and `SetDeaf` change the connection without leaving. A refused join returns a `*fluxer.GatewayError`, for example with the code `fluxer.VoiceChannelFull`.

See [voice/examples/play](voice/examples/play).

## Webhooks

```go
hook, err := fluxer.ParseWebhookURL(webhookURL)
client := fluxer.NewClient("")
_, err = client.ExecuteWebhook(ctx, hook, "", fluxer.WebhookMessage{
	MessageSend: fluxer.MessageSend{Content: "Build passed"},
	Username:    "CI",
}, nil)
```

## OAuth2

```go
o := &fluxer.OAuth{
	ClientID:     clientID,
	ClientSecret: secret,
	RedirectURI:  "https://example.com/callback",
	Scopes:       []string{fluxer.ScopeIdentify, fluxer.ScopeGuilds},
}
verifier := fluxer.NewVerifier()
http.Redirect(w, r, o.AuthorizeURL(state, verifier), http.StatusFound)

token, err := o.Exchange(ctx, r.URL.Query().Get("code"), verifier)
user, err := o.User(ctx, token.AccessToken)
```

`fluxer.BotInviteURL` builds the link that adds a bot to a guild with a set of permissions.

GitHub, Slack, Instatus and Libredesk payloads can be forwarded to a webhook with `ExecuteGitHubWebhook`, `ExecuteSlackWebhook`, `ExecuteInstatusWebhook` and `ExecuteLibredeskWebhook`.

## Instances

Fluxer can be self-hosted. `fluxer.Discover` reads the discovery document of an instance and returns its endpoints.

```go
inst, err := fluxer.Discover(ctx, "https://chat.example.org")
client := inst.NewClient(token)
session := inst.NewSession(token)
```

A bot that runs on several instances at once keeps a `fluxer.Instances` registry. It caches the discovery documents, keeps serving the last good document when an instance is briefly unreachable and paces identifies per instance. IDs are only unique within one instance, so `state.Instances` keeps one cache per origin. Its `Options` function gives each instance its own settings, for example a Redis prefix.

```go
instances := fluxer.NewInstances()
caches := state.NewInstances()

for origin, token := range tokens {
	client, err := instances.Client(ctx, origin, token)
	session, err := instances.Session(ctx, origin, token)
	cache := caches.For(origin)
	session.Handler = func(event string, data json.RawMessage) {
		cache.Handle(event, data)
		handle(origin, client, event, data)
	}
	go session.Run(ctx)
}
```

See [examples/multi](examples/multi).

## Testing

`fluxertest` starts a fake server with REST routes, a gateway that supports resume, compression and voice grants, and helpers to inspect what your code sent.

```go
srv := fluxertest.NewServer("1000.secret")
defer srv.Close()
srv.AddGuild(fluxer.Guild{ID: "3", Name: "Test"})

client := srv.Client()
session := &fluxer.Session{URL: srv.GatewayURL(), Token: srv.Token, Handler: router.Handle}

srv.Dispatch("MESSAGE_CREATE", fluxer.Message{ID: "1", ChannelID: "2", Content: "!ping"})
reqs := srv.Requests()
```

`FailOnce`, `RateLimitOnce`, `Drop`, `RequestReconnect` and `InvalidateSessions` simulate failures. `Handle` adds routes the fake does not know.

## License

MIT
