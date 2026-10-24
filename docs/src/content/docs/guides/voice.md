---
title: Voice
description: Joining voice channels, playing audio and receiving other participants.
---

Fluxer runs voice through LiveKit. The gateway hands out a token for a LiveKit room, and the media goes over WebRTC. The `voice` module does both steps.

```sh
go get github.com/invaliduser231/fluxer-go/voice
```

## Joining

```go
manager := voice.NewManager(session, router)
defer manager.Close()

conn, err := manager.Join(ctx, guildID, channelID, voice.Options{SelfDeaf: true})
if err != nil {
	return err
}
defer conn.Leave(context.WithoutCancel(ctx))
```

The manager listens on the router for the voice events of your session, so it needs the same router your session feeds.

A refused join returns a `*fluxer.GatewayError`. The code tells you why:

```go
var refused *fluxer.GatewayError
if errors.As(err, &refused) && refused.Code == fluxer.VoiceChannelFull {
	client.Reply(ctx, m, fluxer.MessageSend{Content: "That channel is full."})
}
```

Other refusals, like a missing `CONNECT` permission, send nothing at all. Those end in `voice.ErrTimeout` after ten seconds, or after `manager.Timeout`.

## Playing audio

`Play` takes an Ogg Opus stream and returns when it has finished or the context ends:

```go
err := conn.PlayFile(ctx, "intro.ogg")
```

Anything ffmpeg can read becomes Ogg Opus on the fly:

```go
cmd := exec.CommandContext(ctx, "ffmpeg", "-loglevel", "error", "-i", url,
	"-c:a", "libopus", "-b:a", "96k", "-f", "ogg", "-")
out, err := cmd.StdoutPipe()
if err != nil {
	return err
}
if err := cmd.Start(); err != nil {
	return err
}
defer cmd.Wait()
err = conn.Play(ctx, out)
```

Cancel the context to skip a track. One connection plays one stream at a time; a second `Play` waits for the first.

For a mixer or a synthesizer, `OpusTrack` gives you a track that takes Opus frames:

```go
track, stop, err := conn.OpusTrack("music")
defer stop()
for frame := range frames {
	track.WriteSample(media.Sample{Data: frame, Duration: 20 * time.Millisecond}, nil)
}
```

## Moving, muting, leaving

```go
conn.Move(ctx, otherChannelID)
conn.SetMute(ctx, true)
conn.Leave(ctx)
```

Moving switches to the new room with the new token Fluxer sends. When a moderator disconnects the bot, the connection is closed and removed from the manager.

## Receiving audio

```go
conn, err := manager.Join(ctx, guildID, channelID, voice.Options{
	OnTrack: func(userID string, track *webrtc.TrackRemote) {
		go record(userID, track)
	},
	OnDisconnect: func(reason string) {
		log.Printf("voice dropped: %s", reason)
	},
})
```

`OnTrack` only receives tracks when it is set; without it the bot doesn't subscribe to anything. Participants in a LiveKit room are named `user_<user id>_<connection id>`, and `voice.ParseIdentity` splits that if you work with the room directly through `conn.Room()`.

## Limits

Bots never join peer-to-peer calls. End-to-end encrypted channels admit bots without encryption support, so that case needs nothing from you.
