package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// ReactionPage is one page of users who reacted with an emoji. Pass NextAfter to
// ReactionsPage to fetch the next page while HasMore is true.
type ReactionPage struct {
	Items     []User `json:"items"`
	HasMore   bool   `json:"has_more"`
	NextAfter string `json:"next_after,omitempty"`
}

// FollowerStats counts the channels and guilds following an announcement channel.
type FollowerStats struct {
	ChannelCount int `json:"channel_count"`
	GuildCount   int `json:"guild_count"`
}

// CrosspostSource identifies the guild a crossposted message came from.
type CrosspostSource struct {
	Guild DiscoveryGuild `json:"guild"`
}

// PostData holds the owner and first message of a forum post. Either field may be nil.
type PostData struct {
	Owner        *Member  `json:"owner"`
	FirstMessage *Message `json:"first_message"`
}

// ReactionsPage lists users who reacted to a message with emoji, after the user ID in
// after. An empty after and a limit of zero or less leave paging to the server defaults.
func (c *Client) ReactionsPage(ctx context.Context, channelID, messageID, emoji, after string, limit int) (*ReactionPage, error) {
	v := url.Values{}
	if after != "" {
		v.Set("after", after)
	}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	path := reactionPath(channelID, messageID, emoji) + "/users"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var p ReactionPage
	return &p, c.Do(ctx, http.MethodGet, path, nil, &p)
}

// FollowerStats fetches follower counts for an announcement channel.
func (c *Client) FollowerStats(ctx context.Context, channelID string) (*FollowerStats, error) {
	var s FollowerStats
	return &s, c.Do(ctx, http.MethodGet, channelPath(channelID)+"/follower-stats", nil, &s)
}

// CrosspostSource fetches the origin of a crossposted message.
func (c *Client) CrosspostSource(ctx context.Context, channelID, messageID string) (*CrosspostSource, error) {
	var s CrosspostSource
	return &s, c.Do(ctx, http.MethodGet, messagesPath(channelID)+"/"+messageID+"/crosspost-source", nil, &s)
}

// PostData fetches owner and first-message data for several posts in a forum channel,
// keyed by thread ID.
func (c *Client) PostData(ctx context.Context, forumID string, threadIDs []string) (map[string]PostData, error) {
	var r struct {
		Threads map[string]PostData `json:"threads"`
	}
	err := c.Do(ctx, http.MethodPost, channelPath(forumID)+"/post-data", map[string][]string{"thread_ids": threadIDs}, &r)
	return r.Threads, err
}

// AckPins marks the pins of a channel as seen by the current user.
func (c *Client) AckPins(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodPost, channelPath(channelID)+"/pins/ack", nil, nil)
}

// ClearReadState removes the current user's read state for a channel
// (DELETE /channels/{id}/messages/ack).
func (c *Client) ClearReadState(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodDelete, messagesPath(channelID)+"/ack", nil, nil)
}

// DeleteOwnMessages asks the server to delete all messages the current user sent in a
// channel (POST /channels/{id}/messages/bulk-delete-mine).
func (c *Client) DeleteOwnMessages(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodPost, messagesPath(channelID)+"/bulk-delete-mine", map[string]any{}, nil)
}

// DeleteOwnGuildMessages asks the server to delete all messages the current user sent
// anywhere in a guild.
func (c *Client) DeleteOwnGuildMessages(ctx context.Context, guildID string) error {
	return c.Do(ctx, http.MethodPost, "/users/@me/guilds/"+guildID+"/messages/bulk-delete-mine", map[string]any{}, nil)
}
