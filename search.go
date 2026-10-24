package fluxer

import (
	"context"
	"iter"
	"net/http"
)

type MessageSearch struct {
	Content          string   `json:"content,omitempty"`
	ExactPhrases     []string `json:"exact_phrases,omitempty"`
	ChannelIDs       []string `json:"channel_id,omitempty"`
	AuthorIDs        []string `json:"author_id,omitempty"`
	Mentions         []string `json:"mentions,omitempty"`
	Has              []string `json:"has,omitempty"`
	Pinned           *bool    `json:"pinned,omitempty"`
	MinID            string   `json:"min_id,omitempty"`
	MaxID            string   `json:"max_id,omitempty"`
	SortBy           string   `json:"sort_by,omitempty"`
	SortOrder        string   `json:"sort_order,omitempty"`
	Scope            string   `json:"scope,omitempty"`
	ContextGuildID   string   `json:"context_guild_id,omitempty"`
	ContextChannelID string   `json:"context_channel_id,omitempty"`
	HitsPerPage      int      `json:"hits_per_page,omitempty"`
	Page             int      `json:"page,omitempty"`
	Cursor           []string `json:"cursor,omitempty"`
}

type MessageSearchResult struct {
	Messages    []Message `json:"messages"`
	Channels    []Channel `json:"channels"`
	Total       int       `json:"total"`
	HitsPerPage int       `json:"hits_per_page"`
	Page        int       `json:"page"`
	Cursor      []string  `json:"cursor,omitempty"`
}

func (c *Client) SearchMessages(ctx context.Context, q MessageSearch) (*MessageSearchResult, error) {
	var r MessageSearchResult
	return &r, c.Do(ctx, http.MethodPost, "/search/messages", q, &r)
}

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
