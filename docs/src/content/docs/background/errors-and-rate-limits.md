---
title: Errors and rate limits
description: What the client retries, what it gives up on, and how to read API errors.
---

## API errors

Every error from the API is a `*fluxer.APIError` with the HTTP status, Fluxer's error code and message, and the request ID Fluxer logged it under:

```go
_, err := client.SendMessage(ctx, channelID, msg)
var apiErr *fluxer.APIError
if errors.As(err, &apiErr) {
	log.Printf("%d %s (%s)", apiErr.Status, apiErr.Code, apiErr.RequestID)
}
```

`fluxer.ErrorCode(err)` is the shortcut when you only care about the code:

```go
switch fluxer.ErrorCode(err) {
case "MISSING_PERMISSIONS":
	// the bot lacks a permission or its role is too low
case "UNKNOWN_MESSAGE":
	// already deleted
}
```

Validation failures carry the offending fields in `apiErr.Errors`, each with a path like `embeds.0.title`.

## Rate limits

Fluxer counts requests per route and per bot, and in a global bucket of 50 requests per second. The client reads the `X-RateLimit-*` headers of every response. When a bucket is empty, the next request to that route waits until there is room, before it is sent. When Fluxer answers 429 anyway, the client waits for `retry_after` and tries again, up to `MaxRetries` times (three by default). After that the 429 comes back as an `*APIError` with `RetryAfter` set.

Waiting respects the context. A request with a deadline fails with `context.DeadlineExceeded` instead of sleeping past it.

## Retries

Besides 429, the client retries 423, 502, 503 and 504, but only for requests that are safe to repeat: `GET`, `PUT`, `PATCH`, `DELETE`, and `POST` requests with a nonce. Message sends get a nonce automatically, so a send that timed out on Fluxer's side is not posted twice. Other `POST` requests, like creating a channel, fail right away, because repeating them could create two.

## A rejected token

After a 401 the client marks its token as rejected and fails every later call without sending it. `errors.Is(err, fluxer.ErrTokenRejected)` reports that case. Create a new client once you have a new token.

## Slowmode

A send into a channel with slowmode can fail with `SLOWMODE_RATE_LIMITED`. That is not a bucket limit and is not retried. Bots with `PermBypassSlowmode` are exempt.
