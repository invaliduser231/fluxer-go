package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

func reactionPath(channelID, messageID, emoji string) string {
	return messagesPath(channelID) + "/" + messageID + "/reactions/" + url.PathEscape(emoji)
}

func (c *Client) AddReaction(ctx context.Context, channelID, messageID, emoji string) error {
	return c.Do(ctx, http.MethodPut, reactionPath(channelID, messageID, emoji)+"/@me", nil, nil)
}

func (c *Client) RemoveOwnReaction(ctx context.Context, channelID, messageID, emoji string) error {
	return c.Do(ctx, http.MethodDelete, reactionPath(channelID, messageID, emoji)+"/@me", nil, nil)
}

func (c *Client) RemoveReaction(ctx context.Context, channelID, messageID, emoji, userID string) error {
	return c.Do(ctx, http.MethodDelete, reactionPath(channelID, messageID, emoji)+"/"+userID, nil, nil)
}

func (c *Client) RemoveEmojiReactions(ctx context.Context, channelID, messageID, emoji string) error {
	return c.Do(ctx, http.MethodDelete, reactionPath(channelID, messageID, emoji), nil, nil)
}

func (c *Client) RemoveAllReactions(ctx context.Context, channelID, messageID string) error {
	return c.Do(ctx, http.MethodDelete, messagesPath(channelID)+"/"+messageID+"/reactions", nil, nil)
}

func (c *Client) Reactions(ctx context.Context, channelID, messageID, emoji, after string, limit int) ([]User, error) {
	v := url.Values{}
	if after != "" {
		v.Set("after", after)
	}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	path := reactionPath(channelID, messageID, emoji)
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var users []User
	return users, c.Do(ctx, http.MethodGet, path, nil, &users)
}
