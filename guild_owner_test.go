package fluxer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestGuildOwnership(t *testing.T) {
	srv, c := guildSetup(t)
	srv.Handle("POST /v1/guilds/{guild}/transfer-ownership", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			NewOwnerID string `json:"new_owner_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(fluxer.Guild{ID: r.PathValue("guild"), OwnerID: body.NewOwnerID})
	})
	srv.Handle("POST /v1/guilds/{guild}/delete", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	ctx := context.Background()
	g, err := c.TransferOwnership(ctx, "3", "7")
	if err != nil || g.OwnerID != "7" {
		t.Fatalf("got %+v, %v", g, err)
	}
	if err := c.DeleteGuild(ctx, "3"); err != nil {
		t.Fatal(err)
	}
	if last := srv.Requests()[len(srv.Requests())-1]; last.Path != "/v1/guilds/3/delete" || last.Body != "{}" {
		t.Errorf("got %+v", last)
	}
}
