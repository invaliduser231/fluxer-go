package fluxertest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/klauspost/compress/zstd"
)

const dropNoticed = 100 * time.Millisecond

// Stats counts gateway activity since the server started.
type Stats struct {
	// Dials counts gateway connection attempts, including rejected ones.
	Dials int
	// Identifies counts accepted IDENTIFY payloads.
	Identifies int
	// Resumes counts RESUME attempts, including failed ones.
	Resumes    int
	Heartbeats int
	// Overlaps counts sessions started or resumed while another connection with the
	// same token was attached to a session.
	Overlaps int
	// Ignored is the ignored_events list of the last accepted IDENTIFY.
	Ignored []string
	// Identified holds the time of each accepted IDENTIFY.
	Identified []time.Time
	// Identify is the raw payload of the last accepted IDENTIFY.
	Identify json.RawMessage
	// Shards holds the shard field of each accepted IDENTIFY that sent one.
	Shards [][]int
	// Compressed counts connections that negotiated zstd-stream compression.
	Compressed int
	// Commands records every command other than heartbeat, identify and resume that an
	// identified connection sent, in order.
	Commands []Command
}

// Command is a gateway payload received by the server.
type Command struct {
	Op int
	D  json.RawMessage
}

type session struct {
	id      string
	seq     int64
	buffer  []frame
	conn    *conn
	ignored []string
}

type frame struct {
	seq  int64
	data []byte
}

type conn struct {
	ws    *websocket.Conn
	mu    sync.Mutex
	sess  *session
	token string
	held  bool
	zw    *zstd.Encoder
	zbuf  bytes.Buffer
}

func (c *conn) send(v any) {
	data, _ := json.Marshal(v)
	c.write(data)
}

func (c *conn) write(data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if c.zw == nil {
		_ = c.ws.Write(ctx, websocket.MessageText, data)
		return
	}
	c.zbuf.Reset()
	_, _ = c.zw.Write(data)
	_ = c.zw.Flush()
	_ = c.ws.Write(ctx, websocket.MessageBinary, c.zbuf.Bytes())
}

func (c *conn) reader(ctx context.Context) func() ([]byte, error) {
	if c.zw == nil {
		return func() ([]byte, error) {
			_, data, err := c.ws.Read(ctx)
			return data, err
		}
	}
	pr, pw := io.Pipe()
	go func() {
		for {
			_, data, err := c.ws.Read(ctx)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			if _, err := pw.Write(data); err != nil {
				return
			}
		}
	}()
	zr, _ := zstd.NewReader(pr, zstd.WithDecoderConcurrency(1))
	dec := json.NewDecoder(zr)
	return func() ([]byte, error) {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			zr.Close()
			return nil, err
		}
		return raw, nil
	}
}

type gatewayState struct {
	interval     time.Duration
	dropIdentify int
	holdIdentify time.Duration
	rejectResume bool
	ignoreResume bool
	sessions     map[string]*session
	conns        map[*conn]bool
	stats        Stats
	nextSess     int
}

func (s *Server) gatewayRoutes() {
	s.gw = gatewayState{interval: 50 * time.Millisecond, sessions: map[string]*session{}, conns: map[*conn]bool{}}
	s.Handle("GET /v1/gateway/bot", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"url": s.GatewayURL(), "shards": 1, "session_start_limit": map[string]int{
			"total": 1000, "remaining": 1000, "reset_after": 0, "max_concurrency": 1,
		}})
	})
	s.mux.HandleFunc("GET /gateway", s.gateway)
}

// GatewayURL returns the WebSocket URL of the fake gateway.
func (s *Server) GatewayURL() string {
	return "ws" + strings.TrimPrefix(s.URL, "http") + "/gateway"
}

// SetHeartbeatInterval sets the interval sent in HELLO to new connections. The
// default is 50ms.
func (s *Server) SetHeartbeatInterval(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gw.interval = d
}

// DropNextIdentify makes the server ignore the next IDENTIFY without replying or
// closing the connection. Each call drops one more.
func (s *Server) DropNextIdentify() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gw.dropIdentify++
}

// HoldNextIdentify makes the server answer the next IDENTIFY only after d, as
// Fluxer does while it is busy. Another IDENTIFY on the same connection in the
// meantime is closed with 4005.
func (s *Server) HoldNextIdentify(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gw.holdIdentify = d
}

// IgnoreResumes makes the gateway leave every resume unanswered while on is
// true, as a Fluxer node does that cannot reach the session.
func (s *Server) IgnoreResumes(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gw.ignoreResume = on
}

// RejectNextResume makes every RESUME fail with close code 4004 until the next
// accepted IDENTIFY.
func (s *Server) RejectNextResume() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gw.rejectResume = true
}

// Stats returns a copy of the gateway counters.
func (s *Server) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.gw.stats
	st.Ignored = append([]string(nil), st.Ignored...)
	st.Identified = append([]time.Time(nil), st.Identified...)
	st.Shards = append([][]int(nil), st.Shards...)
	st.Commands = append([]Command(nil), st.Commands...)
	return st
}

func (s *Server) gateway(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.gw.stats.Dials++
	ban := s.ban
	s.mu.Unlock()
	if ban != nil {
		writeStatus(w, http.StatusForbidden, ban)
		return
	}
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer ws.CloseNow()
	if r.URL.Query().Get("v") != "1" {
		ws.Close(4012, "Invalid API version")
		return
	}
	c := &conn{ws: ws}
	if q := r.URL.Query(); q.Get("compress") == "zstd-stream" && (q.Get("stream") == "1" || q.Get("stream") == "true") {
		c.zw, _ = zstd.NewWriter(&c.zbuf, zstd.WithEncoderConcurrency(1))
		s.mu.Lock()
		s.gw.stats.Compressed++
		s.mu.Unlock()
	}
	s.mu.Lock()
	s.gw.conns[c] = true
	interval := s.gw.interval
	s.mu.Unlock()
	defer s.detach(c)

	c.send(map[string]any{"op": 10, "d": map[string]any{"heartbeat_interval": interval.Milliseconds()}})
	next := c.reader(r.Context())
	for {
		data, err := next()
		if err != nil {
			if websocket.CloseStatus(err) == -1 {
				time.Sleep(dropNoticed)
			}
			return
		}
		var p struct {
			Op int             `json:"op"`
			D  json.RawMessage `json:"d"`
		}
		if json.Unmarshal(data, &p) != nil {
			ws.Close(4002, "Decode failed")
			return
		}
		switch p.Op {
		case 1:
			s.mu.Lock()
			s.gw.stats.Heartbeats++
			s.mu.Unlock()
			c.send(map[string]any{"op": 11})
		case 2:
			s.mu.Lock()
			attached, drop, hold := c.sess != nil || c.held, s.gw.dropIdentify > 0, s.gw.holdIdentify
			if drop && !attached {
				s.gw.dropIdentify--
			}
			if hold > 0 && !attached && !drop {
				s.gw.holdIdentify, c.held = 0, true
			}
			s.mu.Unlock()
			switch {
			case attached:
				ws.Close(4005, "Already authenticated")
				return
			case drop:
			case hold > 0:
				go s.release(c, p.D, hold)
			case !s.identify(c, p.D):
				ws.Close(4004, "Invalid token")
				return
			}
		case 6:
			if !s.resume(c, p.D) {
				ws.Close(4004, "Invalid token")
				return
			}
		default:
			s.mu.Lock()
			attached := c.sess != nil
			s.mu.Unlock()
			if !attached {
				ws.Close(4003, "Not authenticated")
				return
			}
			if !s.command(c, p.Op, p.D) {
				ws.Close(4001, "Unknown opcode")
				return
			}
		}
	}
}

func (s *Server) release(c *conn, d json.RawMessage, after time.Duration) {
	time.Sleep(after)
	s.mu.Lock()
	open := s.gw.conns[c]
	s.mu.Unlock()
	if open && !s.identify(c, d) {
		c.ws.Close(4004, "Invalid token")
	}
}

func (s *Server) identify(c *conn, d json.RawMessage) bool {
	var id struct {
		Token         string   `json:"token"`
		IgnoredEvents []string `json:"ignored_events"`
		Shard         []int    `json:"shard"`
	}
	if json.Unmarshal(d, &id) != nil {
		return false
	}
	s.mu.Lock()
	if id.Token != s.Token {
		s.mu.Unlock()
		return false
	}
	s.gw.stats.Identifies++
	s.gw.rejectResume = false
	s.gw.stats.Identified = append(s.gw.stats.Identified, time.Now())
	s.gw.stats.Ignored = id.IgnoredEvents
	s.gw.stats.Identify = append(json.RawMessage(nil), d...)
	if id.Shard != nil {
		s.gw.stats.Shards = append(s.gw.stats.Shards, id.Shard)
	}
	s.gw.nextSess++
	sess := &session{id: "session-" + strconv.Itoa(s.gw.nextSess), seq: 1, conn: c, ignored: id.IgnoredEvents}
	s.gw.sessions[sess.id] = sess
	s.attach(c, sess, id.Token)
	user := s.User
	s.mu.Unlock()
	c.send(map[string]any{"op": 0, "s": 1, "t": "READY", "d": map[string]any{
		"session_id": sess.id, "version": 1, "user": user, "guilds": []any{},
	}})
	return true
}

func (s *Server) resume(c *conn, d json.RawMessage) bool {
	var r struct {
		Token     string `json:"token"`
		SessionID string `json:"session_id"`
		Seq       int64  `json:"seq"`
	}
	_ = json.Unmarshal(d, &r)
	s.mu.Lock()
	s.gw.stats.Resumes++
	if s.gw.ignoreResume {
		s.mu.Unlock()
		return true
	}
	if r.Token != s.Token || s.gw.rejectResume {
		s.mu.Unlock()
		return false
	}
	sess := s.gw.sessions[r.SessionID]
	if sess == nil {
		s.mu.Unlock()
		c.send(map[string]any{"op": 9, "d": false})
		return true
	}
	sess.conn = c
	s.attach(c, sess, r.Token)
	var replay [][]byte
	for _, f := range sess.buffer {
		if f.seq > r.Seq {
			replay = append(replay, f.data)
		}
	}
	seq := sess.seq
	s.mu.Unlock()
	for _, data := range replay {
		c.write(data)
	}
	c.send(map[string]any{"op": 0, "s": seq, "t": "RESUMED", "d": map[string]any{}})
	return true
}

func (s *Server) attach(c *conn, sess *session, token string) {
	for other := range s.gw.conns {
		if other != c && other.sess != nil && other.token == token {
			s.gw.stats.Overlaps++
			break
		}
	}
	c.sess, c.token = sess, token
}

func (s *Server) detach(c *conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.gw.conns, c)
	if c.sess != nil && c.sess.conn == c {
		c.sess.conn = nil
	}
}

// Dispatch sends an event to every session that did not list it in ignored_events.
// Events are buffered per session, so a client that resumes receives what it
// missed.
func (s *Server) Dispatch(event string, data any) {
	raw, _ := json.Marshal(data)
	s.mu.Lock()
	var targets []*conn
	var frames [][]byte
	for _, sess := range s.gw.sessions {
		if slices.Contains(sess.ignored, event) {
			continue
		}
		sess.seq++
		f, _ := json.Marshal(map[string]any{"op": 0, "s": sess.seq, "t": event, "d": json.RawMessage(raw)})
		sess.buffer = append(sess.buffer, frame{seq: sess.seq, data: f})
		if sess.conn != nil {
			targets = append(targets, sess.conn)
			frames = append(frames, f)
		}
	}
	s.mu.Unlock()
	for i, c := range targets {
		c.write(frames[i])
	}
}

// Drop closes every gateway connection without a close frame, as if the network
// failed. Sessions stay valid for resuming.
func (s *Server) Drop() {
	for _, c := range s.liveConns() {
		s.detach(c)
		c.ws.CloseNow()
	}
}

// RequestReconnect sends RECONNECT to every connection and closes it with code 4000.
// Sessions stay valid for resuming.
func (s *Server) RequestReconnect() {
	for _, c := range s.liveConns() {
		c.send(map[string]any{"op": 7})
		c.ws.Close(4000, "Session drain requested; reconnect to continue")
	}
}

// InvalidateSessions forgets all sessions and sends a non-resumable INVALID_SESSION
// to every identified connection, so clients have to identify again.
func (s *Server) InvalidateSessions() {
	s.mu.Lock()
	s.gw.sessions = map[string]*session{}
	var conns []*conn
	for c := range s.gw.conns {
		if c.sess != nil {
			c.sess = nil
			conns = append(conns, c)
		}
	}
	s.mu.Unlock()
	for _, c := range conns {
		c.send(map[string]any{"op": 9, "d": false})
	}
}

// Connections returns the number of open gateway connections.
func (s *Server) Connections() int {
	return len(s.liveConns())
}

func (s *Server) liveConns() []*conn {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*conn
	for c := range s.gw.conns {
		out = append(out, c)
	}
	return out
}
