package fluxer_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/fluxertest"
)

func TestOAuthFlow(t *testing.T) {
	srv, _ := setup(t)
	o := &fluxer.OAuth{
		API: srv.API(), ClientID: fluxertest.OAuthClientID, ClientSecret: fluxertest.OAuthSecret,
		RedirectURI: "https://app.example/callback", Scopes: []string{fluxer.ScopeIdentify, fluxer.ScopeGuilds},
	}
	verifier := fluxer.NewVerifier()
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := noRedirect.Get(o.AuthorizeURL("st4te", verifier))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	back, err := url.Parse(resp.Header.Get("Location"))
	if err != nil || back.Query().Get("state") != "st4te" {
		t.Fatalf("redirect %v, %v", back, err)
	}
	ctx := context.Background()
	tok, err := o.Exchange(ctx, back.Query().Get("code"), verifier)
	if err != nil || tok.Scope != "identify guilds" {
		t.Fatalf("got %+v, %v", tok, err)
	}
	u, err := o.User(ctx, tok.AccessToken)
	if err != nil || u.Username != "owner" {
		t.Errorf("got %+v, %v", u, err)
	}
	next, err := o.Refresh(ctx, tok.RefreshToken)
	if err != nil || next.AccessToken == tok.AccessToken {
		t.Errorf("got %+v, %v", next, err)
	}
	if err := o.Revoke(ctx, next.AccessToken); err != nil {
		t.Fatal(err)
	}
	if _, err := o.User(ctx, next.AccessToken); fluxer.ErrorCode(err) != "UNAUTHORIZED" {
		t.Errorf("got %v", err)
	}
	_, err = o.Exchange(ctx, "code-unknown", verifier)
	var oe *fluxer.OAuthError
	if !errors.As(err, &oe) || oe.Code != "invalid_grant" {
		t.Errorf("got %v", err)
	}
}

func TestBotInviteURL(t *testing.T) {
	got := fluxer.BotInviteURL("", "900", fluxer.PermSendMessages|fluxer.PermViewChannel, "3")
	want := "https://api.fluxer.app/v1/oauth2/authorize?client_id=900&guild_id=3&permissions=3072&scope=bot"
	if got != want {
		t.Errorf("got %s", got)
	}
}
