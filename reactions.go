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

// AddReaction reacts to a message as the current user. The emoji is a Unicode emoji or
// name:id for a custom emoji, as returned by ReactionEmoji.APIName. It is
// path-escaped before use.
func (c *Client) AddReaction(ctx context.Context, channelID, messageID, emoji string) error {
	return c.Do(ctx, http.MethodPut, reactionPath(channelID, messageID, emoji)+"/@me", nil, nil)
}

// RemoveOwnReaction removes the current user's reaction with emoji from a message.
func (c *Client) RemoveOwnReaction(ctx context.Context, channelID, messageID, emoji string) error {
	return c.Do(ctx, http.MethodDelete, reactionPath(channelID, messageID, emoji)+"/@me", nil, nil)
}

// RemoveReaction removes another user's reaction with emoji from a message.
func (c *Client) RemoveReaction(ctx context.Context, channelID, messageID, emoji, userID string) error {
	return c.Do(ctx, http.MethodDelete, reactionPath(channelID, messageID, emoji)+"/"+userID, nil, nil)
}

// RemoveEmojiReactions removes all reactions with emoji from a message.
func (c *Client) RemoveEmojiReactions(ctx context.Context, channelID, messageID, emoji string) error {
	return c.Do(ctx, http.MethodDelete, reactionPath(channelID, messageID, emoji), nil, nil)
}

// RemoveAllReactions removes every reaction from a message.
func (c *Client) RemoveAllReactions(ctx context.Context, channelID, messageID string) error {
	return c.Do(ctx, http.MethodDelete, messagesPath(channelID)+"/"+messageID+"/reactions", nil, nil)
}

// Reactions lists users who reacted to a message with emoji, after the user ID in after.
// An empty after and a limit of zero or less leave paging to the server. ReactionsPage
// returns the same list with paging information.
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
