package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ThreadStart configures StartThread and StartThreadFromMessage. Zero values are
// omitted. Type selects ThreadPublic or ThreadPrivate and is ignored when starting from
// a message.
type ThreadStart struct {
	Name                string `json:"name"`
	Type                int    `json:"type,omitempty"`
	AutoArchiveDuration int    `json:"auto_archive_duration,omitempty"`
	RateLimitPerUser    int    `json:"rate_limit_per_user,omitempty"`
	Invitable           *bool  `json:"invitable,omitempty"`
}

// PostMessage is the first message of a new forum post.
type PostMessage struct {
	Content         string           `json:"content,omitempty"`
	Embeds          []Embed          `json:"embeds,omitempty"`
	AllowedMentions *AllowedMentions `json:"allowed_mentions,omitempty"`
	StickerIDs      []string         `json:"sticker_ids,omitempty"`
	Flags           int              `json:"flags,omitempty"`
}

// NewPost is the body for CreatePost. Name is the post title and AppliedTags holds
// ForumTag IDs.
type NewPost struct {
	Name                string      `json:"name"`
	Message             PostMessage `json:"message"`
	AppliedTags         []string    `json:"applied_tags,omitempty"`
	AutoArchiveDuration int         `json:"auto_archive_duration,omitempty"`
	RateLimitPerUser    int         `json:"rate_limit_per_user,omitempty"`
}

// ThreadList is a set of threads along with the current user's ThreadMember entries for
// them. HasMore is set by the archived thread listings.
type ThreadList struct {
	Threads []Channel      `json:"threads"`
	Members []ThreadMember `json:"members"`
	HasMore bool           `json:"has_more,omitempty"`
}

// ThreadSearch filters SearchThreads. Empty fields are not sent: a nil Archived matches
// both states, and zero Limit and Offset use the server defaults. Each entry in Tags is
// sent as its own tag parameter, and TagSetting, TagsMatchSome or TagsMatchAll, decides
// whether a post needs one of them or all. SortBy is one of the ThreadSort constants and
// SortOrder asc or desc. AfterID and BeforeID limit the results by thread ID.
type ThreadSearch struct {
	Name       string
	Archived   *bool
	Tags       []string
	TagSetting string
	SortBy     string
	SortOrder  string
	AfterID    string
	BeforeID   string
	Limit      int
	Offset     int
}

// Sort keys for ThreadSearch.SortBy.
const (
	ThreadSortLastMessage = "last_message_time"
	ThreadSortArchived    = "archive_time"
	ThreadSortRelevance   = "relevance"
	ThreadSortCreated     = "creation_time"
)

// ThreadSearchResult is the result of SearchThreads, with the total number of matches.
// In forum and media channels FirstMessages holds the first message of each post.
type ThreadSearchResult struct {
	ThreadList
	TotalResults  int       `json:"total_results"`
	FirstMessages []Message `json:"first_messages,omitempty"`
}

// StartThreadFromMessage starts a thread attached to an existing message. Any Type set in
// t is cleared before sending.
func (c *Client) StartThreadFromMessage(ctx context.Context, channelID, messageID string, t ThreadStart) (*Channel, error) {
	t.Type = 0
	var ch Channel
	return &ch, c.Do(ctx, http.MethodPost, messagesPath(channelID)+"/"+messageID+"/threads", t, &ch)
}

// StartThread starts a thread in a channel that is not tied to a message. A zero t.Type
// defaults to ThreadPublic.
func (c *Client) StartThread(ctx context.Context, channelID string, t ThreadStart) (*Channel, error) {
	if t.Type == 0 {
		t.Type = ThreadPublic
	}
	var ch Channel
	return &ch, c.Do(ctx, http.MethodPost, channelPath(channelID)+"/threads", t, &ch)
}

// Post is a forum post: the thread channel and, when returned, its first message.
type Post struct {
	Channel
	Message *Message `json:"message,omitempty"`
}

// CreatePost creates a post in a forum channel.
func (c *Client) CreatePost(ctx context.Context, forumID string, p NewPost) (*Post, error) {
	return c.CreatePostFiles(ctx, forumID, p, nil)
}

// CreatePostFiles creates a forum post with file uploads attached to its first message.
// Unlike SendMessageFiles it sets no nonce, so a failed POST is not retried.
func (c *Client) CreatePostFiles(ctx context.Context, forumID string, p NewPost, files []File) (*Post, error) {
	var post Post
	return &post, c.DoFiles(ctx, http.MethodPost, channelPath(forumID)+"/threads", p, files, &post)
}

// ArchiveThread archives or unarchives a thread.
func (c *Client) ArchiveThread(ctx context.Context, threadID string, archived bool) (*Channel, error) {
	return c.ModifyChannel(ctx, threadID, ChannelEdit{Archived: &archived})
}

// LockThread locks or unlocks a thread.
func (c *Client) LockThread(ctx context.Context, threadID string, locked bool) (*Channel, error) {
	return c.ModifyChannel(ctx, threadID, ChannelEdit{Locked: &locked})
}

// JoinThread adds the current user to a thread.
func (c *Client) JoinThread(ctx context.Context, threadID string) error {
	return c.Do(ctx, http.MethodPut, channelPath(threadID)+"/thread-members/@me", nil, nil)
}

// LeaveThread removes the current user from a thread.
func (c *Client) LeaveThread(ctx context.Context, threadID string) error {
	return c.Do(ctx, http.MethodDelete, channelPath(threadID)+"/thread-members/@me", nil, nil)
}

// AddThreadMember adds a user to a thread.
func (c *Client) AddThreadMember(ctx context.Context, threadID, userID string) error {
	return c.Do(ctx, http.MethodPut, channelPath(threadID)+"/thread-members/"+userID, nil, nil)
}

// RemoveThreadMember removes a user from a thread.
func (c *Client) RemoveThreadMember(ctx context.Context, threadID, userID string) error {
	return c.Do(ctx, http.MethodDelete, channelPath(threadID)+"/thread-members/"+userID, nil, nil)
}

// ThreadMember fetches one user's membership in a thread.
func (c *Client) ThreadMember(ctx context.Context, threadID, userID string) (*ThreadMember, error) {
	var m ThreadMember
	return &m, c.Do(ctx, http.MethodGet, channelPath(threadID)+"/thread-members/"+userID, nil, &m)
}

// ThreadMembers lists the members of a thread after the user ID in after. With
// withMember set, each entry includes the guild Member. An empty after and a limit of
// zero or less leave paging to the server.
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

// ActiveThreads lists all active threads in a guild that the current user can see.
func (c *Client) ActiveThreads(ctx context.Context, guildID string) (*ThreadList, error) {
	var l ThreadList
	return &l, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/threads/active", nil, &l)
}

// ArchivedThreads lists public or private archived threads in a channel, archived before
// the given time. A zero before starts with the most recent, and a limit of zero or less
// uses the server default.
func (c *Client) ArchivedThreads(ctx context.Context, channelID string, private bool, before time.Time, limit int) (*ThreadList, error) {
	kind := "public"
	if private {
		kind = "private"
	}
	return c.threadList(ctx, channelPath(channelID)+"/threads/archived/"+kind, before, limit)
}

// JoinedArchivedThreads lists archived private threads in a channel that the current user
// has joined. before and limit work as in ArchivedThreads.
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

// SearchThreads searches the threads of a channel (GET /channels/{id}/threads/search).
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
	for key, val := range map[string]string{"tag_setting": q.TagSetting, "sort_by": q.SortBy, "sort_order": q.SortOrder, "min_id": q.AfterID, "max_id": q.BeforeID} {
		if val != "" {
			v.Set(key, val)
		}
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

// CreateForumTag adds a tag to a forum channel and returns the updated channel.
func (c *Client) CreateForumTag(ctx context.Context, channelID string, tag ForumTag) (*Channel, error) {
	var ch Channel
	return &ch, c.Do(ctx, http.MethodPost, channelPath(channelID)+"/tags", tag, &ch)
}

// ModifyForumTag replaces the forum tag with ID tag.ID and returns the updated channel.
func (c *Client) ModifyForumTag(ctx context.Context, channelID string, tag ForumTag) (*Channel, error) {
	var ch Channel
	return &ch, c.Do(ctx, http.MethodPut, channelPath(channelID)+"/tags/"+tag.ID, tag, &ch)
}

// DeleteForumTag removes a tag from a forum channel and returns the updated channel.
func (c *Client) DeleteForumTag(ctx context.Context, channelID, tagID string) (*Channel, error) {
	var ch Channel
	return &ch, c.Do(ctx, http.MethodDelete, channelPath(channelID)+"/tags/"+tagID, nil, &ch)
}
