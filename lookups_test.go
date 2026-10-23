package fluxer_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/fluxertest"
)

func TestLookups(t *testing.T) {
	srv, c := setup(t)
	srv.Handle("GET /v1/users/{user}/profile", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"user":{"id":"` + r.PathValue("user") + `","username":"anna"},"user_profile":{"bio":"hello"},
			"guild_member":{"user":{"id":"7"},"roles":["40"]},"timezone_offset":120}`))
	})
	srv.Handle("GET /v1/users/check-tag", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"taken":` + strconv.FormatBool(r.URL.Query().Get("discriminator") == "0001") + `}`))
	})
	srv.Handle("GET /v1/oauth2/applications/{id}/public", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"900","name":"Relay","redirect_uris":[],"scopes":["bot"],"bot_public":true,"bot":{"id":"1000","username":"testbot"}}`))
	})
	ctx := context.Background()
	p, err := c.UserProfile(ctx, "7", "3")
	if err != nil || p.User.Username != "anna" || p.Bio() != "hello" || p.Member.Roles[0] != "40" || *p.TimezoneOffset != 120 {
		t.Fatalf("got %+v, %v", p, err)
	}
	if path := srv.Requests()[0].Path; path != "/v1/users/7/profile" {
		t.Errorf("path %s", path)
	}
	if taken, err := c.TagTaken(ctx, "testbot", "0001"); err != nil || !taken {
		t.Errorf("got %v, %v", taken, err)
	}
	if taken, _ := c.TagTaken(ctx, "testbot", "0002"); taken {
		t.Error("free tag reported taken")
	}
	app, err := c.PublicApplication(ctx, "900")
	if err != nil || !app.BotPublic || app.Bot.ID != "1000" {
		t.Errorf("got %+v, %v", app, err)
	}
}

func TestIntrospect(t *testing.T) {
	srv, _ := setup(t)
	srv.HandleRaw("POST /v1/oauth2/introspect", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		active := r.PostForm.Get("token") == "good" && r.PostForm.Get("client_id") == fluxertest.OAuthClientID
		if !active {
			_, _ = w.Write([]byte(`{"active":false}`))
			return
		}
		_, _ = w.Write([]byte(`{"active":true,"scope":"identify","sub":"1","exp":1900000000}`))
	})
	o := &fluxer.OAuth{API: srv.API(), ClientID: fluxertest.OAuthClientID, ClientSecret: fluxertest.OAuthSecret}
	info, err := o.Introspect(context.Background(), "good")
	if err != nil || !info.Active || info.Subject != "1" || info.Expires != 1900000000 {
		t.Errorf("got %+v, %v", info, err)
	}
	if info, _ := o.Introspect(context.Background(), "bad"); info.Active {
		t.Error("bad token active")
	}
}
