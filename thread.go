package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type ThreadStart struct {
	Name                string `json:"name"`
	Type                int    `json:"type,omitempty"`
	AutoArchiveDuration int    `json:"auto_archive_duration,omitempty"`
	RateLimitPerUser    int    `json:"rate_limit_per_user,omitempty"`
	Invitable           *bool  `json:"invitable,omitempty"`
}

type PostMessage struct {
	Content         string           `json:"content,omitempty"`
	Embeds          []Embed          `json:"embeds,omitempty"`
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
	StickerIDs      []string         `json:"sticker_ids,omitempty"`
	Flags           int              `json:"flags,omitempty"`
}

type NewPost struct {
	Name                string      `json:"name"`
	Message             PostMessage `json:"message"`
	AppliedTags         []string    `json:"applied_tags,omitempty"`
	AutoArchiveDuration int         `json:"auto_archive_duration,omitempty"`
	RateLimitPerUser    int         `json:"rate_limit_per_user,omitempty"`
}

type ThreadList struct {
	Threads []Channel      `json:"threads"`
	Members []ThreadMember `json:"members"`
	HasMore bool           `json:"has_more,omitempty"`
}

type ThreadSearch struct {
	Name     string
	Archived *bool
	Tags     []string
	Limit    int
	Offset   int
}

type ThreadSearchResult struct {
	ThreadList
	TotalResults int `json:"total_results"`
}

func (c *Client) StartThreadFromMessage(ctx context.Context, channelID, messageID string, t ThreadStart) (*Channel, error) {
	t.Type = 0
	var ch Channel
	return &ch, c.Do(ctx, http.MethodPost, messagesPath(channelID)+"/"+messageID+"/threads", t, &ch)
}

func (c *Client) StartThread(ctx context.Context, channelID string, t ThreadStart) (*Channel, error) {
	if t.Type == 0 {
		t.Type = ThreadPublic
	}
	var ch Channel
	return &ch, c.Do(ctx, http.MethodPost, channelPath(channelID)+"/threads", t, &ch)
}

type Post struct {
	Channel
	Message *Message `json:"message,omitempty"`
}

func (c *Client) CreatePost(ctx context.Context, forumID string, p NewPost) (*Post, error) {
	return c.CreatePostFiles(ctx, forumID, p, nil)
}

func (c *Client) CreatePostFiles(ctx context.Context, forumID string, p NewPost, files []File) (*Post, error) {
	var post Post
	return &post, c.DoFiles(ctx, http.MethodPost, channelPath(forumID)+"/threads", p, files, &post)
}

func (c *Client) ArchiveThread(ctx context.Context, threadID string, archived bool) (*Channel, error) {
	return c.ModifyChannel(ctx, threadID, ChannelEdit{Archived: &archived})
}

func (c *Client) LockThread(ctx context.Context, threadID string, locked bool) (*Channel, error) {
	return c.ModifyChannel(ctx, threadID, ChannelEdit{Locked: &locked})
}

func (c *Client) JoinThread(ctx context.Context, threadID string) error {
	return c.Do(ctx, http.MethodPut, channelPath(threadID)+"/thread-members/@me", nil, nil)
}

func (c *Client) LeaveThread(ctx context.Context, threadID string) error {
	return c.Do(ctx, http.MethodDelete, channelPath(threadID)+"/thread-members/@me", nil, nil)
}

func (c *Client) AddThreadMember(ctx context.Context, threadID, userID string) error {
	return c.Do(ctx, http.MethodPut, channelPath(threadID)+"/thread-members/"+userID, nil, nil)
}

func (c *Client) RemoveThreadMember(ctx context.Context, threadID, userID string) error {
	return c.Do(ctx, http.MethodDelete, channelPath(threadID)+"/thread-members/"+userID, nil, nil)
}

func (c *Client) ThreadMember(ctx context.Context, threadID, userID string) (*ThreadMember, error) {
	var m ThreadMember
	return &m, c.Do(ctx, http.MethodGet, channelPath(threadID)+"/thread-members/"+userID, nil, &m)
}

func (c *Client) ThreadMembers(ctx context.Context, threadID string, withMember bool, after string, limit int) ([]ThreadMember, error) {
	v := url.Values{}
	if withMember {
		v.Set("with_member", "true")
	}
	if after != "" {
		v.Set("after", after)
	}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	path := channelPath(threadID) + "/thread-members"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var ms []ThreadMember
	return ms, c.Do(ctx, http.MethodGet, path, nil, &ms)
}

func (c *Client) ActiveThreads(ctx context.Context, guildID string) (*ThreadList, error) {
	var l ThreadList
	return &l, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/threads/active", nil, &l)
}

func (c *Client) ArchivedThreads(ctx context.Context, channelID string, private bool, before time.Time, limit int) (*ThreadList, error) {
	kind := "public"
	if private {
		kind = "private"
	}
	return c.threadList(ctx, channelPath(channelID)+"/threads/archived/"+kind, before, limit)
}

func (c *Client) JoinedArchivedThreads(ctx context.Context, channelID string, before time.Time, limit int) (*ThreadList, error) {
	return c.threadList(ctx, channelPath(channelID)+"/users/@me/threads/archived/private", before, limit)
}

func (c *Client) threadList(ctx context.Context, path string, before time.Time, limit int) (*ThreadList, error) {
	v := url.Values{}
	if !before.IsZero() {
		v.Set("before", before.UTC().Format(time.RFC3339Nano))
	}
	if limit > 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var l ThreadList
	return &l, c.Do(ctx, http.MethodGet, path, nil, &l)
}

func (c *Client) SearchThreads(ctx context.Context, channelID string, q ThreadSearch) (*ThreadSearchResult, error) {
	v := url.Values{}
	if q.Name != "" {
		v.Set("name", q.Name)
	}
	if q.Archived != nil {
		v.Set("archived", strconv.FormatBool(*q.Archived))
	}
	for _, tag := range q.Tags {
		v.Add("tag", tag)
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Offset > 0 {
		v.Set("offset", strconv.Itoa(q.Offset))
	}
	path := channelPath(channelID) + "/threads/search"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var r ThreadSearchResult
	return &r, c.Do(ctx, http.MethodGet, path, nil, &r)
}

func (c *Client) CreateForumTag(ctx context.Context, channelID string, tag ForumTag) (*Channel, error) {
	var ch Channel
	return &ch, c.Do(ctx, http.MethodPost, channelPath(channelID)+"/tags", tag, &ch)
}

func (c *Client) ModifyForumTag(ctx context.Context, channelID string, tag ForumTag) (*Channel, error) {
	var ch Channel
	return &ch, c.Do(ctx, http.MethodPut, channelPath(channelID)+"/tags/"+tag.ID, tag, &ch)
}

func (c *Client) DeleteForumTag(ctx context.Context, channelID, tagID string) (*Channel, error) {
	var ch Channel
	return &ch, c.Do(ctx, http.MethodDelete, channelPath(channelID)+"/tags/"+tagID, nil, &ch)
}
