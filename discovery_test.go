package fluxer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestDiscovery(t *testing.T) {
	srv, c := setup(t)
	srv.Handle("GET /v1/discovery/categories", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]fluxer.DiscoveryCategory{{ID: 1, Name: "Gaming"}})
	})
	srv.Handle("GET /v1/discovery/guilds", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		_ = json.NewEncoder(w).Encode(fluxer.DiscoveryResult{Total: 1, Guilds: []fluxer.DiscoveryGuild{{ID: "3", Name: q.Get("query") + " " + q.Get("category")}}})
	})
	srv.Handle("POST /v1/guilds/{guild}/discovery", func(w http.ResponseWriter, r *http.Request) {
		var a fluxer.DiscoveryApplication
		_ = json.NewDecoder(r.Body).Decode(&a)
		a.GuildID, a.Status = r.PathValue("guild"), "pending"
		_ = json.NewEncoder(w).Encode(a)
	})
	srv.Handle("DELETE /v1/guilds/{guild}/discovery", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })

	ctx := context.Background()
	cats, err := c.DiscoveryCategories(ctx)
	if err != nil || len(cats) != 1 || cats[0].Name != "Gaming" {
		t.Fatalf("got %+v, %v", cats, err)
	}
	gaming := 1
	res, err := c.SearchDiscovery(ctx, fluxer.DiscoverySearch{Query: "go", Category: &gaming, Limit: 10})
	if err != nil || res.Total != 1 || res.Guilds[0].Name != "go 1" {
		t.Errorf("got %+v, %v", res, err)
	}
	app, err := c.ApplyForDiscovery(ctx, "3", fluxer.DiscoveryApplication{Description: "A place for Go people", CategoryType: 1, Status: "ignored"})
	if err != nil || app.Status != "pending" || app.GuildID != "3" {
		t.Errorf("got %+v, %v", app, err)
	}
	if body := srv.Requests()[2].Body; strings.Contains(body, "ignored") {
		t.Errorf("sent read-only fields: %s", body)
	}
	if err := c.WithdrawDiscovery(ctx, "3"); err != nil {
		t.Fatal(err)
	}
}
