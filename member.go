package fluxer

import "time"

type Member struct {
	User                       *User      `json:"user,omitempty"`
	Nick                       *string    `json:"nick,omitempty"`
	Avatar                     *string    `json:"avatar,omitempty"`
	Banner                     *string    `json:"banner,omitempty"`
	AccentColor                *int       `json:"accent_color,omitempty"`
	Roles                      []string   `json:"roles"`
	JoinedAt                   time.Time  `json:"joined_at,omitzero"`
	Mute                       bool       `json:"mute,omitempty"`
	Deaf                       bool       `json:"deaf,omitempty"`
	CommunicationDisabledUntil *time.Time `json:"communication_disabled_until,omitempty"`
	GuildID                    string     `json:"guild_id,omitempty"`
}

func (m Member) DisplayName() string {
	if m.Nick != nil && *m.Nick != "" {
		return *m.Nick
	}
	if m.User != nil {
		return m.User.DisplayName()
	}
	return ""
}

func (m Member) TimedOut() bool {
	return m.CommunicationDisabledUntil != nil && m.CommunicationDisabledUntil.After(time.Now())
}

func MemberAvatarURL(media, guildID, userID, hash string) string {
	if hash == "" || media == "" {
		return ""
	}
	return media + "/guilds/" + guildID + "/users/" + userID + "/avatars/" + hash + ".webp"
}
