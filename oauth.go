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

const (
	ScopeIdentify    = "identify"
	ScopeEmail       = "email"
	ScopeGuilds      = "guilds"
	ScopeConnections = "connections"
	ScopeBot         = "bot"
)

type OAuth struct {
	API          string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	HTTP         *http.Client
}

type OAuthToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func (t OAuthToken) Expiry(issued time.Time) time.Time {
	return issued.Add(time.Duration(t.ExpiresIn) * time.Second)
}

type OAuthError struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

func (e *OAuthError) Error() string { return "fluxer: oauth2 " + e.Code + ": " + e.Description }

func NewVerifier() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (o *OAuth) base() string { return strings.TrimSuffix(cmp.Or(o.API, DefaultAPI), "/") }

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
