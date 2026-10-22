package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type ReactionPage struct {
	Items     []User `json:"items"`
	HasMore   bool   `json:"has_more"`
	NextAfter string `json:"next_after,omitempty"`
}

type FollowerStats struct {
	ChannelCount int `json:"channel_count"`
	GuildCount   int `json:"guild_count"`
}

type CrosspostSource struct {
	Guild DiscoveryGuild `json:"guild"`
}

type PostData struct {
	Owner        *Member  `json:"owner"`
	FirstMessage *Message `json:"first_message"`
}

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

func (c *Client) FollowerStats(ctx context.Context, channelID string) (*FollowerStats, error) {
	var s FollowerStats
	return &s, c.Do(ctx, http.MethodGet, channelPath(channelID)+"/follower-stats", nil, &s)
}

func (c *Client) CrosspostSource(ctx context.Context, channelID, messageID string) (*CrosspostSource, error) {
	var s CrosspostSource
	return &s, c.Do(ctx, http.MethodGet, messagesPath(channelID)+"/"+messageID+"/crosspost-source", nil, &s)
}

func (c *Client) PostData(ctx context.Context, forumID string, threadIDs []string) (map[string]PostData, error) {
	var r struct {
		Threads map[string]PostData `json:"threads"`
	}
	err := c.Do(ctx, http.MethodPost, channelPath(forumID)+"/post-data", map[string][]string{"thread_ids": threadIDs}, &r)
	return r.Threads, err
}

func (c *Client) AckPins(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodPost, channelPath(channelID)+"/pins/ack", nil, nil)
}

func (c *Client) ClearReadState(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodDelete, messagesPath(channelID)+"/ack", nil, nil)
}

func (c *Client) DeleteOwnMessages(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodPost, messagesPath(channelID)+"/bulk-delete-mine", map[string]any{}, nil)
}

func (c *Client) DeleteOwnGuildMessages(ctx context.Context, guildID string) error {
	return c.Do(ctx, http.MethodPost, "/users/@me/guilds/"+guildID+"/messages/bulk-delete-mine", map[string]any{}, nil)
}
