package fluxer

import "time"

// Embed is rich content attached to a message. Timestamp is an RFC 3339 string; use
// SetTimestamp to fill it from a time.Time. Color is nil when no color is set. HTML, NSFW
// and Children are only set on embeds Fluxer generates from links: HTML is sanitized
// oEmbed markup and Children holds nested embeds of the same link.
type Embed struct {
	Type        string         `json:"type,omitempty"`
	Title       string         `json:"title,omitempty"`
	URL         string         `json:"url,omitempty"`
	Description string         `json:"description,omitempty"`
	Color       *int           `json:"color,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
	Author      *EmbedAuthor   `json:"author,omitempty"`
	Image       *EmbedMedia    `json:"image,omitempty"`
	Thumbnail   *EmbedMedia    `json:"thumbnail,omitempty"`
	Video       *EmbedMedia    `json:"video,omitempty"`
	Audio       *EmbedMedia    `json:"audio,omitempty"`
	Footer      *EmbedFooter   `json:"footer,omitempty"`
	Provider    *EmbedProvider `json:"provider,omitempty"`
	Fields      []EmbedField   `json:"fields,omitempty"`
	HTML        string         `json:"html,omitempty"`
	HTMLWidth   int            `json:"html_width,omitempty"`
	HTMLHeight  int            `json:"html_height,omitempty"`
	NSFW        bool           `json:"nsfw,omitempty"`
	Children    []Embed        `json:"children,omitempty"`
}

// EmbedAuthor is the author line of an embed.
type EmbedAuthor struct {
	Name         string `json:"name"`
	URL          string `json:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

// EmbedMedia is an image, thumbnail, video or audio item in an embed. ProxyURL, the
// dimensions, ContentHash, Placeholder, Duration in seconds and Flags, which uses the
// AttachmentExplicitMedia and AttachmentAnimated bits, are filled in by the server.
type EmbedMedia struct {
	URL         string `json:"url"`
	ProxyURL    string `json:"proxy_url,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	ContentHash string `json:"content_hash,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Description string `json:"description,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	Flags       int    `json:"flags,omitempty"`
}

// EmbedFooter is the footer of an embed.
type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

// EmbedProvider names the site an embed was generated from.
type EmbedProvider struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// EmbedField is a name and value pair shown in an embed. Inline fields may be placed
// side by side.
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// SetColor sets the embed color to rgb, given as 0xRRGGBB, and returns e.
func (e *Embed) SetColor(rgb int) *Embed {
	e.Color = &rgb
	return e
}

// SetTimestamp sets the embed timestamp to t in UTC and returns e.
func (e *Embed) SetTimestamp(t time.Time) *Embed {
	e.Timestamp = t.UTC().Format(time.RFC3339)
	return e
}

// AddField appends a field to the embed and returns e.
func (e *Embed) AddField(name, value string, inline bool) *Embed {
	e.Fields = append(e.Fields, EmbedField{Name: name, Value: value, Inline: inline})
	return e
}
