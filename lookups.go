package fluxer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// UserProfile is a user's profile. Profile holds the raw profile fields, Member
// is set when the profile was requested for a guild, and Limited reports that
// the API returned a limited profile.
type UserProfile struct {
	User           User              `json:"user"`
	Profile        map[string]any    `json:"user_profile"`
	Member         *Member           `json:"guild_member,omitempty"`
	MutualGuilds   []json.RawMessage `json:"mutual_guilds,omitempty"`
	TimezoneOffset *int              `json:"timezone_offset"`
	Limited        bool              `json:"profile_limited,omitempty"`
}

// Bio returns the bio from the profile fields, or an empty string.
func (p UserProfile) Bio() string {
	bio, _ := p.Profile["bio"].(string)
	return bio
}

// PublicApplication is the public information about an application.
type PublicApplication struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Icon         string   `json:"icon,omitempty"`
	Description  string   `json:"description,omitempty"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	BotPublic    bool     `json:"bot_public"`
	Bot          *User    `json:"bot,omitempty"`
}

// TokenInfo is the result of token introspection. Active reports whether the
// token is valid. Expires and IssuedAt are Unix times in seconds.
type TokenInfo struct {
	Active    bool   `json:"active"`
	Scope     string `json:"scope,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	Username  string `json:"username,omitempty"`
	TokenType string `json:"token_type,omitempty"`
	Expires   int64  `json:"exp,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
	Subject   string `json:"sub,omitempty"`
}

// UserProfile returns the profile of userID. With a non-empty guildID the
// profile includes the user's membership in that guild. It calls
// GET /users/{id}/profile.
func (c *Client) UserProfile(ctx context.Context, userID, guildID string) (*UserProfile, error) {
	path := "/users/" + userID + "/profile"
	if guildID != "" {
		path += "?" + url.Values{"guild_id": {guildID}}.Encode()
	}
	var p UserProfile
	return &p, c.Do(ctx, http.MethodGet, path, nil, &p)
}

// TagTaken reports whether the combination of username and discriminator is
// already in use. It calls GET /users/check-tag.
func (c *Client) TagTaken(ctx context.Context, username, discriminator string) (bool, error) {
	q := url.Values{"username": {username}, "discriminator": {discriminator}}
	var r struct {
		Taken bool `json:"taken"`
	}
	err := c.Do(ctx, http.MethodGet, "/users/check-tag?"+q.Encode(), nil, &r)
	return r.Taken, err
}

// OAuthApplication returns the application of the current token. It calls
// GET /oauth2/applications/@me.
func (c *Client) OAuthApplication(ctx context.Context) (*Application, error) {
	var a Application
	return &a, c.Do(ctx, http.MethodGet, "/oauth2/applications/@me", nil, &a)
}

// PublicApplication returns the public information of an application. It
// calls GET /oauth2/applications/{id}/public.
func (c *Client) PublicApplication(ctx context.Context, applicationID string) (*PublicApplication, error) {
	var a PublicApplication
	return &a, c.Do(ctx, http.MethodGet, "/oauth2/applications/"+applicationID+"/public", nil, &a)
}

// Introspect asks the server about token at POST /oauth2/introspect,
// authenticating with the client credentials.
func (o *OAuth) Introspect(ctx context.Context, token string) (*TokenInfo, error) {
	form := url.Values{"token": {token}, "client_id": {o.ClientID}}
	if o.ClientSecret != "" {
		form.Set("client_secret", o.ClientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base()+"/oauth2/introspect", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var info TokenInfo
	return &info, o.do(req, &info)
}
