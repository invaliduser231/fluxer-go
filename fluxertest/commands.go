package fluxertest

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/coder/websocket"
	"github.com/invaliduser231/fluxer-go"
)

func (s *Server) command(c *conn, op int, d json.RawMessage) bool {
	s.mu.Lock()
	s.gw.stats.Commands = append(s.gw.stats.Commands, Command{Op: op, D: append(json.RawMessage(nil), d...)})
	sess := c.sess
	s.mu.Unlock()
	switch op {
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
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.ws.Write(context.Background(), websocket.MessageText, f)
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
