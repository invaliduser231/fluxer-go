package fluxer

import (
	"context"
	"encoding/json"
	"net/http"
)

const opVoiceStateUpdate = 4

const (
	VoiceP2PConsentRequired = "VOICE_P2P_CONSENT_REQUIRED"
	VoiceP2PUnavailable     = "VOICE_P2P_UNAVAILABLE"
	VoiceChannelFull        = "VOICE_CHANNEL_FULL"
)

type VoiceStateCommand struct {
	GuildID      string
	ChannelID    string
	ConnectionID string
	SelfMute     bool
	SelfDeaf     bool
	SelfVideo    bool
	SelfStream   bool
}

func (v VoiceStateCommand) MarshalJSON() ([]byte, error) {
	d := map[string]any{
		"guild_id":    nullable(v.GuildID),
		"channel_id":  nullable(v.ChannelID),
		"self_mute":   v.SelfMute,
		"self_deaf":   v.SelfDeaf,
		"self_video":  v.SelfVideo,
		"self_stream": v.SelfStream,
	}
	if v.ConnectionID != "" {
		d["connection_id"] = v.ConnectionID
	}
	return json.Marshal(d)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

type GatewayError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (GatewayError) EventName() string { return "GATEWAY_ERROR" }

func (e *GatewayError) Error() string { return "fluxer: gateway refused: " + e.Code + ": " + e.Message }

func init() {
	registerEvents(func() Event { return &GatewayError{} })
}

func (s *Session) UpdateVoiceState(ctx context.Context, v VoiceStateCommand) error {
	return s.Send(ctx, opVoiceStateUpdate, v)
}

func (c *Client) MoveMember(ctx context.Context, guildID, userID, channelID string) (*Member, error) {
	return c.ModifyMember(ctx, guildID, userID, MemberEdit{ChannelID: &channelID})
}

func (c *Client) DisconnectMember(ctx context.Context, guildID, userID string) (*Member, error) {
	body := map[string]any{"channel_id": nil}
	var m Member
	return &m, c.Do(ctx, http.MethodPatch, membersPath(guildID)+"/"+userID, body, &m)
}
