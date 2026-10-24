package fluxer

import (
	"context"
	"net/http"
)

// BulkFailure reports why one item of a bulk upload, identified by its name, was rejected.
type BulkFailure struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

// EmojiBulkResult is the outcome of CreateEmojis: the emoji that were created and the
// ones that failed.
type EmojiBulkResult struct {
	Success []Emoji       `json:"success"`
	Failed  []BulkFailure `json:"failed"`
}

// StickerBulkResult is the outcome of CreateStickers: the stickers that were created and
// the ones that failed.
type StickerBulkResult struct {
	Success []Sticker     `json:"success"`
	Failed  []BulkFailure `json:"failed"`
}

// ExpressionMetadata describes an emoji or sticker by ID, including the guild it belongs
// to and whether other guilds may clone it.
type ExpressionMetadata struct {
	ID           string `json:"id"`
	GuildID      string `json:"guild_id"`
	Name         string `json:"name"`
	Animated     bool   `json:"animated"`
	AllowCloning bool   `json:"allow_cloning"`
}

// ExpressionSource is the guild an emoji or sticker comes from.
type ExpressionSource struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Icon     string   `json:"icon,omitempty"`
	Features []string `json:"features"`
}

// CreateEmojis uploads several emoji in one request. Individual failures are listed in
// the result instead of failing the whole call.
func (c *Client) CreateEmojis(ctx context.Context, guildID string, emojis []NewEmoji) (*EmojiBulkResult, error) {
	var r EmojiBulkResult
	return &r, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/emojis/bulk", map[string][]NewEmoji{"emojis": emojis}, &r)
}

// CreateStickers uploads several stickers in one request. Individual failures are listed
// in the result instead of failing the whole call.
func (c *Client) CreateStickers(ctx context.Context, guildID string, stickers []NewSticker) (*StickerBulkResult, error) {
	var r StickerBulkResult
	return &r, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/stickers/bulk", map[string][]NewSticker{"stickers": stickers}, &r)
}

// EmojiMetadata fetches metadata for any emoji by ID (GET /emojis/{id}/metadata).
func (c *Client) EmojiMetadata(ctx context.Context, emojiID string) (*ExpressionMetadata, error) {
	var m ExpressionMetadata
	return &m, c.Do(ctx, http.MethodGet, "/emojis/"+emojiID+"/metadata", nil, &m)
}

// EmojiSource fetches the guild an emoji belongs to (GET /emojis/{id}/source).
func (c *Client) EmojiSource(ctx context.Context, emojiID string) (*ExpressionSource, error) {
	var s ExpressionSource
	return &s, c.Do(ctx, http.MethodGet, "/emojis/"+emojiID+"/source", nil, &s)
}

// StickerMetadata fetches metadata for any sticker by ID (GET /stickers/{id}/metadata).
func (c *Client) StickerMetadata(ctx context.Context, stickerID string) (*ExpressionMetadata, error) {
	var m ExpressionMetadata
	return &m, c.Do(ctx, http.MethodGet, "/stickers/"+stickerID+"/metadata", nil, &m)
}

// StickerSource fetches the guild a sticker belongs to (GET /stickers/{id}/source).
func (c *Client) StickerSource(ctx context.Context, stickerID string) (*ExpressionSource, error) {
	var s ExpressionSource
	return &s, c.Do(ctx, http.MethodGet, "/stickers/"+stickerID+"/source", nil, &s)
}
