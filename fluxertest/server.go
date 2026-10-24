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

// Request is an HTTP request recorded by Server.
type Request struct {
	Method string
	Path   string
	// Body is the raw request body.
	Body string
	// Reason is the X-Audit-Log-Reason header.
	Reason string
}

// Server is a fake Fluxer API and gateway for tests, built on httptest. It serves
// instance discovery, REST routes for users, channels, messages, guilds, members,
// bans, webhooks and OAuth2, and a gateway with sessions, resume and voice.
type Server struct {
	*httptest.Server
	// Token is the bot token the server accepts. Use SetToken once the server is in use.
	Token string
	// User is the bot user returned by /users/@me and READY. Use SetUser once the server
	// is in use.
	User fluxer.User
	// Channels holds the channels the server knows, keyed by ID. Use AddChannel once the
	// server is in use.
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

// NewServer starts a server that accepts token. The bot user is testbot with ID
// 1000. Call Close when done.
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

// API returns the base URL of the REST API.
func (s *Server) API() string { return s.URL + "/v1" }

// Client returns a REST client authenticated with Token and pointed at the server.
func (s *Server) Client() *fluxer.Client {
	c := fluxer.NewClient(s.Token)
	c.BaseURL = s.API()
	return c
}

// Handle adds a route that goes through the same authentication, request log and
// failure injection as the built-in routes.
func (s *Server) Handle(pattern string, h http.HandlerFunc) {
	s.mux.HandleFunc(pattern, s.auth(h))
}

// HandleRaw adds a route without authentication or request logging.
func (s *Server) HandleRaw(pattern string, h http.HandlerFunc) {
	s.mux.HandleFunc(pattern, h)
}

// SetUser replaces the bot user.
func (s *Server) SetUser(u fluxer.User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.User = u
}

// SetToken changes the token the server accepts.
func (s *Server) SetToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Token = token
}

// AddChannel adds or replaces a channel.
func (s *Server) AddChannel(ch fluxer.Channel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Channels[ch.ID] = ch
}

// Ban makes every authenticated route and the gateway answer 403 with code, as for
// a banned IP. A non-zero until is sent as expires_at.
func (s *Server) Ban(code string, until time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ban = map[string]any{"code": code, "message": "This IP address is banned."}
	if !until.IsZero() {
		s.ban["expires_at"] = until.UTC().Format(time.RFC3339Nano)
	}
}

// Unban lifts a ban set by Ban.
func (s *Server) Unban() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ban = nil
}

// SetDelay delays every authenticated request by d.
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

// FailOnce makes the next request to pattern fail with status and body. pattern is
// the route pattern as registered, for example "POST /v1/channels/{channel}/messages".
func (s *Server) FailOnce(pattern string, status int, body any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext[pattern] = failure{status, body}
}

// RateLimitOnce makes the next request to pattern fail with a 429 RATE_LIMITED
// response that asks the client to retry after retryAfter seconds.
func (s *Server) RateLimitOnce(pattern string, retryAfter float64) {
	s.FailOnce(pattern, http.StatusTooManyRequests, map[string]any{
		"code": "RATE_LIMITED", "message": "You are being rate limited.", "retry_after": retryAfter, "global": false,
	})
}

// ExhaustOnce makes the next response for pattern report an empty rate limit bucket
// that resets after resetAfter seconds. The request itself still succeeds.
func (s *Server) ExhaustOnce(pattern string, resetAfter float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exhaust[pattern] = resetAfter
}

// Requests returns a copy of the recorded requests in order. Webhook requests are
// recorded without body and reason.
func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

// WriteJSON writes v as a 200 JSON response, for handlers added with Handle.
func WriteJSON(w http.ResponseWriter, v any) { writeJSON(w, v) }

// WriteError writes a Fluxer error body with status, code and msg.
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
