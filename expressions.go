package fluxer

import (
	"context"
	"net/http"
)

type EmojiCreate struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type StickerCreate struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Image       string   `json:"image"`
}

type StickerEdit struct {
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
}

func (c *Client) Emojis(ctx context.Context, guildID string) ([]Emoji, error) {
	var es []Emoji
	return es, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/emojis", nil, &es)
}

func (c *Client) CreateEmoji(ctx context.Context, guildID string, e EmojiCreate) (*Emoji, error) {
	var out Emoji
	return &out, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/emojis", e, &out)
}

func (c *Client) CloneEmoji(ctx context.Context, guildID, sourceEmojiID string) (*Emoji, error) {
	var out Emoji
	return &out, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/emojis/clone", map[string]string{"source_emoji_id": sourceEmojiID}, &out)
}

func (c *Client) RenameEmoji(ctx context.Context, guildID, emojiID, name string) (*Emoji, error) {
	var out Emoji
	return &out, c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/emojis/"+emojiID, map[string]string{"name": name}, &out)
}

func (c *Client) DeleteEmoji(ctx context.Context, guildID, emojiID string) error {
	return c.Do(ctx, http.MethodDelete, guildPath(guildID)+"/emojis/"+emojiID, nil, nil)
}

func (c *Client) Stickers(ctx context.Context, guildID string) ([]Sticker, error) {
	var ss []Sticker
	return ss, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/stickers", nil, &ss)
}

func (c *Client) CreateSticker(ctx context.Context, guildID string, s StickerCreate) (*Sticker, error) {
	var out Sticker
	return &out, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/stickers", s, &out)
}

func (c *Client) CloneSticker(ctx context.Context, guildID, sourceStickerID string) (*Sticker, error) {
	var out Sticker
	return &out, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/stickers/clone", map[string]string{"source_sticker_id": sourceStickerID}, &out)
}

func (c *Client) ModifySticker(ctx context.Context, guildID, stickerID string, e StickerEdit) (*Sticker, error) {
	var out Sticker
	return &out, c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/stickers/"+stickerID, e, &out)
}

func (c *Client) DeleteSticker(ctx context.Context, guildID, stickerID string) error {
	return c.Do(ctx, http.MethodDelete, guildPath(guildID)+"/stickers/"+stickerID, nil, nil)
}
