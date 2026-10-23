package fluxertest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

type Request struct {
	Method string
	Path   string
	Body   string
	Reason string
}

type Server struct {
	*httptest.Server
	Token    string
	User     fluxer.User
	Channels map[string]fluxer.Channel

	mu       sync.Mutex
	mux      *http.ServeMux
	requests []Request
	nextID   int
	failNext map[string]failure
	exhaust  map[string]float64
	messages map[string][]fluxer.Message
	gw       gatewayState
	gs       guildState
	vs       voiceState
	oauth    oauthState
	ban      map[string]any
	delay    time.Duration
}

func NewServer(token string) *Server {
	s := &Server{
		Token:    token,
		User:     fluxer.User{ID: "1000", Username: "testbot", Discriminator: "0001", Bot: true},
		Channels: map[string]fluxer.Channel{},
		failNext: map[string]failure{},
		exhaust:  map[string]float64{},
		messages: map[string][]fluxer.Message{},
		nextID:   5000,
		mux:      http.NewServeMux(),
	}
	s.mux.HandleFunc("GET /.well-known/fluxer", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"api_code_version": 1, "endpoints": map[string]string{
			"api_public": s.URL, "gateway": s.GatewayURL(), "media": s.URL + "/media", "webapp": s.URL,
		}, "features": map[string]bool{"voice_enabled": true, "self_hosted": true}})
	})
	s.mux.HandleFunc("GET /v1/users/@me", s.auth(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		user := s.User
		s.mu.Unlock()
		writeJSON(w, user)
	}))
	s.mux.HandleFunc("GET /v1/channels/{channel}", s.auth(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		ch, ok := s.Channels[r.PathValue("channel")]
		s.mu.Unlock()
		if !ok {
			writeError(w, http.StatusNotFound, "UNKNOWN_CHANNEL", "Unknown channel")
			return
		}
		writeJSON(w, ch)
	}))
	s.gatewayRoutes()
	s.messageRoutes()
	s.guildRoutes()
	s.oauthRoutes()
	s.Server = httptest.NewServer(s.mux)
	return s
}

func (s *Server) API() string { return s.URL + "/v1" }

func (s *Server) Client() *fluxer.Client {
	c := fluxer.NewClient(s.Token)
	c.BaseURL = s.API()
	return c
}

func (s *Server) Handle(pattern string, h http.HandlerFunc) {
	s.mux.HandleFunc(pattern, s.auth(h))
}

func (s *Server) HandleRaw(pattern string, h http.HandlerFunc) {
	s.mux.HandleFunc(pattern, h)
}

func (s *Server) SetUser(u fluxer.User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.User = u
}

func (s *Server) SetToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Token = token
}

func (s *Server) AddChannel(ch fluxer.Channel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Channels[ch.ID] = ch
}

func (s *Server) Ban(code string, until time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ban = map[string]any{"code": code, "message": "This IP address is banned."}
	if !until.IsZero() {
		s.ban["expires_at"] = until.UTC().Format(time.RFC3339Nano)
	}
}

func (s *Server) Unban() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ban = nil
}

func (s *Server) SetDelay(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.delay = d
}

func (s *Server) newID() string {
	s.nextID++
	return strconv.Itoa(s.nextID)
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		s.mu.Lock()
		s.requests = append(s.requests, Request{Method: r.Method, Path: r.URL.EscapedPath(), Body: string(body), Reason: r.Header.Get("X-Audit-Log-Reason")})
		fail, failing := s.failNext[r.Pattern]
		delete(s.failNext, r.Pattern)
		ban, banned := s.ban, s.ban != nil
		if after, ok := s.exhaust[r.Pattern]; ok {
			delete(s.exhaust, r.Pattern)
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset-After", strconv.FormatFloat(after, 'f', -1, 64))
		}
		token, isBot := strings.CutPrefix(r.Header.Get("Authorization"), "Bot ")
		known := isBot && token == s.Token
		delay := s.delay
		s.mu.Unlock()
		time.Sleep(delay)
		if banned {
			writeStatus(w, http.StatusForbidden, ban)
			return
		}
		if !known {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "401: Unauthorized")
			return
		}
		if failing {
			writeStatus(w, fail.status, fail.body)
			return
		}
		next(w, r)
	}
}

type failure struct {
	status int
	body   any
}

func (s *Server) FailOnce(pattern string, status int, body any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext[pattern] = failure{status, body}
}

func (s *Server) RateLimitOnce(pattern string, retryAfter float64) {
	s.FailOnce(pattern, http.StatusTooManyRequests, map[string]any{
		"code": "RATE_LIMITED", "message": "You are being rate limited.", "retry_after": retryAfter, "global": false,
	})
}

func (s *Server) ExhaustOnce(pattern string, resetAfter float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exhaust[pattern] = resetAfter
}

func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

func WriteJSON(w http.ResponseWriter, v any) { writeJSON(w, v) }

func WriteError(w http.ResponseWriter, status int, code, msg string) {
	writeError(w, status, code, msg)
}

func writeJSON(w http.ResponseWriter, v any) { writeStatus(w, http.StatusOK, v) }

func writeStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeStatus(w, status, map[string]string{"code": code, "message": msg})
}

func noContent(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }
