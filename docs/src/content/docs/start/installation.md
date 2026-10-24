---
title: Installation
description: Add fluxer-go and the optional modules to your project.
---

You need Go 1.26 or newer.

```sh
go get github.com/invaliduser231/fluxer-go
```

That one module covers REST, the gateway, events, the cache and the test server. Two parts live in their own modules because they pull in larger dependencies:

| Module | Add it when you |
| --- | --- |
| `github.com/invaliduser231/fluxer-go/voice` | want to join voice channels. It brings the LiveKit SDK and Pion WebRTC. |
| `github.com/invaliduser231/fluxer-go/state/redis` | want several processes to share one cache. |

```sh
go get github.com/invaliduser231/fluxer-go/voice
go get github.com/invaliduser231/fluxer-go/state/redis
```

## A bot token

Create an application with a bot in Fluxer and copy the bot token. The token looks like `1427719902113.AbCd...`: the application ID, a dot and a secret.

Keep it out of your code and your repository. The examples on these pages read it from `FLUXER_TOKEN`:

```sh
export FLUXER_TOKEN=1427719902113.AbCd...
```

`fluxer.NewClient` also accepts the token with a `Bot ` prefix, in case you copied it from an HTTP header.

## Inviting the bot

```go
url := fluxer.BotInviteURL("", applicationID, fluxer.PermSendMessages|fluxer.PermViewChannel, "")
fmt.Println(url)
```

Open the link while logged in as someone who can manage the guild. The empty first argument means fluxer.app; pass your instance's API URL if you run your own.
