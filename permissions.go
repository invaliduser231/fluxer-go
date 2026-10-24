package fluxer

import (
	"fmt"
	"math/bits"
	"strconv"
	"strings"
)

// Permissions is a set of permission bits. In JSON it is a decimal string, the
// form the Fluxer API uses.
type Permissions uint64

// Permission bits used by the Fluxer API.
const (
	PermCreateInstantInvite   Permissions = 1 << 0
	PermKickMembers           Permissions = 1 << 1
	PermBanMembers            Permissions = 1 << 2
	PermAdministrator         Permissions = 1 << 3
	PermManageChannels        Permissions = 1 << 4
	PermManageGuild           Permissions = 1 << 5
	PermAddReactions          Permissions = 1 << 6
	PermViewAuditLog          Permissions = 1 << 7
	PermPrioritySpeaker       Permissions = 1 << 8
	PermStream                Permissions = 1 << 9
	PermViewChannel           Permissions = 1 << 10
	PermSendMessages          Permissions = 1 << 11
	PermSendTTSMessages       Permissions = 1 << 12
	PermManageMessages        Permissions = 1 << 13
	PermEmbedLinks            Permissions = 1 << 14
	PermAttachFiles           Permissions = 1 << 15
	PermReadMessageHistory    Permissions = 1 << 16
	PermMentionEveryone       Permissions = 1 << 17
	PermUseExternalEmojis     Permissions = 1 << 18
	PermConnect               Permissions = 1 << 20
	PermSpeak                 Permissions = 1 << 21
	PermMuteMembers           Permissions = 1 << 22
	PermDeafenMembers         Permissions = 1 << 23
	PermMoveMembers           Permissions = 1 << 24
	PermUseVAD                Permissions = 1 << 25
	PermChangeNickname        Permissions = 1 << 26
	PermManageNicknames       Permissions = 1 << 27
	PermManageRoles           Permissions = 1 << 28
	PermManageWebhooks        Permissions = 1 << 29
	PermManageExpressions     Permissions = 1 << 30
	PermManageThreads         Permissions = 1 << 34
	PermCreatePublicThreads   Permissions = 1 << 35
	PermCreatePrivateThreads  Permissions = 1 << 36
	PermUseExternalStickers   Permissions = 1 << 37
	PermSendMessagesInThreads Permissions = 1 << 38
	PermModerateMembers       Permissions = 1 << 40
	PermCreateExpressions     Permissions = 1 << 43
	PermPinMessages           Permissions = 1 << 51
	PermBypassSlowmode        Permissions = 1 << 52
	PermUpdateRTCRegion       Permissions = 1 << 53
	PermViewChannelMembers    Permissions = 1 << 54

	// PermAll has every bit set. BasePermissions returns it for the guild owner
	// and for members with PermAdministrator.
	PermAll Permissions = 1<<64 - 1
)

var permissionNames = map[Permissions]string{
	PermCreateInstantInvite: "CREATE_INSTANT_INVITE", PermKickMembers: "KICK_MEMBERS", PermBanMembers: "BAN_MEMBERS",
	PermAdministrator: "ADMINISTRATOR", PermManageChannels: "MANAGE_CHANNELS", PermManageGuild: "MANAGE_GUILD",
	PermAddReactions: "ADD_REACTIONS", PermViewAuditLog: "VIEW_AUDIT_LOG", PermPrioritySpeaker: "PRIORITY_SPEAKER",
	PermStream: "STREAM", PermViewChannel: "VIEW_CHANNEL", PermSendMessages: "SEND_MESSAGES",
	PermSendTTSMessages: "SEND_TTS_MESSAGES", PermManageMessages: "MANAGE_MESSAGES", PermEmbedLinks: "EMBED_LINKS",
	PermAttachFiles: "ATTACH_FILES", PermReadMessageHistory: "READ_MESSAGE_HISTORY", PermMentionEveryone: "MENTION_EVERYONE",
	PermUseExternalEmojis: "USE_EXTERNAL_EMOJIS", PermConnect: "CONNECT", PermSpeak: "SPEAK",
	PermMuteMembers: "MUTE_MEMBERS", PermDeafenMembers: "DEAFEN_MEMBERS", PermMoveMembers: "MOVE_MEMBERS",
	PermUseVAD: "USE_VAD", PermChangeNickname: "CHANGE_NICKNAME", PermManageNicknames: "MANAGE_NICKNAMES",
	PermManageRoles: "MANAGE_ROLES", PermManageWebhooks: "MANAGE_WEBHOOKS", PermManageExpressions: "MANAGE_EXPRESSIONS",
	PermManageThreads: "MANAGE_THREADS", PermCreatePublicThreads: "CREATE_PUBLIC_THREADS", PermCreatePrivateThreads: "CREATE_PRIVATE_THREADS",
	PermUseExternalStickers: "USE_EXTERNAL_STICKERS", PermSendMessagesInThreads: "SEND_MESSAGES_IN_THREADS", PermModerateMembers: "MODERATE_MEMBERS",
	PermCreateExpressions: "CREATE_EXPRESSIONS", PermPinMessages: "PIN_MESSAGES", PermBypassSlowmode: "BYPASS_SLOWMODE",
	PermUpdateRTCRegion: "UPDATE_RTC_REGION", PermViewChannelMembers: "VIEW_CHANNEL_MEMBERS",
}

// Has reports whether p contains every bit of perms.
func (p Permissions) Has(perms Permissions) bool { return p&perms == perms }

// Add returns p with the bits of perms set.
func (p Permissions) Add(perms Permissions) Permissions { return p | perms }

// Remove returns p with the bits of perms cleared.
func (p Permissions) Remove(perms Permissions) Permissions { return p &^ perms }

// Missing returns the bits of perms that p lacks.
func (p Permissions) Missing(perms Permissions) Permissions { return perms &^ p }

// Names returns the API names of the bits set in p, such as SEND_MESSAGES,
// from the lowest bit up. Bits without a known name are skipped.
func (p Permissions) Names() []string {
	var names []string
	for rest := p; rest != 0; rest &= rest - 1 {
		bit := Permissions(1) << bits.TrailingZeros64(uint64(rest))
		if name, ok := permissionNames[bit]; ok {
			names = append(names, name)
		}
	}
	return names
}

// String returns the names of the set bits joined by "|", or "NONE" when p is
// zero.
func (p Permissions) String() string {
	if p == 0 {
		return "NONE"
	}
	names := p.Names()
	var known Permissions
	for bit := range permissionNames {
		if p.Has(bit) {
			known |= bit
		}
	}
	if rest := p &^ known; rest != 0 {
		names = append(names, "0x"+strconv.FormatUint(uint64(rest), 16))
	}
	return strings.Join(names, "|")
}

// MarshalJSON encodes p as a quoted decimal number.
func (p Permissions) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatUint(uint64(p), 10) + `"`), nil
}

// UnmarshalJSON accepts a decimal number, quoted or not. null and an empty
// string decode as zero.
func (p *Permissions) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" {
		*p = 0
		return nil
	}
	s = strings.Trim(s, `"`)
	if s == "" {
		*p = 0
		return nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("fluxer: permissions %s are not a decimal number", data)
	}
	*p = Permissions(v)
	return nil
}
