// Package voice joins Fluxer voice channels and sends and receives audio.
//
// Fluxer places a session into a channel over the gateway and hands out a
// token for a LiveKit room. A [Manager] does both steps and returns a [Conn]:
//
//	manager := voice.NewManager(session, router)
//	conn, err := manager.Join(ctx, guildID, channelID, voice.Options{SelfDeaf: true})
//	if err != nil {
//		return err
//	}
//	defer conn.Leave(ctx)
//	err = conn.PlayFile(ctx, "intro.ogg")
//
// [Conn.Play] takes any Ogg Opus stream, for example the output of ffmpeg, and
// [Conn.OpusTrack] accepts raw Opus frames.
package voice
