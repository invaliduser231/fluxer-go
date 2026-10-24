---
title: OAuth2
description: Logging users in with Fluxer and inviting bots.
---

OAuth2 lets a website identify a Fluxer user. A dashboard for your bot is the common case.

```go
o := &fluxer.OAuth{
	ClientID:     os.Getenv("FLUXER_CLIENT_ID"),
	ClientSecret: os.Getenv("FLUXER_CLIENT_SECRET"),
	RedirectURI:  "https://bot.example.com/callback",
	Scopes:       []string{fluxer.ScopeIdentify, fluxer.ScopeGuilds},
}
```

Send the browser to the authorize URL. The state protects against forged callbacks, and the verifier is PKCE; keep both in the user's session:

```go
state := randomString()
verifier := fluxer.NewVerifier()
saveInSession(w, state, verifier)
http.Redirect(w, r, o.AuthorizeURL(state, verifier), http.StatusFound)
```

In the callback, check the state and exchange the code:

```go
if r.URL.Query().Get("state") != savedState {
	http.Error(w, "bad state", http.StatusBadRequest)
	return
}
token, err := o.Exchange(r.Context(), r.URL.Query().Get("code"), savedVerifier)
user, err := o.User(r.Context(), token.AccessToken)
```

The redirect URI must match the one registered for the application exactly, trailing slash included.

Access tokens expire after `token.ExpiresIn` seconds. `o.Refresh` trades the refresh token for a new pair, `o.Revoke` ends a token, and `o.Introspect` reports whether one is still active.

## Inviting a bot

```go
link := fluxer.BotInviteURL("", clientID, fluxer.PermSendMessages|fluxer.PermManageMessages, "")
```

The last argument preselects a guild.
