package fluxer

import "time"

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
}

type EmbedAuthor struct {
	Name         string `json:"name"`
	URL          string `json:"url,omitempty"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

type EmbedMedia struct {
	URL         string `json:"url"`
	ProxyURL    string `json:"proxy_url,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Description string `json:"description,omitempty"`
}

type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url,omitempty"`
	ProxyIconURL string `json:"proxy_icon_url,omitempty"`
}

type EmbedProvider struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

func (e *Embed) SetColor(rgb int) *Embed {
	e.Color = &rgb
	return e
}

func (e *Embed) SetTimestamp(t time.Time) *Embed {
	e.Timestamp = t.UTC().Format(time.RFC3339)
	return e
}

func (e *Embed) AddField(name, value string, inline bool) *Embed {
	e.Fields = append(e.Fields, EmbedField{Name: name, Value: value, Inline: inline})
	return e
}
