package fluxer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// Audit log action types for AuditLogEntry.ActionType and AuditLogQuery.ActionType.
const (
	AuditGuildUpdate            = 1
	AuditChannelCreate          = 10
	AuditChannelUpdate          = 11
	AuditChannelDelete          = 12
	AuditChannelOverwriteCreate = 13
	AuditChannelOverwriteUpdate = 14
	AuditChannelOverwriteDelete = 15
	AuditMemberKick             = 20
	AuditMemberPrune            = 21
	AuditMemberBanAdd           = 22
	AuditMemberBanRemove        = 23
	AuditMemberUpdate           = 24
	AuditMemberRoleUpdate       = 25
	AuditMemberMove             = 26
	AuditMemberDisconnect       = 27
	AuditBotAdd                 = 28
	AuditRoleCreate             = 30
	AuditRoleUpdate             = 31
	AuditRoleDelete             = 32
	AuditInviteCreate           = 40
	AuditInviteUpdate           = 41
	AuditInviteDelete           = 42
	AuditWebhookCreate          = 50
	AuditWebhookUpdate          = 51
	AuditWebhookDelete          = 52
	AuditEmojiCreate            = 60
	AuditEmojiUpdate            = 61
	AuditEmojiDelete            = 62
	AuditMessageDelete          = 72
	AuditMessageBulkDelete      = 73
	AuditMessagePin             = 74
	AuditMessageUnpin           = 75
	AuditStickerCreate          = 90
	AuditStickerUpdate          = 91
	AuditStickerDelete          = 92
	AuditThreadCreate           = 110
	AuditThreadUpdate           = 111
	AuditThreadDelete           = 112
)

// AuditLogChange is one changed field in an audit log entry. The old and new values are
// kept as raw JSON because their type depends on Key.
type AuditLogChange struct {
	Key      string          `json:"key"`
	OldValue json.RawMessage `json:"old_value,omitempty"`
	NewValue json.RawMessage `json:"new_value,omitempty"`
}

// AuditLogEntry is a single audit log record. UserID is the user who performed the
// action and TargetID the affected object, if any.
type AuditLogEntry struct {
	ID         string           `json:"id"`
	ActionType int              `json:"action_type"`
	UserID     string           `json:"user_id,omitempty"`
	TargetID   string           `json:"target_id,omitempty"`
	Reason     string           `json:"reason,omitempty"`
	Options    map[string]any   `json:"options,omitempty"`
	Changes    []AuditLogChange `json:"changes,omitempty"`
}

// AuditLog is a page of audit log entries plus the users, webhooks and threads they
// reference.
type AuditLog struct {
	Entries  []AuditLogEntry `json:"audit_log_entries"`
	Users    []User          `json:"users"`
	Webhooks []Webhook       `json:"webhooks"`
	Threads  []Channel       `json:"threads,omitempty"`
}

// AuditLogQuery filters AuditLog. Before and After are entry IDs and UserID limits the
// results to one actor. Zero values are not sent.
type AuditLogQuery struct {
	Before     string
	After      string
	UserID     string
	ActionType int
	Limit      int
}

// GuildAuditLogEntryCreate is the gateway event sent when a new audit log entry is
// written in a guild.
type GuildAuditLogEntryCreate struct {
	AuditLogEntry
	GuildID string `json:"guild_id"`
}

// EventName returns the gateway event name GUILD_AUDIT_LOG_ENTRY_CREATE.
func (GuildAuditLogEntryCreate) EventName() string { return "GUILD_AUDIT_LOG_ENTRY_CREATE" }

func init() {
	registerEvents(func() Event { return &GuildAuditLogEntryCreate{} })
}

// AuditLog fetches the audit log of a guild.
func (c *Client) AuditLog(ctx context.Context, guildID string, q AuditLogQuery) (*AuditLog, error) {
	v := url.Values{}
	for key, val := range map[string]string{"before": q.Before, "after": q.After, "user_id": q.UserID} {
		if val != "" {
			v.Set(key, val)
		}
	}
	if q.ActionType > 0 {
		v.Set("action_type", strconv.Itoa(q.ActionType))
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	path := guildPath(guildID) + "/audit-logs"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var l AuditLog
	return &l, c.Do(ctx, http.MethodGet, path, nil, &l)
}
