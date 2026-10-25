---
title: Messages and files
description: Sending, editing and reading messages, embeds, reactions, attachments and threads.
---

## Sending

```go
msg, err := client.SendText(ctx, channelID, "Deploy finished")
```

`SendText` is a shortcut for `SendMessage` with only content. Everything else goes through `fluxer.MessageSend`:

```go
embed := fluxer.Embed{Title: "v1.4.2", Description: "Fixes the login loop on Safari."}
embed.SetColor(0x1f8a80).SetTimestamp(time.Now()).AddField("Downtime", "none", true)

msg, err := client.SendMessage(ctx, channelID, fluxer.MessageSend{
	Content:         "Release notes",
	Embeds:          []fluxer.Embed{embed},
	AllowedMentions: &fluxer.AllowedMentions{Parse: []string{}},
})
```

An empty `Parse` list keeps the message from pinging anyone, which is a good default for content you didn't write yourself.

To answer a message, use `Reply`. It fills in the message reference:

```go
client.Reply(ctx, m, fluxer.MessageSend{Content: "Done."})
```

`Forward` sends a message on to another channel:

```go
client.Forward(ctx, m, logChannelID)
```

A forward has no content of its own. Fluxer copies the original into `MessageSnapshots`, so a bot that reads forwards finds the text, embeds and attachments there, and `MessageReference` says where the original came from. To forward only some of the media, set `AttachmentIDs` or `EmbedIndices` on a `MessageReference` with `Type: fluxer.ReferenceForward` and send it with `SendMessage`.

`SendMessage` adds a nonce when you don't set one. With a nonce Fluxer recognizes a repeated send, so the client can retry after a timeout or a 503 without posting the message twice.

## Editing and deleting

Edit structs use pointers. Fields you leave `nil` are not sent and stay as they are:

```go
content := "Release notes (updated)"
client.EditMessage(ctx, channelID, msgID, fluxer.MessageEdit{Content: &content})
```

Attachments work differently: once you set `Attachments` or `Uploads`, the message keeps exactly the attachments you list, so name every one that should stay. New files have to go through the upload URLs, which `EditMessageUploads` does for you:

```go
keep := []fluxer.AttachmentEdit{{ID: m.Attachments[0].ID}}
client.EditMessageUploads(ctx, channelID, msgID, fluxer.MessageEdit{Attachments: &keep},
	[]fluxer.File{{Name: "chart-v2.png", Data: png}})
```

```go
client.DeleteMessage(ctx, channelID, msgID)
client.BulkDelete(ctx, channelID, []string{"1189375284394692701", "1189375284394692702"})
```

Moderation calls accept an audit log reason through the context. It shows up in the guild's audit log:

```go
client.DeleteMessage(fluxer.WithReason(ctx, "spam"), channelID, msgID)
```

## Reading history

`Messages` returns one page. `MessagesBefore` walks backwards through the whole channel and fetches pages of 100 as you go:

```go
for m, err := range client.MessagesBefore(ctx, channelID, "") {
	if err != nil {
		return err
	}
	if m.Timestamp.Before(cutoff) {
		break
	}
	fmt.Println(m.Author.Tag(), m.Content)
}
```

Breaking out of the loop stops further requests.

## Reactions

Unicode emoji are passed as they are. Custom emoji use `name:id`:

```go
client.AddReaction(ctx, channelID, msgID, "\U0001F44D")
client.AddReaction(ctx, channelID, msgID, "coin:1189375284394692800")
```

`Emoji.APIName()` and `ReactionEmoji.APIName()` give you that form from an emoji you already have.

## Files

Small files go straight into the message as multipart parts:

```go
files := []fluxer.File{{Name: "report.csv", Data: csv, Description: "Weekly numbers"}}
client.SendMessageFiles(ctx, channelID, fluxer.MessageSend{Content: "Weekly report"}, files)
```

For anything larger use `SendMessageUploads`. It asks Fluxer for upload URLs, sends the bytes there directly, splits files above 10 MiB into parts and only then posts the message:

```go
client.SendMessageUploads(ctx, channelID, fluxer.MessageSend{Content: "Call recording"},
	[]fluxer.File{{Name: "call.mp4", Data: video}})
```

`Flags: fluxer.AttachmentSpoiler` hides a file behind a spoiler, and `Title` gives it a caption. For voice messages set `Duration` and `Waveform` as well.

At most ten files fit into one message. Some self-hosted instances turn presigned uploads off; `inst.Features.PresignedAttachmentUploads` tells you.

Attachment URLs expire, and `Attachment.ExpiresAt` says when. `RefreshAttachmentURLs` turns old URLs into fresh ones.

## Threads and forums

```go
thread, err := client.StartThreadFromMessage(ctx, channelID, msgID, fluxer.ThreadStart{Name: "Bug: login loop"})

post, err := client.CreatePost(ctx, forumID, fluxer.NewPost{
	Name:        "Weekly sync",
	Message:     fluxer.PostMessage{Content: "Agenda in the first reply."},
	AppliedTags: []string{tagID},
})
```

`ArchiveThread`, `LockThread`, `ThreadMembers` and `ArchivedThreads` cover the rest. Forum tags are managed with `CreateForumTag`, `ModifyForumTag` and `DeleteForumTag`.
