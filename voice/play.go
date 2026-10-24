package voice

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
)

// Play publishes an Ogg Opus stream and blocks until it has been sent or ctx is
// done, then unpublishes the track. Calls on the same Conn run one after another.
func (c *Conn) Play(ctx context.Context, ogg io.ReadCloser) error {
	c.playing.Lock()
	defer c.playing.Unlock()
	room := c.Room()
	if room == nil {
		_ = ogg.Close()
		return ErrClosed
	}
	done := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(done) }) }
	track, err := lksdk.NewLocalReaderTrack(ogg, webrtc.MimeTypeOpus, lksdk.ReaderTrackWithOnWriteComplete(finish))
	if err != nil {
		_ = ogg.Close()
		return err
	}
	unpublish, err := room.Publish(track, lksdk.TrackPublicationOptions{Name: "audio", Source: livekit.TrackSource_MICROPHONE})
	if err != nil {
		_ = track.Close()
		return err
	}
	defer func() {
		_ = unpublish()
		_ = track.Close()
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PlayFile plays an Ogg Opus file with Play.
func (c *Conn) PlayFile(ctx context.Context, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return c.Play(ctx, f)
}

// OpusTrack publishes a 48 kHz stereo Opus sample track called name, for callers
// that write samples themselves. The returned function unpublishes and closes the
// track.
func (c *Conn) OpusTrack(name string) (*lksdk.LocalTrack, func() error, error) {
	room := c.Room()
	if room == nil {
		return nil, nil, ErrClosed
	}
	track, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2})
	if err != nil {
		return nil, nil, err
	}
	unpublish, err := room.Publish(track, lksdk.TrackPublicationOptions{Name: name, Source: livekit.TrackSource_MICROPHONE, Stereo: true})
	if err != nil {
		_ = track.Close()
		return nil, nil, err
	}
	return track, func() error {
		err := unpublish()
		_ = track.Close()
		return err
	}, nil
}
