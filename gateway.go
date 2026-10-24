package fluxer

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	opDispatch       = 0
	opHeartbeat      = 1
	opIdentify       = 2
	opResume         = 6
	opReconnect      = 7
	opInvalidSession = 9
	opHello          = 10
	opHeartbeatACK   = 11
	opGatewayError   = 12
)

const (
	statusResumable websocket.StatusCode = 4000
	closeTimeout                         = 2 * time.Second
)

var (
	// ErrInvalidToken is returned by Session.Run when the gateway closes a new session with code
	// 4004. A 4004 during a resume drops the saved session and Run identifies again instead.
	ErrInvalidToken = errors.New("fluxer: invalid token")
	// ErrNotConnected is returned by Session.Send and the command methods while the session has no
	// connection that received READY or RESUMED.
	ErrNotConnected = errors.New("fluxer: gateway is not connected")
	errReconnect    = errors.New("fluxer: server requested reconnect")
	errZombie       = errors.New("fluxer: heartbeat not acknowledged")
	errNoReady      = errors.New("fluxer: no ready before timeout")
)

// CloseError carries the close code and reason the gateway sent when it closed the connection.
type CloseError struct {
	Code   int
	Reason string
}

// Error returns the close code and reason.
func (e *CloseError) Error() string {
	return fmt.Sprintf("fluxer: gateway closed with %d: %s", e.Code, e.Reason)
}

// Checkpoint holds the session ID and last sequence number needed to resume a gateway session.
// Passing one to Session.Restore lets a new process resume a session Fluxer still keeps.
type Checkpoint struct {
	SessionID string
	Seq       int64
}

// Shard identifies one shard by its ID and the total shard count. A zero Count leaves the shard out
// of IDENTIFY.
type Shard struct {
	ID    int
	Count int
}

// Session is a single gateway connection that reconnects and resumes on its own. Set the exported
// fields before calling Run and leave them alone while it runs.
type Session struct {
	// URL is the gateway base URL. Empty means DefaultGateway. The session adds the version,
	// encoding and compression query parameters.
	URL   string
	Token string
	// Shard is sent with IDENTIFY when Shard.Count is above zero.
	Shard Shard
	// Flags is a bitmask of the Flag constants, sent with IDENTIFY when non-zero.
	Flags int
	// Compress negotiates zstd-stream compression. Outgoing commands are compressed once the server
	// has sent its first binary frame.
	Compress bool
	// Presence is the initial presence sent with IDENTIFY.
	Presence *StatusUpdate
	// IgnoredEvents lists dispatch names Fluxer should not send to this session.
	IgnoredEvents []string
	// Handler receives every dispatch with its name and raw data, and op 12 errors as
	// GATEWAY_ERROR. It runs on the gateway read goroutine, so a handler that blocks delays all
	// later events and heartbeat ACKs and can make the session drop the connection as dead.
	Handler func(event string, data json.RawMessage)
	// Logger receives a warning for each ended connection. Nil uses slog.Default.
	Logger *slog.Logger
	// MinBackoff and MaxBackoff bound the jittered delay between reconnect attempts, by default one
	// second and one minute. The delay doubles after each attempt and starts over at MinBackoff
	// once a connection reached READY. MinBackoff is also the wait before identifying again after
	// INVALID_SESSION.
	MinBackoff time.Duration
	MaxBackoff time.Duration
	// ReadyTimeout is how long the session waits for READY or RESUMED before it drops the
	// connection and starts over, 65 seconds by default. Fluxer holds an IDENTIFY while it is busy
	// and drops one over the per-IP budget without a reply for 60 seconds, so a shorter timeout
	// gives up on identifies that would still succeed.
	ReadyTimeout time.Duration
	// Pacer, when set, spaces out IDENTIFY calls, for example across shards that share one identify
	// limit. Resumes are not paced.
	Pacer *Pacer
	// Observe is called at connection milestones with one of connect, connect_failed, identify,
	// resume or disconnect.
	Observe func(step string)
	// HTTP is the client for the WebSocket handshake. Nil uses http.DefaultClient.
	HTTP *http.Client
	// ReadLimit caps the size of a single WebSocket message in bytes, 32 MiB by default.
	ReadLimit int64

	mu        sync.Mutex
	sessionID string
	seq       int64
	latency   time.Duration
	user      User
	live      *wire
}

type payload struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d,omitempty"`
	S  int64           `json:"s,omitempty"`
	T  string          `json:"t,omitempty"`
}

// Checkpoint returns the current session ID and sequence number. It is safe to call while Run is
// active.
func (s *Session) Checkpoint() Checkpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Checkpoint{SessionID: s.sessionID, Seq: s.seq}
}

// Restore replaces the stored session ID and sequence number. Call it before Run to resume from an
// earlier Checkpoint. An empty Checkpoint makes the next connection identify from scratch.
func (s *Session) Restore(c Checkpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionID, s.seq = c.SessionID, c.Seq
}

// User returns the bot user from the most recent READY.
func (s *Session) User() User {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.user
}

// Latency returns the round trip time of the last acknowledged heartbeat, or zero before the first
// ACK.
func (s *Session) Latency() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latency
}

// Run connects to the gateway and keeps the session alive until ctx ends. After a dropped
// connection it waits with jittered backoff and reconnects, resuming when it holds a session ID.
// Close code 4007 discards the session so the next attempt identifies. Run returns ctx.Err() once
// ctx ends, ErrInvalidToken when a new session is closed with 4004, a *BanError when the handshake
// reports an IP ban, and a *CloseError for close codes 4001, 4002, 4003, 4005, 4010, 4011 and 4012,
// which point at a client bug or a setting that another attempt would not fix. Every other error
// leads to another attempt.
func (s *Session) Run(ctx context.Context) error {
	backoff := s.minBackoff()
	for {
		resuming := s.Checkpoint().SessionID != ""
		ready, err := s.connect(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.observe("disconnect")
		if b, ok := errors.AsType[*BanError](err); ok {
			return b
		}
		var ce *CloseError
		if errors.As(err, &ce) {
			switch ce.Code {
			case 4004:
				if !resuming {
					return ErrInvalidToken
				}
				s.Restore(Checkpoint{})
			case 4001, 4002, 4003, 4005, 4010, 4011, 4012:
				return ce
			case 4007:
				s.Restore(Checkpoint{})
			}
		}
		if resuming && errors.Is(err, errNoReady) {
			s.Restore(Checkpoint{})
		}
		if ready {
			backoff = s.minBackoff()
		}
		s.log().Warn("gateway connection ended", "err", err)
		if err := sleep(ctx, backoff/2+rand.N(backoff/2+1)); err != nil {
			return err
		}
		backoff = min(backoff*2, s.maxBackoff())
	}
}

func (s *Session) connect(parent context.Context) (bool, error) {
	u, err := gatewayURL(cmp.Or(s.URL, DefaultGateway), s.Compress)
	if err != nil {
		return false, err
	}
	ws, resp, err := websocket.Dial(parent, u, &websocket.DialOptions{HTTPClient: s.HTTP})
	if err != nil {
		s.observe("connect_failed")
		if b := banned(resp); b != nil {
			return false, b
		}
		return false, err
	}
	s.observe("connect")
	ws.SetReadLimit(cmp.Or(s.ReadLimit, 32<<20))
	conn := &wire{ws: ws, compress: s.Compress}
	defer conn.close()

	ctx, cancel := context.WithCancel(parent)
	rctx, abort := context.WithCancel(context.WithoutCancel(parent))
	closed := make(chan struct{})
	context.AfterFunc(ctx, func() {
		defer close(closed)
		t := time.AfterFunc(closeTimeout, abort)
		defer t.Stop()
		_ = ws.Close(statusResumable, "")
	})
	defer func() {
		s.setLive(nil)
		cancel()
		<-closed
		abort()
	}()

	hello, err := s.read(rctx, conn)
	if err != nil {
		return false, err
	}
	var h struct {
		HeartbeatInterval int64 `json:"heartbeat_interval"`
	}
	if hello.Op != opHello || json.Unmarshal(hello.D, &h) != nil || h.HeartbeatInterval <= 0 {
		return false, fmt.Errorf("fluxer: expected hello, got op %d", hello.Op)
	}
	if err := s.authenticate(ctx, conn); err != nil {
		return false, err
	}

	ready := false
	hb := &beat{}
	awaiting := make(chan bool)
	go s.watchReady(ctx, cancel, awaiting)
	signal := func(isReady bool) {
		hb.setReady(isReady)
		select {
		case awaiting <- isReady:
		case <-ctx.Done():
		}
	}

	beatErr := make(chan error, 1)
	interval := min(time.Duration(h.HeartbeatInterval), 3_600_000) * time.Millisecond
	go func() {
		if err := s.heartbeat(ctx, conn, interval, hb); err != nil {
			beatErr <- err
			cancel()
		}
	}()

	for {
		p, err := s.read(rctx, conn)
		if ctx.Err() != nil && err == nil {
			continue
		}
		if err != nil {
			select {
			case err := <-beatErr:
				return ready, err
			default:
			}
			if !ready && parent.Err() == nil && ctx.Err() != nil {
				return false, errNoReady
			}
			return ready, err
		}
		switch p.Op {
		case opDispatch:
			s.mu.Lock()
			s.seq = max(s.seq, p.S)
			s.mu.Unlock()
			switch p.T {
			case "READY":
				var r Ready
				if err := json.Unmarshal(p.D, &r); err != nil {
					return false, err
				}
				s.mu.Lock()
				s.sessionID, s.user = r.SessionID, r.User
				s.mu.Unlock()
				ready = true
				signal(true)
				s.setLive(conn)
			case "RESUMED":
				ready = true
				signal(true)
				s.setLive(conn)
			}
			if s.Handler != nil {
				s.Handler(p.T, p.D)
			}
		case opHeartbeat:
			if err := s.sendHeartbeat(ctx, conn, hb); err != nil {
				return ready, err
			}
		case opHeartbeatACK:
			if d := hb.ack(); d > 0 {
				s.mu.Lock()
				s.latency = d
				s.mu.Unlock()
			}
		case opGatewayError:
			if s.Handler != nil {
				s.Handler("GATEWAY_ERROR", p.D)
			}
		case opReconnect:
			return ready, errReconnect
		case opInvalidSession:
			s.setLive(nil)
			s.Restore(Checkpoint{})
			if err := sleep(ctx, s.minBackoff()); err != nil {
				return ready, err
			}
			if err := s.identify(ctx, conn); err != nil {
				return ready, err
			}
			ready = false
			signal(false)
		}
	}
}

func (s *Session) watchReady(ctx context.Context, cancel func(), events <-chan bool) {
	t := time.NewTimer(s.readyTimeout())
	defer t.Stop()
	waiting := true
	for {
		select {
		case <-ctx.Done():
			return
		case ready := <-events:
			waiting = !ready
			t.Reset(s.readyTimeout())
		case <-t.C:
			if !waiting {
				continue
			}
			cancel()
			return
		}
	}
}

func (s *Session) authenticate(ctx context.Context, conn *wire) error {
	c := s.Checkpoint()
	if c.SessionID == "" {
		return s.identify(ctx, conn)
	}
	s.observe("resume")
	return s.send(ctx, conn, opResume, map[string]any{"token": s.token(), "session_id": c.SessionID, "seq": c.Seq})
}

func (s *Session) identify(ctx context.Context, conn *wire) error {
	if s.Pacer != nil {
		if err := s.Pacer.Wait(ctx); err != nil {
			return err
		}
	}
	s.observe("identify")
	d := map[string]any{
		"token":      s.token(),
		"properties": map[string]string{"os": runtime.GOOS, "browser": "fluxer-go", "device": "fluxer-go"},
	}
	if len(s.IgnoredEvents) > 0 {
		d["ignored_events"] = s.IgnoredEvents
	}
	if s.Flags != 0 {
		d["flags"] = s.Flags
	}
	if s.Shard.Count > 0 {
		d["shard"] = []int{s.Shard.ID, s.Shard.Count}
	}
	if s.Presence != nil {
		d["presence"] = s.Presence
	}
	return s.send(ctx, conn, opIdentify, d)
}

func (s *Session) send(ctx context.Context, conn *wire, op int, d any) error {
	data, err := json.Marshal(struct {
		Op int `json:"op"`
		D  any `json:"d"`
	}{op, d})
	if err != nil {
		return err
	}
	return conn.write(ctx, data)
}

func (s *Session) read(ctx context.Context, conn *wire) (payload, error) {
	return conn.read(ctx)
}

type beat struct {
	mu     sync.Mutex
	sentAt time.Time
	acked  bool
	ready  bool
}

func (b *beat) setReady(ready bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ready = ready
}

func (b *beat) ack() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sentAt.IsZero() || b.acked {
		return 0
	}
	b.acked = true
	return max(time.Since(b.sentAt), time.Microsecond)
}

func (s *Session) heartbeat(ctx context.Context, conn *wire, interval time.Duration, b *beat) error {
	t := time.NewTimer(rand.N(interval))
	defer t.Stop()
	late := false
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
		b.mu.Lock()
		missed := b.ready && !b.sentAt.IsZero() && !b.acked && time.Since(b.sentAt) >= interval/2
		b.mu.Unlock()
		if missed {
			if late {
				return errZombie
			}
			late = true
			t.Reset(interval / 4)
			continue
		}
		late = false
		if err := s.sendHeartbeat(ctx, conn, b); err != nil {
			return err
		}
		t.Reset(interval)
	}
}

func (s *Session) sendHeartbeat(ctx context.Context, conn *wire, b *beat) error {
	var d any
	if seq := s.Checkpoint().Seq; seq > 0 {
		d = seq
	}
	b.mu.Lock()
	b.sentAt, b.acked = time.Now(), false
	b.mu.Unlock()
	return s.send(ctx, conn, opHeartbeat, d)
}

func (s *Session) token() string {
	return strings.TrimPrefix(strings.TrimSpace(s.Token), "Bot ")
}

func gatewayURL(base string, compress bool) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	if u.Path == "" {
		u.Path = "/"
	}
	q := u.Query()
	q.Set("v", "1")
	q.Set("encoding", "json")
	if compress {
		q.Set("compress", "zstd-stream")
		q.Set("stream", "1")
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *Session) log() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

func (s *Session) observe(step string) {
	if s.Observe != nil {
		s.Observe(step)
	}
}

func (s *Session) minBackoff() time.Duration {
	if s.MinBackoff > 0 {
		return s.MinBackoff
	}
	return time.Second
}

func (s *Session) maxBackoff() time.Duration {
	if s.MaxBackoff > 0 {
		return s.MaxBackoff
	}
	return time.Minute
}

func (s *Session) readyTimeout() time.Duration {
	if s.ReadyTimeout > 0 {
		return s.ReadyTimeout
	}
	return 65 * time.Second
}
