package fluxer

import (
	"context"
	"encoding/json"
	"net/http"
)

const opVoiceStateUpdate = 4

// Voice related codes Fluxer reports in GatewayError.Code.
const (
	VoiceP2PConsentRequired = "VOICE_P2P_CONSENT_REQUIRED"
	VoiceP2PUnavailable     = "VOICE_P2P_UNAVAILABLE"
	VoiceChannelFull        = "VOICE_CHANNEL_FULL"
)

// VoiceStateCommand joins, moves or leaves voice through UpdateVoiceState. An empty ChannelID is
// sent as null and leaves voice, an empty GuildID is sent as null for calls outside a guild.
// ConnectionID is sent only when set.
type VoiceStateCommand struct {
	GuildID      string
	ChannelID    string
	ConnectionID string
	SelfMute     bool
	SelfDeaf     bool
	SelfVideo    bool
	SelfStream   bool
}

// MarshalJSON encodes an empty GuildID or ChannelID as null and leaves out an empty ConnectionID.
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

// GatewayError is the payload of a gateway error (op 12), which Session passes to Handler as
// GATEWAY_ERROR. It is both an Event and an error.
type GatewayError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (GatewayError) EventName() string { return "GATEWAY_ERROR" }

// Error returns the code and message.
func (e *GatewayError) Error() string { return "fluxer: gateway refused: " + e.Code + ": " + e.Message }

func init() {
	registerEvents(func() Event { return &GatewayError{} })
}

// UpdateVoiceState sends a voice state update. Fluxer reports a refusal as a GATEWAY_ERROR
// dispatch, for example with VoiceChannelFull.
func (s *Session) UpdateVoiceState(ctx context.Context, v VoiceStateCommand) error {
	return s.Send(ctx, opVoiceStateUpdate, v)
}

// MoveMember moves a member who is in voice to another voice channel.
func (c *Client) MoveMember(ctx context.Context, guildID, userID, channelID string) (*Member, error) {
	return c.ModifyMember(ctx, guildID, userID, MemberEdit{ChannelID: &channelID})
}

// DisconnectMember removes a member from voice by setting their voice channel to null.
func (c *Client) DisconnectMember(ctx context.Context, guildID, userID string) (*Member, error) {
	body := map[string]any{"channel_id": nil}
	var m Member
	return &m, c.Do(ctx, http.MethodPatch, membersPath(guildID)+"/"+userID, body, &m)
}
