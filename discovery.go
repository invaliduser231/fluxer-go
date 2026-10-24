package fluxer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DiscoveryCategory is a category guilds can be listed under in discovery.
type DiscoveryCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// DiscoveryGuild is a guild as it appears in discovery listings. CategoryType is a
// DiscoveryCategory ID.
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

// DiscoveryResult is one page of SearchDiscovery results and the total number of matches.
type DiscoveryResult struct {
	Guilds []DiscoveryGuild `json:"guilds"`
	Total  int              `json:"total"`
}

// DiscoverySearch filters SearchDiscovery. Empty strings and zero Limit and Offset are
// not sent. Category is a pointer so that category 0 can be requested; nil means any.
type DiscoverySearch struct {
	Query    string
	Category *int
	Language string
	Tag      string
	SortBy   string
	Limit    int
	Offset   int
}

// DiscoveryApplication is a guild's request to be listed in discovery. When passed to
// ApplyForDiscovery only Description, CategoryType, PrimaryLanguage and CustomTags are
// used; the rest is filled in by the server.
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

// DiscoveryStatus reports whether a guild is eligible for discovery and its current
// application, which is nil if it never applied.
type DiscoveryStatus struct {
	Application    *DiscoveryApplication `json:"application"`
	Eligible       bool                  `json:"eligible"`
	MinMemberCount int                   `json:"min_member_count"`
}

// DiscoveryEdit holds the fields to change on a discovery application. Only non-nil
// fields are sent.
type DiscoveryEdit struct {
	Description     *string   `json:"description,omitempty"`
	CategoryType    *int      `json:"category_type,omitempty"`
	PrimaryLanguage *string   `json:"primary_language,omitempty"`
	CustomTags      *[]string `json:"custom_tags,omitempty"`
}

// DiscoveryCategories lists the available discovery categories.
func (c *Client) DiscoveryCategories(ctx context.Context) ([]DiscoveryCategory, error) {
	var cs []DiscoveryCategory
	return cs, c.Do(ctx, http.MethodGet, "/discovery/categories", nil, &cs)
}

// SearchDiscovery searches the public guild directory (GET /discovery/guilds).
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

// DiscoveryStatus fetches the discovery state of a guild.
func (c *Client) DiscoveryStatus(ctx context.Context, guildID string) (*DiscoveryStatus, error) {
	var s DiscoveryStatus
	return &s, c.Do(ctx, http.MethodGet, guildPath(guildID)+"/discovery", nil, &s)
}

// ApplyForDiscovery submits a guild for discovery listing.
func (c *Client) ApplyForDiscovery(ctx context.Context, guildID string, a DiscoveryApplication) (*DiscoveryApplication, error) {
	body := DiscoveryApplication{Description: a.Description, CategoryType: a.CategoryType, PrimaryLanguage: a.PrimaryLanguage, CustomTags: a.CustomTags}
	var out DiscoveryApplication
	return &out, c.Do(ctx, http.MethodPost, guildPath(guildID)+"/discovery", body, &out)
}

// EditDiscoveryApplication applies e to a guild's discovery application.
func (c *Client) EditDiscoveryApplication(ctx context.Context, guildID string, e DiscoveryEdit) (*DiscoveryApplication, error) {
	var out DiscoveryApplication
	return &out, c.Do(ctx, http.MethodPatch, guildPath(guildID)+"/discovery", e, &out)
}

// WithdrawDiscovery withdraws a guild's discovery application or removes its listing.
func (c *Client) WithdrawDiscovery(ctx context.Context, guildID string) error {
	return c.Do(ctx, http.MethodDelete, guildPath(guildID)+"/discovery", nil, nil)
}
