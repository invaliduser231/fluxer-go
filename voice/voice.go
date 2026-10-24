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

// Errors returned by Manager and Conn.
var (
	// ErrTimeout is returned by Manager.Join when no voice server grant arrives within
	// Manager.Timeout.
	ErrTimeout = errors.New("voice: no voice grant before timeout")
	// ErrClosed is returned when the connection has already been closed.
	ErrClosed = errors.New("voice: connection closed")
)

// Gateway sends voice state updates. *fluxer.Session implements it.
type Gateway interface {
	UpdateVoiceState(ctx context.Context, v fluxer.VoiceStateCommand) error
}

// Room is a joined media room. The default implementation wraps a LiveKit room;
// tests can provide their own through Manager.Connect.
type Room interface {
	// Publish publishes a local track and returns a function that unpublishes it.
	Publish(track *lksdk.LocalTrack, opts lksdk.TrackPublicationOptions) (func() error, error)
	// Disconnect leaves the room.
	Disconnect()
}

// Connector joins the media room at endpoint with token. receive reports whether
// remote tracks should be subscribed.
type Connector func(ctx context.Context, endpoint, token string, cb *lksdk.RoomCallback, receive bool) (Room, error)

// Options configures a connection made by Manager.Join.
type Options struct {
	// SelfMute joins with the bot muted.
	SelfMute bool
	// SelfDeaf joins with the bot deafened.
	SelfDeaf bool
	// OnTrack is called for every remote track the connection subscribes to, with the
	// user ID taken from the participant identity. Remote tracks are only subscribed
	// when OnTrack is set.
	OnTrack func(userID string, track *webrtc.TrackRemote)
	// OnDisconnect is called when the media room drops the connection. It is not called
	// after Leave or Close, or for a room that a move has replaced.
	OnDisconnect func(reason string)
}

// Manager joins voice channels and keeps one connection per guild, or per channel
// for DM calls. It listens on a router for voice server updates, voice state updates
// and VOICE_ gateway errors.
type Manager struct {
	// Gateway sends the voice state commands.
	Gateway Gateway
	// Connect joins the media room. When nil, the manager connects to LiveKit.
	Connect Connector
	// Timeout is how long Join waits for a voice server grant. The default is 10
	// seconds.
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

// NewManager returns a manager that sends commands through gw and registers its
// handlers on r. Close removes them again.
func NewManager(gw Gateway, r *fluxer.Router) *Manager {
	m := &Manager{Gateway: gw, conns: map[string]*Conn{}, pending: map[string]chan result{}}
	m.remove = []func(){
		fluxer.On(r, m.onServer),
		fluxer.On(r, m.onState),
		fluxer.On(r, m.onError),
	}
	return m
}

// Close disconnects every media room and removes the router handlers. It sends no
// voice state update, so the gateway is not told to leave the channels; use
// Conn.Leave for that.
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

// Conn returns the connection for a guild, or nil if there is none. DM call
// connections are not returned.
func (m *Manager) Conn(guildID string) *Conn {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.conns[guildID]
}

// Join joins a voice channel and returns once the media room is connected. An empty
// guildID joins a DM call in channelID. If a connection for the guild already
// exists, Join returns it, moving it first when channelID differs. Only one join per
// guild can run at a time. Join fails with ErrTimeout when no grant arrives within
// Timeout, and with the gateway error when the server refuses the join.
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

// ParseIdentity splits a participant identity of the form
// user_<userID>_<connectionID>.
func ParseIdentity(identity string) (userID, connectionID string, ok bool) {
	rest, ok := strings.CutPrefix(identity, "user_")
	if !ok {
		return "", "", false
	}
	userID, connectionID, ok = strings.Cut(rest, "_")
	return userID, connectionID, ok && userID != "" && connectionID != ""
}
