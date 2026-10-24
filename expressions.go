package fluxer

import (
	"context"
	"net/http"
)

// NewEmoji is the body for CreateEmoji and CreateEmojis. Image is a data URI, see
// ImageData.
type NewEmoji struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

// NewSticker is the body for CreateSticker and CreateStickers. Image is a data URI, see
// ImageData.
type NewSticker struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Image       string   `json:"image"`
}

// StickerEdit holds the changes for ModifySticker. Name is always sent, so it must be set
// even when only Description or Tags change. Nil Description and Tags are left unchanged.
type StickerEdit struct {
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
}

// Emojis lists the custom emoji of a guild.
func (c *Client) Emojis(ctx context.Context, guildID string) ([]Emoji, error) {
	var es []Emoji
	return es, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/emojis", nil, &es)
}

// CreateEmoji uploads a custom emoji to a guild.
func (c *Client) CreateEmoji(ctx context.Context, guildID string, e NewEmoji) (*Emoji, error) {
	var out Emoji
	return &out, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/emojis", e, &out)
}

// CloneEmoji copies an existing emoji from another guild into guildID by its ID.
func (c *Client) CloneEmoji(ctx context.Context, guildID, sourceEmojiID string) (*Emoji, error) {
	var out Emoji
	return &out, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/emojis/clone", map[string]string{"source_emoji_id": sourceEmojiID}, &out)
}

// RenameEmoji changes the name of a custom emoji.
func (c *Client) RenameEmoji(ctx context.Context, guildID, emojiID, name string) (*Emoji, error) {
	var out Emoji
	return &out, c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/emojis/"+emojiID, map[string]string{"name": name}, &out)
}

// DeleteEmoji removes a custom emoji from a guild.
func (c *Client) DeleteEmoji(ctx context.Context, guildID, emojiID string) error {
	return c.Do(ctx, http.MethodDelete, guildPath(guildID)+"/emojis/"+emojiID, nil, nil)
}

// Stickers lists the custom stickers of a guild.
func (c *Client) Stickers(ctx context.Context, guildID string) ([]Sticker, error) {
	var ss []Sticker
	return ss, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/stickers", nil, &ss)
}

// CreateSticker uploads a custom sticker to a guild.
func (c *Client) CreateSticker(ctx context.Context, guildID string, s NewSticker) (*Sticker, error) {
	var out Sticker
	return &out, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/stickers", s, &out)
}

// CloneSticker copies an existing sticker from another guild into guildID by its ID.
func (c *Client) CloneSticker(ctx context.Context, guildID, sourceStickerID string) (*Sticker, error) {
	var out Sticker
	return &out, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/stickers/clone", map[string]string{"source_sticker_id": sourceStickerID}, &out)
}

// ModifySticker applies e to a sticker and returns the updated sticker.
func (c *Client) ModifySticker(ctx context.Context, guildID, stickerID string, e StickerEdit) (*Sticker, error) {
	var out Sticker
	return &out, c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/stickers/"+stickerID, e, &out)
}

// DeleteSticker removes a custom sticker from a guild.
func (c *Client) DeleteSticker(ctx context.Context, guildID, stickerID string) error {
	return c.Do(ctx, http.MethodDelete, guildPath(guildID)+"/stickers/"+stickerID, nil, nil)
}
