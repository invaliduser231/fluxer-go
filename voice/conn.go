package voice

import (
	"context"
	"sync"

	"github.com/invaliduser231/fluxer-go"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
)

type Conn struct {
	m       *Manager
	key     string
	guildID string
	opts    Options

	playing sync.Mutex

	mu           sync.Mutex
	channelID    string
	connectionID string
	room         Room
	selfMute     bool
	selfDeaf     bool
	closed       bool
	moved        chan *fluxer.VoiceServerUpdate
}

func (c *Conn) GuildID() string { return c.guildID }

func (c *Conn) ChannelID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.channelID
}

func (c *Conn) ConnectionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectionID
}

func (c *Conn) Room() Room {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.room
}

func (c *Conn) connect(ctx context.Context, grant *fluxer.VoiceServerUpdate) error {
	cb := lksdk.NewRoomCallback()
	if c.opts.OnTrack != nil {
		cb.OnTrackSubscribed = func(track *webrtc.TrackRemote, _ *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
			if user, _, ok := ParseIdentity(rp.Identity()); ok {
				c.opts.OnTrack(user, track)
			}
		}
	}
	cb.OnDisconnected = func() {}
	connect := c.m.Connect
	if connect == nil {
		connect = connectLiveKit
	}
	room, err := connect(ctx, grant.Endpoint, grant.Token, cb, c.opts.OnTrack != nil)
	if err != nil {
		return err
	}
	c.mu.Lock()
	old := c.room
	c.room = room
	c.channelID, c.connectionID = grant.ChannelID, grant.ConnectionID
	c.mu.Unlock()
	if old != nil {
		old.Disconnect()
	}
	return nil
}

func (c *Conn) regrant(grant *fluxer.VoiceServerUpdate) {
	c.mu.Lock()
	moved := c.moved
	c.mu.Unlock()
	if moved != nil {
		select {
		case moved <- grant:
			return
		default:
		}
	}
	_ = c.connect(context.Background(), grant)
}

func (c *Conn) state() fluxer.VoiceStateCommand {
	c.mu.Lock()
	defer c.mu.Unlock()
	return fluxer.VoiceStateCommand{
		GuildID: c.guildID, ChannelID: c.channelID, ConnectionID: c.connectionID,
		SelfMute: c.selfMute, SelfDeaf: c.selfDeaf,
	}
}

func (c *Conn) SetMute(ctx context.Context, mute bool) error {
	c.mu.Lock()
	c.selfMute = mute
	c.mu.Unlock()
	return c.m.Gateway.UpdateVoiceState(ctx, c.state())
}

func (c *Conn) SetDeaf(ctx context.Context, deaf bool) error {
	c.mu.Lock()
	c.selfDeaf = deaf
	c.mu.Unlock()
	return c.m.Gateway.UpdateVoiceState(ctx, c.state())
}

func (c *Conn) Move(ctx context.Context, channelID string) error {
	moved := make(chan *fluxer.VoiceServerUpdate, 1)
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrClosed
	}
	c.moved = moved
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.moved = nil
		c.mu.Unlock()
	}()
	cmd := c.state()
	cmd.ChannelID = channelID
	if err := c.m.Gateway.UpdateVoiceState(ctx, cmd); err != nil {
		return err
	}
	select {
	case grant := <-moved:
		return c.connect(ctx, grant)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Conn) Leave(ctx context.Context) error {
	cmd := c.state()
	cmd.ChannelID = ""
	c.disconnect()
	return c.m.Gateway.UpdateVoiceState(ctx, cmd)
}

func (c *Conn) disconnect() {
	c.mu.Lock()
	room := c.room
	c.room, c.closed = nil, true
	c.mu.Unlock()
	c.m.forget(c)
	if room != nil {
		room.Disconnect()
	}
}

type liveKitRoom struct{ *lksdk.Room }

func (r liveKitRoom) Publish(track *lksdk.LocalTrack, opts lksdk.TrackPublicationOptions) (func() error, error) {
	pub, err := r.LocalParticipant.PublishTrack(track, &opts)
	if err != nil {
		return nil, err
	}
	return func() error { return r.LocalParticipant.UnpublishTrack(pub.SID()) }, nil
}

func connectLiveKit(ctx context.Context, endpoint, token string, cb *lksdk.RoomCallback, receive bool) (Room, error) {
	room := lksdk.NewRoom(cb)
	if err := room.JoinWithContextAndToken(ctx, endpoint, token, lksdk.WithAutoSubscribe(receive)); err != nil {
		return nil, err
	}
	return liveKitRoom{room}, nil
}
