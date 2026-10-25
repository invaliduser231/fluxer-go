# Changelog

## 0.2.0

- Forwarding with `Client.Forward`, media-only forwards and `Message.MessageSnapshots`; messages also carry `Users` and call info
- `GUILD_MEMBER_LIST_UPDATE`, `THREAD_MEMBER_LIST_UPDATE` and `GUILD_HEALTH_UPDATE` events, thread options for `Subscribe`
- Attachments report expiry, content hash, placeholder and waveform; files can be spoilers and carry a title, duration and waveform
- Message edits can keep, change and add attachments, `EditMessageUploads`
- Channels carry forum defaults, voice connection limits, P2P, content warnings and thread counters, and forums can be set up fully on create and edit
- Channels can be placed after a sibling in `ModifyChannelPositions`
- Thread search sorts, matches all tags and returns first messages; thread members show their mute state
- The cache follows thread member updates and group DM recipients
- Member search filters by join date, account age, join source and invite; members can edit their guild profile
- Guilds report threads, visits, content warnings, the history cutoff and image sizes; guild edits cover features, MFA, NSFW and content warnings
- Message search takes every filter and exclusion of the API and returns threads
- `ExecuteWebhookNoWait`, `PurgeEmoji`, `PurgeSticker`
- Embeds carry oEmbed HTML, nested embeds and media details; follower webhooks name their source; the application includes its bot user
- Voice states report E2EE and P2P, reaction removals carry the member
- Constants for guild levels, operations and features, public user flags, reply mention preferences, attachment flags, forum settings, archive durations and `FlagGuildVisits`

## 0.1.1

- Close codes 4001, 4003 and 4005 stop `Session.Run` with a `*CloseError` instead of reconnecting forever
- No second identify while Fluxer holds the first; `ReadyTimeout` defaults to 65 seconds
- Webhook avatars in audit logs are read from `avatar_hash`
- Channel pin updates take the guild lock in the cache and no longer race with channel updates
- Concurrent `Instances.Get` calls for one origin share a single discovery
- Discovery of an https instance rejects http API endpoints, ws gateways and redirects to http

## 0.1.0

- REST client for the bot routes of the API: messages, reactions, pins, channels, group DMs, threads, forums, guilds, roles, members, bans, emojis, stickers, invites, webhooks, audit logs, search, discovery, profiles and OAuth2
- Presigned uploads for large attachments, including multipart uploads
- Webhook relays for GitHub, Slack, Instatus and Libredesk payloads
- Rate limit buckets, retries for idempotent requests and nonce-guarded sends, typed API errors
- Gateway session with resume, reconnect, invalid session handling and zstd-stream compression
- Gateway commands for presence, member requests, counts, subscriptions and voice states
- Typed events for the dispatches a bot receives and a router with typed handlers
- Shard manager with paced identifies
- Permission bitset and permission computation
- State cache for guilds, channels, threads, roles, members, users, presences, voice states, emojis, stickers and messages
- Cache flags and combinable policies, old values for updates and deletes, partial member tracking, message limits and TTL
- Member chunker, waiting for guilds after READY, read-through resolver with request collapsing, hierarchy and lookup helpers
- Pluggable cache storage with an in-memory and a Redis store
- Voice module on LiveKit: join, move, mute, leave, Ogg Opus playback, raw Opus tracks, receiving tracks
- Instance discovery for self-hosted Fluxer and a registry for bots on several instances, with a state cache per instance
- Fake server for tests with REST routes and a gateway
