package fluxer

import (
	"context"
	"iter"
	"net/http"
)

// MessageSearch is the query for SearchMessages. Zero values are omitted. Scope together
// with ContextGuildID or ContextChannelID decides where the search runs. Results can be
// paged with Page or, when the server returns one, by passing the previous result's Cursor.
// AuthorTypes takes user, bot and webhook; Has takes image, sound, video, file, sticker,
// embed, link, poll and snapshot; EmbedTypes takes image, video, sound and article. Each
// Exclude field drops messages that match it.
type MessageSearch struct {
	Content                     string   `json:"content,omitempty"`
	Contents                    []string `json:"contents,omitempty"`
	ExactPhrases                []string `json:"exact_phrases,omitempty"`
	ChannelIDs                  []string `json:"channel_id,omitempty"`
	ExcludeChannelIDs           []string `json:"exclude_channel_id,omitempty"`
	AuthorIDs                   []string `json:"author_id,omitempty"`
	ExcludeAuthorIDs            []string `json:"exclude_author_id,omitempty"`
	AuthorTypes                 []string `json:"author_type,omitempty"`
	ExcludeAuthorTypes          []string `json:"exclude_author_type,omitempty"`
	Mentions                    []string `json:"mentions,omitempty"`
	ExcludeMentions             []string `json:"exclude_mentions,omitempty"`
	MentionEveryone             *bool    `json:"mention_everyone,omitempty"`
	Has                         []string `json:"has,omitempty"`
	ExcludeHas                  []string `json:"exclude_has,omitempty"`
	EmbedTypes                  []string `json:"embed_type,omitempty"`
	ExcludeEmbedTypes           []string `json:"exclude_embed_type,omitempty"`
	EmbedProviders              []string `json:"embed_provider,omitempty"`
	ExcludeEmbedProviders       []string `json:"exclude_embed_provider,omitempty"`
	LinkHostnames               []string `json:"link_hostname,omitempty"`
	ExcludeLinkHostnames        []string `json:"exclude_link_hostname,omitempty"`
	AttachmentFilenames         []string `json:"attachment_filename,omitempty"`
	ExcludeAttachmentFilenames  []string `json:"exclude_attachment_filename,omitempty"`
	AttachmentExtensions        []string `json:"attachment_extension,omitempty"`
	ExcludeAttachmentExtensions []string `json:"exclude_attachment_extension,omitempty"`
	IncludeNSFW                 bool     `json:"include_nsfw,omitempty"`
	Pinned                      *bool    `json:"pinned,omitempty"`
	MinID                       string   `json:"min_id,omitempty"`
	MaxID                       string   `json:"max_id,omitempty"`
	SortBy                      string   `json:"sort_by,omitempty"`
	SortOrder                   string   `json:"sort_order,omitempty"`
	Scope                       string   `json:"scope,omitempty"`
	ContextGuildID              string   `json:"context_guild_id,omitempty"`
	ContextChannelID            string   `json:"context_channel_id,omitempty"`
	HitsPerPage                 int      `json:"hits_per_page,omitempty"`
	Page                        int      `json:"page,omitempty"`
	Cursor                      []string `json:"cursor,omitempty"`
}

// MessageSearchResult is one page of SearchMessages hits plus the channels and threads they
// were found in, with the current user's membership in those threads. Cursor is set when
// another page can be requested by cursor.
type MessageSearchResult struct {
	Messages    []Message      `json:"messages"`
	Channels    []Channel      `json:"channels"`
	Threads     []Channel      `json:"threads,omitempty"`
	Members     []ThreadMember `json:"members,omitempty"`
	Total       int            `json:"total"`
	HitsPerPage int            `json:"hits_per_page"`
	Page        int            `json:"page"`
	Cursor      []string       `json:"cursor,omitempty"`
}

// SearchMessages runs a full-text message search (POST /search/messages). Because it is
// a POST without a nonce, it is only retried on rate limits.
func (c *Client) SearchMessages(ctx context.Context, q MessageSearch) (*MessageSearchResult, error) {
	var r MessageSearchResult
	return &r, c.Do(ctx, http.MethodPost, "/search/messages", q, &r)
}

// MessagesBefore iterates over a channel's messages older than the message ID before,
// fetching 100 at a time. An empty before starts with the newest message. Iteration
// ends after a short page or after yielding the first error.
func (c *Client) MessagesBefore(ctx context.Context, channelID, before string) iter.Seq2[Message, error] {
	return func(yield func(Message, error) bool) {
		for {
			page, err := c.Messages(ctx, channelID, MessageQuery{Before: before, Limit: 100})
			if err != nil {
				yield(Message{}, err)
				return
			}
			for _, m := range page {
				if !yield(m, nil) {
					return
				}
			}
			if len(page) < 100 {
				return
			}
			before = page[len(page)-1].ID
		}
	}
}

// AllMembers iterates over every member of a guild, fetching 1000 at a time. Iteration
// ends after a short page or after yielding the first error.
func (c *Client) AllMembers(ctx context.Context, guildID string) iter.Seq2[Member, error] {
	return func(yield func(Member, error) bool) {
		after := ""
		for {
			page, err := c.Members(ctx, guildID, after, maxMembersPage)
			if err != nil {
				yield(Member{}, err)
				return
			}
			for _, m := range page {
				if !yield(m, nil) {
					return
				}
			}
			if len(page) < maxMembersPage || page[len(page)-1].User == nil {
				return
			}
			after = page[len(page)-1].User.ID
		}
	}
}
