package fluxertest

import (
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

// Webhooks returns the webhooks created through the API, sorted by ID.
func (s *Server) Webhooks() []fluxer.Webhook {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []fluxer.Webhook
	for _, w := range s.gs.webhooks {
		out = append(out, w)
	}
	slices.SortFunc(out, func(a, b fluxer.Webhook) int { return compareIDs(a.ID, b.ID) })
	return out
}

func (s *Server) webhookRoutes() {
	s.Handle("POST /v1/channels/{channel}/webhooks", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		s.mu.Lock()
		ch := s.Channels[r.PathValue("channel")]
		owner := s.User
		hook := fluxer.Webhook{ID: s.newID(), GuildID: ch.GuildID, ChannelID: ch.ID, Name: body.Name, Type: fluxer.WebhookIncoming, User: &owner}
		hook.Token = "token-" + hook.ID
		s.gs.webhooks[hook.ID] = hook
		s.mu.Unlock()
		writeJSON(w, hook)
	})
	s.Handle("GET /v1/channels/{channel}/webhooks", func(w http.ResponseWriter, r *http.Request) {
		out := []fluxer.Webhook{}
		for _, hook := range s.Webhooks() {
			if hook.ChannelID == r.PathValue("channel") {
				out = append(out, hook)
			}
		}
		writeJSON(w, out)
	})
	s.mux.HandleFunc("POST /v1/webhooks/{id}/{token}", s.webhook(func(w http.ResponseWriter, r *http.Request, hook fluxer.Webhook) {
		var body fluxer.WebhookMessage
		files := decodeBody(r, &body)
		s.mu.Lock()
		channel := hook.ChannelID
		if t := r.URL.Query().Get("thread_id"); t != "" {
			channel = t
		}
		name := hook.Name
		if body.Username != "" {
			name = body.Username
		}
		m := fluxer.Message{
			ID: s.newID(), ChannelID: channel, WebhookID: hook.ID, Content: body.Content, Embeds: body.Embeds,
			Author: fluxer.User{ID: hook.ID, Username: name, Bot: true}, Timestamp: time.Now().UTC(),
		}
		m.Attachments = s.stored(channel, files)
		s.messages[channel] = append(s.messages[channel], m)
		s.mu.Unlock()
		if r.URL.Query().Get("wait") != "true" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, m)
	}))
	s.mux.HandleFunc("GET /v1/webhooks/{id}/{token}", s.webhook(func(w http.ResponseWriter, r *http.Request, hook fluxer.Webhook) {
		hook.User = nil
		writeJSON(w, hook)
	}))
	s.mux.HandleFunc("DELETE /v1/webhooks/{id}/{token}", s.webhook(func(w http.ResponseWriter, r *http.Request, hook fluxer.Webhook) {
		s.mu.Lock()
		delete(s.gs.webhooks, hook.ID)
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
}

func (s *Server) webhook(h func(w http.ResponseWriter, r *http.Request, hook fluxer.Webhook)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		hook, ok := s.gs.webhooks[r.PathValue("id")]
		s.requests = append(s.requests, Request{Method: r.Method, Path: r.URL.EscapedPath()})
		s.mu.Unlock()
		if !ok || hook.Token != r.PathValue("token") {
			writeError(w, http.StatusNotFound, "UNKNOWN_WEBHOOK", "Unknown webhook")
			return
		}
		h(w, r, hook)
	}
}
