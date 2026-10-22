package voice

import (
	"cmp"
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/invaliduser231/fluxer-go"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
)

var (
	ErrTimeout = errors.New("voice: no voice grant before timeout")
	ErrClosed  = errors.New("voice: connection closed")
)

type Gateway interface {
	UpdateVoiceState(ctx context.Context, v fluxer.VoiceStateCommand) error
}

type Room interface {
	Publish(track *lksdk.LocalTrack, opts lksdk.TrackPublicationOptions) (func() error, error)
	Disconnect()
}

type Connector func(ctx context.Context, endpoint, token string, cb *lksdk.RoomCallback, receive bool) (Room, error)

type Options struct {
	SelfMute bool
	SelfDeaf bool
	OnTrack  func(userID string, track *webrtc.TrackRemote)
}

type Manager struct {
	Gateway Gateway
	Connect Connector
	Timeout time.Duration

	mu      sync.Mutex
	conns   map[string]*Conn
	pending map[string]chan result
	remove  []func()
}

type result struct {
	grant *fluxer.VoiceServerUpdate
	err   error
}

func NewManager(gw Gateway, r *fluxer.Router) *Manager {
	m := &Manager{Gateway: gw, conns: map[string]*Conn{}, pending: map[string]chan result{}}
	m.remove = []func(){
		fluxer.On(r, m.onServer),
		fluxer.On(r, m.onState),
		fluxer.On(r, m.onError),
	}
	return m
}

func (m *Manager) Close() {
	m.mu.Lock()
	conns := make([]*Conn, 0, len(m.conns))
	for _, c := range m.conns {
		conns = append(conns, c)
	}
	remove := m.remove
	m.remove = nil
	m.mu.Unlock()
	for _, c := range conns {
		c.disconnect()
	}
	for _, f := range remove {
		f()
	}
}

func scope(guildID, channelID string) string {
	if guildID == "" {
		return "dm:" + channelID
	}
	return guildID
}

func (m *Manager) Conn(guildID string) *Conn {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.conns[guildID]
}

func (m *Manager) Join(ctx context.Context, guildID, channelID string, opts Options) (*Conn, error) {
	key := scope(guildID, channelID)
	m.mu.Lock()
	if c := m.conns[key]; c != nil {
		m.mu.Unlock()
		if c.ChannelID() == channelID {
			return c, nil
		}
		return c, c.Move(ctx, channelID)
	}
	if _, busy := m.pending[key]; busy {
		m.mu.Unlock()
		return nil, errors.New("voice: a join for this guild is already running")
	}
	wait := make(chan result, 1)
	m.pending[key] = wait
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		delete(m.pending, key)
		m.mu.Unlock()
	}()

	cmd := fluxer.VoiceStateCommand{GuildID: guildID, ChannelID: channelID, SelfMute: opts.SelfMute, SelfDeaf: opts.SelfDeaf}
	if err := m.Gateway.UpdateVoiceState(ctx, cmd); err != nil {
		return nil, err
	}
	timer := time.NewTimer(cmp.Or(m.Timeout, 10*time.Second))
	defer timer.Stop()
	var res result
	select {
	case res = <-wait:
	case <-timer.C:
		return nil, ErrTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if res.err != nil {
		return nil, res.err
	}
	c := &Conn{
		m: m, key: key, guildID: guildID, channelID: channelID, connectionID: res.grant.ConnectionID,
		opts: opts, selfMute: opts.SelfMute, selfDeaf: opts.SelfDeaf,
	}
	if err := c.connect(ctx, res.grant); err != nil {
		_ = m.Gateway.UpdateVoiceState(context.WithoutCancel(ctx), fluxer.VoiceStateCommand{GuildID: guildID, ConnectionID: c.connectionID})
		return nil, err
	}
	m.mu.Lock()
	m.conns[key] = c
	m.mu.Unlock()
	return c, nil
}

func (m *Manager) onServer(e *fluxer.VoiceServerUpdate) {
	key := scope(e.GuildID, e.ChannelID)
	m.mu.Lock()
	c := m.conns[key]
	wait := m.pending[key]
	m.mu.Unlock()
	switch {
	case c != nil && c.ConnectionID() == e.ConnectionID:
		go c.regrant(e)
	case wait != nil:
		select {
		case wait <- result{grant: e}:
		default:
		}
	}
}

func (m *Manager) onState(e *fluxer.VoiceStateUpdate) {
	if e.ChannelID != "" {
		return
	}
	m.mu.Lock()
	var gone *Conn
	for key, c := range m.conns {
		if c.ConnectionID() == e.ConnectionID {
			gone = c
			delete(m.conns, key)
		}
	}
	m.mu.Unlock()
	if gone != nil {
		gone.disconnect()
	}
}

func (m *Manager) onError(e *fluxer.GatewayError) {
	if !strings.HasPrefix(e.Code, "VOICE_") {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, wait := range m.pending {
		select {
		case wait <- result{err: e}:
		default:
		}
	}
}

func (m *Manager) forget(c *Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conns[c.key] == c {
		delete(m.conns, c.key)
	}
}

func ParseIdentity(identity string) (userID, connectionID string, ok bool) {
	rest, ok := strings.CutPrefix(identity, "user_")
	if !ok {
		return "", "", false
	}
	userID, connectionID, ok = strings.Cut(rest, "_")
	return userID, connectionID, ok && userID != "" && connectionID != ""
}
