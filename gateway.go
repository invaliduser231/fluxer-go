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
	ErrInvalidToken = errors.New("fluxer: invalid token")
	ErrNotConnected = errors.New("fluxer: gateway is not connected")
	errReconnect    = errors.New("fluxer: server requested reconnect")
	errZombie       = errors.New("fluxer: heartbeat not acknowledged")
	errNoReady      = errors.New("fluxer: no ready before timeout")
)

type CloseError struct {
	Code   int
	Reason string
}

func (e *CloseError) Error() string {
	return fmt.Sprintf("fluxer: gateway closed with %d: %s", e.Code, e.Reason)
}

type Checkpoint struct {
	SessionID string
	Seq       int64
}

type Shard struct {
	ID    int
	Count int
}

type Session struct {
	URL           string
	Token         string
	Shard         Shard
	Flags         int
	Compress      bool
	Presence      *StatusUpdate
	IgnoredEvents []string
	Handler       func(event string, data json.RawMessage)
	Logger        *slog.Logger
	MinBackoff    time.Duration
	MaxBackoff    time.Duration
	ReadyTimeout  time.Duration
	Pacer         *Pacer
	Observe       func(step string)
	HTTP          *http.Client
	ReadLimit     int64

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

func (s *Session) Checkpoint() Checkpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Checkpoint{SessionID: s.sessionID, Seq: s.seq}
}

func (s *Session) Restore(c Checkpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionID, s.seq = c.SessionID, c.Seq
}

func (s *Session) User() User {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.user
}

func (s *Session) Latency() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latency
}

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
			case 4002, 4010, 4011, 4012:
				return ce
			case 4007:
				s.Restore(Checkpoint{})
			}
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
	go s.watchReady(ctx, conn, cancel, awaiting)
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

func (s *Session) watchReady(ctx context.Context, conn *wire, cancel func(), events <-chan bool) {
	t := time.NewTimer(s.readyTimeout())
	defer t.Stop()
	waiting, resent := true, false
	for {
		select {
		case <-ctx.Done():
			return
		case ready := <-events:
			waiting, resent = !ready, false
			t.Reset(s.readyTimeout())
		case <-t.C:
			if !waiting {
				continue
			}
			if !resent && s.Checkpoint().SessionID == "" && s.identify(ctx, conn) == nil {
				resent = true
				t.Reset(s.readyTimeout())
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
	return s.send(ctx, conn, opResume, map[string]any{"token": s.Token, "session_id": c.SessionID, "seq": c.Seq})
}

func (s *Session) identify(ctx context.Context, conn *wire) error {
	if s.Pacer != nil {
		if err := s.Pacer.Wait(ctx); err != nil {
			return err
		}
	}
	s.observe("identify")
	d := map[string]any{
		"token":      s.Token,
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
	return 30 * time.Second
}
