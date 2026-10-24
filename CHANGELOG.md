# Changelog

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
