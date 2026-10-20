package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type DiscoveryCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type DiscoveryGuild struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Icon            string   `json:"icon,omitempty"`
	Banner          string   `json:"banner,omitempty"`
	Description     string   `json:"description,omitempty"`
	CategoryType    int      `json:"category_type"`
	PrimaryLanguage string   `json:"primary_language,omitempty"`
	CustomTags      []string `json:"custom_tags,omitempty"`
	MemberCount     int      `json:"member_count"`
	OnlineCount     int      `json:"online_count"`
	Features        []string `json:"features,omitempty"`
}

type DiscoveryResult struct {
	Guilds []DiscoveryGuild `json:"guilds"`
	Total  int              `json:"total"`
}

type DiscoverySearch struct {
	Query    string
	Category *int
	Language string
	Tag      string
	SortBy   string
	Limit    int
	Offset   int
}

type DiscoveryApplication struct {
	GuildID         string     `json:"guild_id,omitempty"`
	Status          string     `json:"status,omitempty"`
	Description     string     `json:"description"`
	CategoryType    int        `json:"category_type"`
	PrimaryLanguage string     `json:"primary_language,omitempty"`
	CustomTags      []string   `json:"custom_tags,omitempty"`
	AppliedAt       *time.Time `json:"applied_at,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	ReviewReason    string     `json:"review_reason,omitempty"`
	RemovedAt       *time.Time `json:"removed_at,omitempty"`
	RemovalReason   string     `json:"removal_reason,omitempty"`
}

type DiscoveryStatus struct {
	Application    *DiscoveryApplication `json:"application"`
	Eligible       bool                  `json:"eligible"`
	MinMemberCount int                   `json:"min_member_count"`
}

type DiscoveryEdit struct {
	Description     *string   `json:"description,omitempty"`
	CategoryType    *int      `json:"category_type,omitempty"`
	PrimaryLanguage *string   `json:"primary_language,omitempty"`
	CustomTags      *[]string `json:"custom_tags,omitempty"`
}

func (c *Client) DiscoveryCategories(ctx context.Context) ([]DiscoveryCategory, error) {
	var cs []DiscoveryCategory
	return cs, c.Do(ctx, http.MethodGet, "/discovery/categories", nil, &cs)
}

func (c *Client) SearchDiscovery(ctx context.Context, q DiscoverySearch) (*DiscoveryResult, error) {
	v := url.Values{}
	for key, val := range map[string]string{"query": q.Query, "language": q.Language, "tag": q.Tag, "sort_by": q.SortBy} {
		if val != "" {
			v.Set(key, val)
		}
	}
	if q.Category != nil {
		v.Set("category", strconv.Itoa(*q.Category))
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Offset > 0 {
		v.Set("offset", strconv.Itoa(q.Offset))
	}
	path := "/discovery/guilds"
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	var r DiscoveryResult
	return &r, c.Do(ctx, http.MethodGet, path, nil, &r)
}

func (c *Client) DiscoveryStatus(ctx context.Context, guildID string) (*DiscoveryStatus, error) {
	var s DiscoveryStatus
	return &s, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/discovery", nil, &s)
}

func (c *Client) ApplyForDiscovery(ctx context.Context, guildID string, a DiscoveryApplication) (*DiscoveryApplication, error) {
	body := DiscoveryApplication{Description: a.Description, CategoryType: a.CategoryType, PrimaryLanguage: a.PrimaryLanguage, CustomTags: a.CustomTags}
	var out DiscoveryApplication
	return &out, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/discovery", body, &out)
}

func (c *Client) EditDiscoveryApplication(ctx context.Context, guildID string, e DiscoveryEdit) (*DiscoveryApplication, error) {
	var out DiscoveryApplication
	return &out, c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/discovery", e, &out)
}

func (c *Client) WithdrawDiscovery(ctx context.Context, guildID string) error {
	return c.Do(ctx, http.MethodDelete, guildPath(guildID)+"/discovery", nil, nil)
}
