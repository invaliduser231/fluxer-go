package fluxertest

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

// AddMessage appends m to its channel's history without dispatching an event. m
// keeps its own ID.
func (s *Server) AddMessage(m fluxer.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[m.ChannelID] = append(s.messages[m.ChannelID], m)
}

// Messages returns a copy of a channel's messages in the order they were added.
func (s *Server) Messages(channelID string) []fluxer.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.messages[channelID])
}

func (s *Server) find(channel, id string) (fluxer.Message, bool) {
	for _, m := range s.messages[channel] {
		if m.ID == id {
			return m, true
		}
	}
	return fluxer.Message{}, false
}

func snapshot(m fluxer.Message) fluxer.MessageSnapshot {
	if len(m.MessageSnapshots) > 0 {
		return m.MessageSnapshots[0]
	}
	snap := fluxer.MessageSnapshot{
		Content: m.Content, Timestamp: m.Timestamp, EditedTimestamp: m.EditedTimestamp, MentionRoles: m.MentionRoles,
		MentionChannels: m.MentionChannels, Embeds: m.Embeds, Attachments: m.Attachments, Stickers: m.Stickers,
		Type: m.Type, Flags: m.Flags,
	}
	for _, u := range m.Mentions {
		snap.Mentions = append(snap.Mentions, u.ID)
	}
	return snap
}

func (s *Server) messageRoutes() {
	update := func(channel, id string, f func(m *fluxer.Message)) (fluxer.Message, bool) {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i := range s.messages[channel] {
			if m := &s.messages[channel][i]; m.ID == id {
				f(m)
				return *m, true
			}
		}
		return fluxer.Message{}, false
	}
	unknown := func(w http.ResponseWriter) { writeError(w, http.StatusNotFound, "UNKNOWN_MESSAGE", "Unknown message") }
	s.Handle("POST /v1/channels/{channel}/messages", func(w http.ResponseWriter, r *http.Request) {
		var body fluxer.MessageSend
		files := decodeBody(r, &body)
		s.mu.Lock()
		m := fluxer.Message{
			ID: s.newID(), ChannelID: r.PathValue("channel"), Author: s.User, Content: body.Content,
			Embeds: body.Embeds, Flags: body.Flags, Nonce: body.Nonce, MessageReference: body.MessageReference,
			Timestamp: time.Now().UTC(),
		}
		switch ref := m.MessageReference; {
		case ref != nil && ref.Type == fluxer.ReferenceForward:
			if body.Content != "" || len(body.Embeds) > 0 || len(files) > 0 || len(body.StickerIDs) > 0 {
				s.mu.Unlock()
				writeError(w, http.StatusBadRequest, "INVALID_FORM_BODY", "Forwarded messages cannot contain content")
				return
			}
			src, ok := s.find(ref.ChannelID, ref.MessageID)
			if !ok {
				s.mu.Unlock()
				unknown(w)
				return
			}
			m.MessageSnapshots = []fluxer.MessageSnapshot{snapshot(src)}
		case ref != nil:
			m.Type = fluxer.MessageReply
		}
		m.Attachments = s.stored(m.ChannelID, files)
		s.messages[m.ChannelID] = append(s.messages[m.ChannelID], m)
		s.mu.Unlock()
		writeJSON(w, m)
	})
	s.Handle("GET /v1/channels/{channel}/messages", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 {
			limit = 50
		}
		before, _ := strconv.ParseUint(r.URL.Query().Get("before"), 10, 64)
		after, _ := strconv.ParseUint(r.URL.Query().Get("after"), 10, 64)
		out := []fluxer.Message{}
		msgs := s.Messages(r.PathValue("channel"))
		for i := len(msgs) - 1; i >= 0 && len(out) < limit; i-- {
			id, _ := strconv.ParseUint(msgs[i].ID, 10, 64)
			if (before == 0 || id < before) && id > after {
				out = append(out, msgs[i])
			}
		}
		writeJSON(w, out)
	})
	s.Handle("GET /v1/channels/{channel}/messages/{message}", func(w http.ResponseWriter, r *http.Request) {
		m, ok := update(r.PathValue("channel"), r.PathValue("message"), func(*fluxer.Message) {})
		if !ok {
			unknown(w)
			return
		}
		writeJSON(w, m)
	})
	s.Handle("PATCH /v1/channels/{channel}/messages/{message}", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body fluxer.MessageEdit
		_ = json.Unmarshal(raw, &body)
		var att struct {
			Attachments *[]struct {
				ID             json.RawMessage `json:"id"`
				Filename       string          `json:"filename"`
				UploadFilename string          `json:"upload_filename"`
				FileSize       int64           `json:"file_size"`
				Title          *string         `json:"title"`
				Description    *string         `json:"description"`
				Flags          *int            `json:"flags"`
			} `json:"attachments"`
		}
		_ = json.Unmarshal(raw, &att)
		s.mu.Lock()
		bot := s.User.ID
		s.mu.Unlock()
		mine := true
		m, ok := update(r.PathValue("channel"), r.PathValue("message"), func(m *fluxer.Message) {
			if mine = m.Author.ID == bot; !mine {
				return
			}
			if body.Content != nil {
				m.Content = *body.Content
			}
			if body.Embeds != nil {
				m.Embeds = *body.Embeds
			}
			if att.Attachments != nil {
				var kept []fluxer.Attachment
				for _, a := range *att.Attachments {
					if a.UploadFilename != "" {
						kept = append(kept, fluxer.Attachment{ID: s.newID(), Filename: a.Filename, Size: a.FileSize})
						continue
					}
					var id string
					_ = json.Unmarshal(a.ID, &id)
					for _, old := range m.Attachments {
						if old.ID == id {
							if a.Title != nil {
								old.Title = *a.Title
							}
							if a.Description != nil {
								old.Description = *a.Description
							}
							if a.Flags != nil {
								old.Flags = *a.Flags
							}
							kept = append(kept, old)
						}
					}
				}
				m.Attachments = kept
			}
			now := time.Now().UTC()
			m.EditedTimestamp = &now
		})
		switch {
		case !ok:
			unknown(w)
		case !mine:
			writeError(w, http.StatusForbidden, "CANNOT_EDIT_OTHER_USER_MESSAGE", "Cannot edit a message authored by another user")
		default:
			writeJSON(w, m)
		}
	})
	s.Handle("DELETE /v1/channels/{channel}/messages/{message}", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		ch, id := r.PathValue("channel"), r.PathValue("message")
		n := len(s.messages[ch])
		s.messages[ch] = slices.DeleteFunc(s.messages[ch], func(m fluxer.Message) bool { return m.ID == id })
		found := len(s.messages[ch]) < n
		s.mu.Unlock()
		if !found {
			unknown(w)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	s.Handle("POST /v1/channels/{channel}/messages/bulk-delete", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			IDs []string `json:"message_ids"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.IDs) < 2 || len(body.IDs) > 100 {
			writeError(w, http.StatusBadRequest, "INVALID_FORM_BODY", "Invalid form body")
			return
		}
		s.mu.Lock()
		ch := r.PathValue("channel")
		s.messages[ch] = slices.DeleteFunc(s.messages[ch], func(m fluxer.Message) bool { return slices.Contains(body.IDs, m.ID) })
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	pin := func(pinned bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if _, ok := update(r.PathValue("channel"), r.PathValue("message"), func(m *fluxer.Message) { m.Pinned = pinned }); !ok {
				unknown(w)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	}
	s.Handle("PUT /v1/channels/{channel}/pins/{message}", pin(true))
	s.Handle("DELETE /v1/channels/{channel}/pins/{message}", pin(false))
	s.Handle("POST /v1/search/messages", func(w http.ResponseWriter, r *http.Request) {
		var q fluxer.MessageSearch
		_ = json.NewDecoder(r.Body).Decode(&q)
		s.mu.Lock()
		res := fluxer.MessageSearchResult{Messages: []fluxer.Message{}, Channels: []fluxer.Channel{}, HitsPerPage: 25, Page: 1}
		for ch, msgs := range s.messages {
			if len(q.ChannelIDs) > 0 && !slices.Contains(q.ChannelIDs, ch) {
				continue
			}
			for _, m := range msgs {
				if strings.Contains(strings.ToLower(m.Content), strings.ToLower(q.Content)) {
					res.Messages = append(res.Messages, m)
				}
			}
		}
		s.mu.Unlock()
		res.Total = len(res.Messages)
		writeJSON(w, res)
	})
	s.Handle("POST /v1/channels/{channel}/typing", noContent)
	s.Handle("PUT /v1/channels/{channel}/messages/{message}/reactions/{emoji}/@me", noContent)
	s.Handle("DELETE /v1/channels/{channel}/messages/{message}/reactions/{emoji}/@me", noContent)
}

func decodeBody(r *http.Request, v any) []fluxer.Attachment {
	kind, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if kind != "multipart/form-data" {
		_ = json.NewDecoder(r.Body).Decode(v)
		return nil
	}
	var files []fluxer.Attachment
	var meta struct {
		Attachments []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Flags       int    `json:"flags"`
			Duration    int    `json:"duration"`
			Waveform    string `json:"waveform"`
		} `json:"attachments"`
	}
	mr := multipart.NewReader(r.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		data, _ := io.ReadAll(part)
		switch name := part.FormName(); {
		case name == "payload_json":
			_ = json.Unmarshal(data, v)
			_ = json.Unmarshal(data, &meta)
		case strings.HasPrefix(name, "files["):
			files = append(files, fluxer.Attachment{Filename: part.FileName(), Size: int64(len(data))})
		}
	}
	for i, m := range meta.Attachments {
		if i < len(files) {
			files[i].Title, files[i].Description, files[i].Flags = m.Title, m.Description, m.Flags
			files[i].Duration, files[i].Waveform = m.Duration, m.Waveform
		}
	}
	return files
}

func (s *Server) stored(channelID string, files []fluxer.Attachment) []fluxer.Attachment {
	for i := range files {
		files[i].ID = s.newID()
		files[i].URL = s.URL + "/media/attachments/" + channelID + "/" + files[i].ID + "/" + files[i].Filename
	}
	return files
}
