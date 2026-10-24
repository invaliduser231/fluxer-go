---
title: Webhooks
description: Posting through webhooks without a bot, and forwarding payloads from other services.
---

A webhook needs no bot and no gateway, only its URL.

```go
hook, err := fluxer.ParseWebhookURL(os.Getenv("FLUXER_WEBHOOK_URL"))
if err != nil {
	log.Fatal(err)
}
client := fluxer.NewClient("")

msg, err := client.ExecuteWebhook(ctx, hook, "", fluxer.WebhookMessage{
	MessageSend: fluxer.MessageSend{Content: "Nightly build passed"},
	Username:    "CI",
	AvatarURL:   "https://example.com/ci.png",
}, nil)
```

`NewClient("")` sends no `Authorization` header, which is all a webhook needs. The second argument is a thread ID, for posting into a thread or a forum post. `ThreadName` and `AppliedTags` on `WebhookMessage` create a new forum post instead.

The returned message lets you change or remove it later:

```go
content := "Nightly build failed"
client.EditWebhookMessage(ctx, hook, "", msg.ID, fluxer.MessageEdit{Content: &content})
client.DeleteWebhookMessage(ctx, hook, "", msg.ID)
```

## Managing webhooks as a bot

```go
hook, err := client.CreateWebhook(ctx, channelID, "Releases", "")
hooks, err := client.ChannelWebhooks(ctx, channelID)
client.DeleteWebhook(ctx, hook.ID)
```

## GitHub, Slack and status pages

Fluxer renders payloads from a few services itself. If you relay them, pass them through unchanged:

```go
func githubHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	err := client.ExecuteGitHubWebhook(r.Context(), hook, "", r.Header.Get("X-GitHub-Event"), r.Header.Get("X-GitHub-Delivery"), body)
	...
}
```

`ExecuteSlackWebhook` takes Slack's message format, `ExecuteInstatusWebhook` and `ExecuteLibredeskWebhook` the payloads of those services. In most setups you'd rather point GitHub straight at the Fluxer URL with `/github` appended; the relay helps when the payload passes through your own server first.
