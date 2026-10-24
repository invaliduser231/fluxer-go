package fluxertest

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/invaliduser231/fluxer-go"
)

// The client credentials accepted by the OAuth2 endpoints.
const (
	OAuthClientID = "900"
	OAuthSecret   = "fake-secret"
)

type oauthCode struct {
	redirect  string
	challenge string
	scope     string
	user      fluxer.User
}

type oauthState struct {
	user    fluxer.User
	codes   map[string]oauthCode
	tokens  map[string]fluxer.User
	refresh map[string]fluxer.User
	next    int
}

func (s *Server) oauthRoutes() {
	s.oauth = oauthState{
		user:    fluxer.User{ID: "1", Username: "owner"},
		codes:   map[string]oauthCode{},
		tokens:  map[string]fluxer.User{},
		refresh: map[string]fluxer.User{},
	}
	s.mux.HandleFunc("GET /v1/oauth2/authorize", s.authorize)
	s.mux.HandleFunc("POST /v1/oauth2/token", s.token)
	s.mux.HandleFunc("POST /v1/oauth2/token/revoke", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		s.mu.Lock()
		delete(s.oauth.tokens, r.PostForm.Get("token"))
		s.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	s.mux.HandleFunc("GET /v1/oauth2/userinfo", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		u, ok := s.oauth.tokens[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
		s.mu.Unlock()
		if !ok {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "401: Unauthorized")
			return
		}
		writeJSON(w, u)
	})
}

// SetOAuthUser sets the user that later authorization codes are issued for. The
// default is a user called owner with ID 1.
func (s *Server) SetOAuthUser(u fluxer.User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.oauth.user = u
}

func (s *Server) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirect, err := url.Parse(q.Get("redirect_uri"))
	if q.Get("client_id") != OAuthClientID || err != nil || !redirect.IsAbs() || q.Get("scope") == "" {
		writeStatus(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	if q.Get("code_challenge") != "" && q.Get("code_challenge_method") != "S256" {
		writeStatus(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	s.mu.Lock()
	s.oauth.next++
	code := "code-" + strconv.Itoa(s.oauth.next)
	s.oauth.codes[code] = oauthCode{redirect: q.Get("redirect_uri"), challenge: q.Get("code_challenge"), scope: q.Get("scope"), user: s.oauth.user}
	s.mu.Unlock()
	back := redirect.Query()
	back.Set("code", code)
	if state := q.Get("state"); state != "" {
		back.Set("state", state)
	}
	redirect.RawQuery = back.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (s *Server) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FORM_BODY", "Input Validation Error")
		return
	}
	id, secret, _ := r.BasicAuth()
	if v := r.PostForm.Get("client_id"); v != "" {
		id = v
	}
	if v := r.PostForm.Get("client_secret"); v != "" {
		secret = v
	}
	if id != OAuthClientID || secret != OAuthSecret {
		writeStatus(w, http.StatusBadRequest, map[string]string{"error": "invalid_client", "error_description": "Invalid client"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var user fluxer.User
	scope := "identify"
	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		code := r.PostForm.Get("code")
		c, ok := s.oauth.codes[code]
		delete(s.oauth.codes, code)
		if !ok || c.redirect != r.PostForm.Get("redirect_uri") || c.challenge != "" && c.challenge != fluxer.Challenge(r.PostForm.Get("code_verifier")) {
			writeStatus(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "Invalid grant"})
			return
		}
		user, scope = c.user, c.scope
	case "refresh_token":
		u, ok := s.oauth.refresh[r.PostForm.Get("refresh_token")]
		if !ok {
			writeStatus(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "Invalid grant"})
			return
		}
		delete(s.oauth.refresh, r.PostForm.Get("refresh_token"))
		user = u
	default:
		writeError(w, http.StatusBadRequest, "INVALID_FORM_BODY", "Input Validation Error")
		return
	}
	s.oauth.next++
	access := "access-" + strconv.Itoa(s.oauth.next)
	s.oauth.tokens[access] = user
	s.oauth.refresh["refresh-"+access] = user
	writeJSON(w, fluxer.OAuthToken{AccessToken: access, TokenType: "Bearer", ExpiresIn: 604800, RefreshToken: "refresh-" + access, Scope: scope})
}
