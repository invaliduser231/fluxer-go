package fluxertest

import (
	"cmp"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/invaliduser231/fluxer-go"
)

func (s *Server) command(c *conn, op int, d json.RawMessage) bool {
	s.mu.Lock()
	s.gw.stats.Commands = append(s.gw.stats.Commands, Command{Op: op, D: append(json.RawMessage(nil), d...)})
	sess := c.sess
	s.mu.Unlock()
	switch op {
	case 4:
		s.voiceCommand(c, sess, d)
	case 3, 14:
	case 8:
		var r fluxer.RequestMembers
		_ = json.Unmarshal(d, &r)
		s.dispatchTo(sess, "GUILD_MEMBERS_CHUNK", s.membersChunk(r))
	case 15:
		var r struct {
			GuildIDs []string `json:"guild_ids"`
			Nonce    string   `json:"nonce"`
		}
		_ = json.Unmarshal(d, &r)
		counts := []fluxer.GuildCount{}
		s.mu.Lock()
		for _, id := range r.GuildIDs {
			if _, ok := s.gs.guilds[id]; ok {
				counts = append(counts, fluxer.GuildCount{GuildID: id, MemberCount: len(s.gs.members[id]), OnlineCount: 1})
			}
		}
		s.mu.Unlock()
		s.dispatchTo(sess, "GUILD_COUNTS_UPDATE", fluxer.GuildCountsUpdate{Counts: counts, Nonce: r.Nonce})
	default:
		return false
	}
	return true
}

func (s *Server) membersChunk(r fluxer.RequestMembers) fluxer.GuildMembersChunk {
	s.mu.Lock()
	defer s.mu.Unlock()
	chunk := fluxer.GuildMembersChunk{GuildID: r.GuildID, Members: []fluxer.Member{}, ChunkCount: 1, Nonce: r.Nonce}
	for _, m := range s.gs.members[r.GuildID] {
		switch {
		case len(r.UserIDs) > 0:
			if !slices.Contains(r.UserIDs, m.User.ID) {
				continue
			}
		case r.Query != "":
			if !strings.HasPrefix(strings.ToLower(m.DisplayName()), strings.ToLower(r.Query)) {
				continue
			}
		}
		m.GuildID = ""
		chunk.Members = append(chunk.Members, m)
	}
	slices.SortFunc(chunk.Members, func(a, b fluxer.Member) int { return compareIDs(a.User.ID, b.User.ID) })
	if r.Limit > 0 && len(chunk.Members) > r.Limit {
		chunk.Members = chunk.Members[:r.Limit]
	}
	return chunk
}

func (s *Server) dispatchTo(sess *session, event string, data any) {
	raw, _ := json.Marshal(data)
	s.mu.Lock()
	sess.seq++
	f, _ := json.Marshal(map[string]any{"op": 0, "s": sess.seq, "t": event, "d": json.RawMessage(raw)})
	sess.buffer = append(sess.buffer, frame{seq: sess.seq, data: f})
	c := sess.conn
	s.mu.Unlock()
	if c != nil {
		c.write(f)
	}
}

func (s *Server) Commands(op int) []json.RawMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []json.RawMessage
	for _, c := range s.gw.stats.Commands {
		if c.Op == op {
			out = append(out, c.D)
		}
	}
	return out
}

type voiceState struct {
	endpoint string
	refuse   string
	states   map[string]fluxer.VoiceState
	next     int
}

func (s *Server) SetVoiceEndpoint(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vs.endpoint = url
}

func (s *Server) RefuseNextVoice(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vs.refuse = code
}

func (s *Server) VoiceStates() []fluxer.VoiceState {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []fluxer.VoiceState{}
	for _, v := range s.vs.states {
		out = append(out, v)
	}
	slices.SortFunc(out, func(a, b fluxer.VoiceState) int { return strings.Compare(a.ConnectionID, b.ConnectionID) })
	return out
}

func (s *Server) voiceCommand(c *conn, sess *session, d json.RawMessage) {
	var cmd struct {
		GuildID      *string `json:"guild_id"`
		ChannelID    *string `json:"channel_id"`
		ConnectionID *string `json:"connection_id"`
		SelfMute     bool    `json:"self_mute"`
		SelfDeaf     bool    `json:"self_deaf"`
	}
	_ = json.Unmarshal(d, &cmd)
	s.mu.Lock()
	if s.vs.states == nil {
		s.vs.states = map[string]fluxer.VoiceState{}
	}
	if code := s.vs.refuse; code != "" {
		s.vs.refuse = ""
		s.mu.Unlock()
		c.send(map[string]any{"op": 12, "d": map[string]string{"code": code, "message": "refused"}})
		return
	}
	guild := deref(cmd.GuildID)
	id := deref(cmd.ConnectionID)
	if id == "" && cmd.ChannelID != nil {
		s.vs.next++
		id = "conn-" + strconv.Itoa(s.vs.next)
	}
	old := s.vs.states[id]
	state := fluxer.VoiceState{
		GuildID: guild, ChannelID: deref(cmd.ChannelID), UserID: s.User.ID, ConnectionID: id, SessionID: sess.id,
		SelfMute: cmd.SelfMute, SelfDeaf: cmd.SelfDeaf, Version: old.Version + 1,
	}
	grant := state.ChannelID != "" && old.ChannelID != state.ChannelID
	if state.ChannelID == "" {
		delete(s.vs.states, id)
	} else {
		s.vs.states[id] = state
	}
	endpoint := cmp.Or(s.vs.endpoint, "wss://voice.fluxer.invalid")
	s.mu.Unlock()
	s.dispatchTo(sess, "VOICE_STATE_UPDATE", state)
	if grant {
		s.dispatchTo(sess, "VOICE_SERVER_UPDATE", fluxer.VoiceServerUpdate{
			Token: "voice-token-" + id, Endpoint: endpoint, ConnectionID: id, ChannelID: state.ChannelID, GuildID: guild,
		})
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
