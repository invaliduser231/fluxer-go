package fluxertest

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const dropNoticed = 100 * time.Millisecond

type Stats struct {
	Dials      int
	Identifies int
	Resumes    int
	Heartbeats int
	Overlaps   int
	Ignored    []string
	Identified []time.Time
	Identify   json.RawMessage
	Commands   []Command
}

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
}

func (c *conn) send(v any) {
	data, _ := json.Marshal(v)
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = c.ws.Write(ctx, websocket.MessageText, data)
}

type gatewayState struct {
	interval     time.Duration
	dropIdentify int
	rejectResume bool
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

func (s *Server) GatewayURL() string {
	return "ws" + strings.TrimPrefix(s.URL, "http") + "/gateway"
}

func (s *Server) SetHeartbeatInterval(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gw.interval = d
}

func (s *Server) DropNextIdentify() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gw.dropIdentify++
}

func (s *Server) RejectNextResume() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gw.rejectResume = true
}

func (s *Server) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.gw.stats
	st.Ignored = append([]string(nil), st.Ignored...)
	st.Identified = append([]time.Time(nil), st.Identified...)
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
	s.mu.Lock()
	s.gw.conns[c] = true
	interval := s.gw.interval
	s.mu.Unlock()
	defer s.detach(c)

	c.send(map[string]any{"op": 10, "d": map[string]any{"heartbeat_interval": interval.Milliseconds()}})
	for {
		_, data, err := ws.Read(r.Context())
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
			attached, drop := c.sess != nil, s.gw.dropIdentify > 0
			if drop && !attached {
				s.gw.dropIdentify--
			}
			s.mu.Unlock()
			switch {
			case attached:
				ws.Close(4005, "Already authenticated")
				return
			case drop:
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

func (s *Server) identify(c *conn, d json.RawMessage) bool {
	var id struct {
		Token         string   `json:"token"`
		IgnoredEvents []string `json:"ignored_events"`
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
	s.gw.stats.Identified = append(s.gw.stats.Identified, time.Now())
	s.gw.stats.Ignored = id.IgnoredEvents
	s.gw.stats.Identify = append(json.RawMessage(nil), d...)
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
	if r.Token != s.Token || s.gw.rejectResume {
		s.gw.rejectResume = false
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
		c.mu.Lock()
		_ = c.ws.Write(context.Background(), websocket.MessageText, data)
		c.mu.Unlock()
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
		c.mu.Lock()
		_ = c.ws.Write(context.Background(), websocket.MessageText, frames[i])
		c.mu.Unlock()
	}
}

func (s *Server) Drop() {
	for _, c := range s.liveConns() {
		s.detach(c)
		c.ws.CloseNow()
	}
}

func (s *Server) RequestReconnect() {
	for _, c := range s.liveConns() {
		c.send(map[string]any{"op": 7})
		c.ws.Close(4000, "Session drain requested; reconnect to continue")
	}
}

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
