package fluxer

import (
	"context"
	"net/http"
)

type BulkFailure struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

type EmojiBulkResult struct {
	Success []Emoji       `json:"success"`
	Failed  []BulkFailure `json:"failed"`
}

type StickerBulkResult struct {
	Success []Sticker     `json:"success"`
	Failed  []BulkFailure `json:"failed"`
}

type ExpressionMetadata struct {
	ID           string `json:"id"`
	GuildID      string `json:"guild_id"`
	Name         string `json:"name"`
	Animated     bool   `json:"animated"`
	AllowCloning bool   `json:"allow_cloning"`
}

type ExpressionSource struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Icon     string   `json:"icon,omitempty"`
	Features []string `json:"features"`
}

func (c *Client) CreateEmojis(ctx context.Context, guildID string, emojis []NewEmoji) (*EmojiBulkResult, error) {
	var r EmojiBulkResult
	return &r, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/emojis/bulk", map[string][]NewEmoji{"emojis": emojis}, &r)
}

func (c *Client) CreateStickers(ctx context.Context, guildID string, stickers []NewSticker) (*StickerBulkResult, error) {
	var r StickerBulkResult
	return &r, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/stickers/bulk", map[string][]NewSticker{"stickers": stickers}, &r)
}

func (c *Client) EmojiMetadata(ctx context.Context, emojiID string) (*ExpressionMetadata, error) {
	var m ExpressionMetadata
	return &m, c.Do(ctx, http.MethodGet, "/emojis/"+emojiID+"/metadata", nil, &m)
}

func (c *Client) EmojiSource(ctx context.Context, emojiID string) (*ExpressionSource, error) {
	var s ExpressionSource
	return &s, c.Do(ctx, http.MethodGet, "/emojis/"+emojiID+"/source", nil, &s)
}

func (c *Client) StickerMetadata(ctx context.Context, stickerID string) (*ExpressionMetadata, error) {
	var m ExpressionMetadata
	return &m, c.Do(ctx, http.MethodGet, "/stickers/"+stickerID+"/metadata", nil, &m)
}

func (c *Client) StickerSource(ctx context.Context, stickerID string) (*ExpressionSource, error) {
	var s ExpressionSource
	return &s, c.Do(ctx, http.MethodGet, "/stickers/"+stickerID+"/source", nil, &s)
}
