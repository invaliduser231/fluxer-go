package fluxertest

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

type guildState struct {
	guilds   map[string]fluxer.Guild
	members  map[string]map[string]fluxer.Member
	bans     map[string]map[string]fluxer.GuildBan
	webhooks map[string]fluxer.Webhook
}

// AddGuild adds or replaces a guild and registers its channels with the guild ID
// set.
func (s *Server) AddGuild(g fluxer.Guild) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gs.guilds[g.ID] = g
	for _, ch := range g.Channels {
		ch.GuildID = g.ID
		s.Channels[ch.ID] = ch
	}
}

// Guild returns a guild including changes made through the API, or the zero value
// if it does not exist.
func (s *Server) Guild(id string) fluxer.Guild {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gs.guilds[id]
}

// AddMember adds or replaces a member of a guild. m.User must be set, and a zero
// JoinedAt is set to the current time.
func (s *Server) AddMember(guildID string, m fluxer.Member) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gs.members[guildID] == nil {
		s.gs.members[guildID] = map[string]fluxer.Member{}
	}
	if m.JoinedAt.IsZero() {
		m.JoinedAt = time.Now().UTC()
	}
	s.gs.members[guildID][m.User.ID] = m
}

// GuildMember returns a member including changes made through the API.
func (s *Server) GuildMember(guildID, userID string) (fluxer.Member, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.gs.members[guildID][userID]
	return m, ok
}

// Banned reports whether the user was banned through the API.
func (s *Server) Banned(guildID, userID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.gs.bans[guildID][userID]
	return ok
}

func (s *Server) guildRoutes() {
	s.gs = guildState{
		guilds:   map[string]fluxer.Guild{},
		members:  map[string]map[string]fluxer.Member{},
		bans:     map[string]map[string]fluxer.GuildBan{},
		webhooks: map[string]fluxer.Webhook{},
	}
	withGuild := func(h func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			g, ok := s.gs.guilds[r.PathValue("guild")]
			if !ok {
				s.mu.Unlock()
				writeError(w, http.StatusNotFound, "UNKNOWN_GUILD", "Unknown guild")
				return
			}
			h(w, r, &g)
			s.gs.guilds[g.ID] = g
			s.mu.Unlock()
		}
	}
	s.Handle("GET /v1/guilds/{guild}", withGuild(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild) {
		writeJSON(w, g)
	}))
	s.Handle("GET /v1/users/@me/guilds", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		out := []fluxer.Guild{}
		for _, g := range s.gs.guilds {
			out = append(out, fluxer.Guild{ID: g.ID, Name: g.Name, Icon: g.Icon, OwnerID: g.OwnerID, Features: g.Features})
		}
		s.mu.Unlock()
		slices.SortFunc(out, func(a, b fluxer.Guild) int { return compareIDs(a.ID, b.ID) })
		writeJSON(w, out)
	})
	s.Handle("GET /v1/guilds/{guild}/roles", withGuild(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild) {
		writeJSON(w, append([]fluxer.Role{}, g.Roles...))
	}))
	s.Handle("POST /v1/guilds/{guild}/roles", withGuild(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild) {
		var body fluxer.NewRole
		_ = json.NewDecoder(r.Body).Decode(&body)
		role := fluxer.Role{ID: s.newID(), Name: body.Name, Color: body.Color, Permissions: body.Permissions, Position: len(g.Roles)}
		g.Roles = append(g.Roles, role)
		writeJSON(w, role)
	}))
	s.Handle("PATCH /v1/guilds/{guild}/roles/{role}", withGuild(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild) {
		var body fluxer.RoleEdit
		_ = json.NewDecoder(r.Body).Decode(&body)
		i := slices.IndexFunc(g.Roles, func(role fluxer.Role) bool { return role.ID == r.PathValue("role") })
		if i < 0 {
			writeError(w, http.StatusNotFound, "UNKNOWN_ROLE", "Unknown role")
			return
		}
		role := &g.Roles[i]
		if body.Name != nil {
			role.Name = *body.Name
		}
		if body.Color != nil {
			role.Color = *body.Color
		}
		if body.Permissions != nil {
			role.Permissions = *body.Permissions
		}
		if body.Hoist != nil {
			role.Hoist = *body.Hoist
		}
		if body.Mentionable != nil {
			role.Mentionable = *body.Mentionable
		}
		writeJSON(w, role)
	}))
	s.Handle("DELETE /v1/guilds/{guild}/roles/{role}", withGuild(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild) {
		g.Roles = slices.DeleteFunc(g.Roles, func(role fluxer.Role) bool { return role.ID == r.PathValue("role") })
		w.WriteHeader(http.StatusNoContent)
	}))
	s.Handle("GET /v1/guilds/{guild}/channels", withGuild(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild) {
		out := []fluxer.Channel{}
		for _, ch := range s.Channels {
			if ch.GuildID == g.ID {
				out = append(out, ch)
			}
		}
		slices.SortFunc(out, func(a, b fluxer.Channel) int { return a.Position - b.Position })
		writeJSON(w, out)
	}))
	s.Handle("POST /v1/guilds/{guild}/channels", withGuild(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild) {
		var body fluxer.NewChannel
		_ = json.NewDecoder(r.Body).Decode(&body)
		ch := fluxer.Channel{
			ID: s.newID(), GuildID: g.ID, Name: body.Name, Type: body.Type, Topic: body.Topic,
			ParentID: body.ParentID, PermissionOverwrites: body.PermissionOverwrites,
		}
		s.Channels[ch.ID] = ch
		writeJSON(w, ch)
	}))
	s.memberRoutes(withGuild)
	s.webhookRoutes()
}

func (s *Server) memberRoutes(withGuild func(func(http.ResponseWriter, *http.Request, *fluxer.Guild)) http.HandlerFunc) {
	member := func(h func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild, m *fluxer.Member)) http.HandlerFunc {
		return withGuild(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild) {
			id := r.PathValue("user")
			if id == "@me" {
				id = s.User.ID
			}
			m, ok := s.gs.members[g.ID][id]
			if !ok {
				writeError(w, http.StatusNotFound, "UNKNOWN_MEMBER", "Unknown member")
				return
			}
			h(w, r, g, &m)
			if _, still := s.gs.members[g.ID][id]; still {
				s.gs.members[g.ID][id] = m
			}
		})
	}
	s.Handle("GET /v1/guilds/{guild}/members", withGuild(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild) {
		out := []fluxer.Member{}
		for _, m := range s.gs.members[g.ID] {
			if after := r.URL.Query().Get("after"); after == "" || compareIDs(m.User.ID, after) > 0 {
				out = append(out, m)
			}
		}
		slices.SortFunc(out, func(a, b fluxer.Member) int { return compareIDs(a.User.ID, b.User.ID) })
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 {
			limit = 1
		}
		writeJSON(w, out[:min(limit, len(out))])
	}))
	s.Handle("GET /v1/guilds/{guild}/members/{user}", member(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild, m *fluxer.Member) {
		writeJSON(w, m)
	}))
	s.Handle("PATCH /v1/guilds/{guild}/members/{user}", member(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild, m *fluxer.Member) {
		var body map[string]json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&body)
		if raw, ok := body["nick"]; ok {
			var nick *string
			_ = json.Unmarshal(raw, &nick)
			if nick != nil && *nick == "" {
				nick = nil
			}
			m.Nick = nick
		}
		if raw, ok := body["roles"]; ok {
			_ = json.Unmarshal(raw, &m.Roles)
		}
		if raw, ok := body["communication_disabled_until"]; ok {
			var until *time.Time
			_ = json.Unmarshal(raw, &until)
			m.CommunicationDisabledUntil = until
		}
		writeJSON(w, m)
	}))
	s.Handle("DELETE /v1/guilds/{guild}/members/{user}", member(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild, m *fluxer.Member) {
		delete(s.gs.members[g.ID], m.User.ID)
		w.WriteHeader(http.StatusNoContent)
	}))
	s.Handle("PUT /v1/guilds/{guild}/members/{user}/roles/{role}", member(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild, m *fluxer.Member) {
		if !slices.Contains(m.Roles, r.PathValue("role")) {
			m.Roles = append(m.Roles, r.PathValue("role"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	s.Handle("DELETE /v1/guilds/{guild}/members/{user}/roles/{role}", member(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild, m *fluxer.Member) {
		m.Roles = slices.DeleteFunc(m.Roles, func(id string) bool { return id == r.PathValue("role") })
		w.WriteHeader(http.StatusNoContent)
	}))
	s.Handle("GET /v1/guilds/{guild}/bans", withGuild(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild) {
		out := []fluxer.GuildBan{}
		for _, b := range s.gs.bans[g.ID] {
			out = append(out, b)
		}
		writeJSON(w, out)
	}))
	s.Handle("PUT /v1/guilds/{guild}/bans/{user}", withGuild(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild) {
		var body fluxer.BanOptions
		_ = json.NewDecoder(r.Body).Decode(&body)
		if s.gs.bans[g.ID] == nil {
			s.gs.bans[g.ID] = map[string]fluxer.GuildBan{}
		}
		id := r.PathValue("user")
		s.gs.bans[g.ID][id] = fluxer.GuildBan{User: fluxer.User{ID: id}, Reason: body.Reason, ModeratorID: s.User.ID, BannedAt: time.Now().UTC()}
		delete(s.gs.members[g.ID], id)
		w.WriteHeader(http.StatusNoContent)
	}))
	s.Handle("DELETE /v1/guilds/{guild}/bans/{user}", withGuild(func(w http.ResponseWriter, r *http.Request, g *fluxer.Guild) {
		if _, ok := s.gs.bans[g.ID][r.PathValue("user")]; !ok {
			writeError(w, http.StatusNotFound, "USER_IS_NOT_BANNED", "User is not banned")
			return
		}
		delete(s.gs.bans[g.ID], r.PathValue("user"))
		w.WriteHeader(http.StatusNoContent)
	}))
}

func compareIDs(a, b string) int {
	if len(a) != len(b) {
		return len(a) - len(b)
	}
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
