package fluxer

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// OAuth2 scopes for OAuth.Scopes. ScopeBot is used by BotInviteURL.
const (
	ScopeIdentify    = "identify"
	ScopeEmail       = "email"
	ScopeGuilds      = "guilds"
	ScopeConnections = "connections"
	ScopeBot         = "bot"
)

// OAuth is an OAuth2 client for a Fluxer application. API is the API base URL
// and defaults to DefaultAPI. Scopes defaults to ScopeIdentify. ClientSecret may
// be empty for a public client that uses PKCE. HTTP defaults to a client with a
// 15 second timeout. Requests made through OAuth are not rate limited or
// retried.
type OAuth struct {
	API          string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	HTTP         *http.Client
}

// OAuthToken is the response of the token endpoint. ExpiresIn is in seconds.
type OAuthToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// Expiry returns when the access token expires, given the time it was issued.
func (t OAuthToken) Expiry(issued time.Time) time.Time {
	return issued.Add(time.Duration(t.ExpiresIn) * time.Second)
}

// OAuthError is an error response in the OAuth2 format. Code holds the error
// field, such as invalid_grant.
type OAuthError struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

// Error returns the OAuth2 error code and description.
func (e *OAuthError) Error() string { return "fluxer: oauth2 " + e.Code + ": " + e.Description }

// NewVerifier returns a random PKCE code verifier. Keep it until the code is
// exchanged and pass it to both AuthorizeURL and Exchange.
func NewVerifier() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// Challenge returns the S256 PKCE code challenge for verifier.
func Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (o *OAuth) base() string { return strings.TrimSuffix(cmp.Or(o.API, DefaultAPI), "/") }

// AuthorizeURL returns the authorization URL to send the user to. state is
// added when non-empty. When verifier is non-empty, its S256 challenge is added
// for PKCE.
func (o *OAuth) AuthorizeURL(state, verifier string) string {
	scopes := o.Scopes
	if len(scopes) == 0 {
		scopes = []string{ScopeIdentify}
	}
	q := url.Values{
		"response_type": {"code"},
		"client_id":     {o.ClientID},
		"redirect_uri":  {o.RedirectURI},
		"scope":         {strings.Join(scopes, " ")},
	}
	if state != "" {
		q.Set("state", state)
	}
	if verifier != "" {
		q.Set("code_challenge", Challenge(verifier))
		q.Set("code_challenge_method", "S256")
	}
	return o.base() + "/oauth2/authorize?" + q.Encode()
}

// BotInviteURL returns the URL that adds the bot of application clientID to a
// guild. api is the API base URL and defaults to DefaultAPI. perms is left out
// when zero, and a non-empty guildID preselects the guild.
func BotInviteURL(api, clientID string, perms Permissions, guildID string) string {
	q := url.Values{"client_id": {clientID}, "scope": {ScopeBot}}
	if perms != 0 {
		q.Set("permissions", strconv.FormatUint(uint64(perms), 10))
	}
	if guildID != "" {
		q.Set("guild_id", guildID)
	}
	return strings.TrimSuffix(cmp.Or(api, DefaultAPI), "/") + "/oauth2/authorize?" + q.Encode()
}

// Exchange trades an authorization code for a token at POST /oauth2/token.
// verifier is the PKCE verifier given to AuthorizeURL, or empty if PKCE was not
// used. Errors are *OAuthError when the server answers in the OAuth2 format and
// *APIError otherwise.
func (o *OAuth) Exchange(ctx context.Context, code, verifier string) (*OAuthToken, error) {
	form := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {o.RedirectURI},
	}
	if verifier != "" {
		form.Set("code_verifier", verifier)
	}
	return o.token(ctx, form)
}

// Refresh trades a refresh token for a new token at POST /oauth2/token.
func (o *OAuth) Refresh(ctx context.Context, refreshToken string) (*OAuthToken, error) {
	return o.token(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}})
}

func (o *OAuth) token(ctx context.Context, form url.Values) (*OAuthToken, error) {
	form.Set("client_id", o.ClientID)
	if o.ClientSecret != "" {
		form.Set("client_secret", o.ClientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base()+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var t OAuthToken
	return &t, o.do(req, &t)
}

// Revoke revokes an access or refresh token at POST /oauth2/token/revoke.
func (o *OAuth) Revoke(ctx context.Context, token string) error {
	form := url.Values{"token": {token}, "client_id": {o.ClientID}}
	if o.ClientSecret != "" {
		form.Set("client_secret", o.ClientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base()+"/oauth2/token/revoke", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return o.do(req, nil)
}

// User returns the user an access token belongs to. It calls
// GET /oauth2/userinfo.
func (o *OAuth) User(ctx context.Context, accessToken string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.base()+"/oauth2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	var u User
	return &u, o.do(req, &u)
}

func (o *OAuth) do(req *http.Request, out any) error {
	req.Header.Set("User-Agent", "fluxer-go/"+Version)
	client := o.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		var oe OAuthError
		if json.Unmarshal(data, &oe) == nil && oe.Code != "" {
			return &oe
		}
		return apiError(resp, data)
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}
