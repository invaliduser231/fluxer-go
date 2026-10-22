package fluxer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestLatestMessages(t *testing.T) {
	srv, c := setup(t)
	srv.AddMessage(fluxer.Message{ID: "10", ChannelID: "1", Content: "old"})
	srv.AddMessage(fluxer.Message{ID: "11", ChannelID: "1", Content: "new"})
	srv.Handle("POST /v1/users/@me/preload-messages", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Channels []string `json:"channels"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		out := map[string]*fluxer.Message{}
		for _, id := range req.Channels {
			msgs := srv.Messages(id)
			if len(msgs) == 0 {
				out[id] = nil
				continue
			}
			out[id] = &msgs[len(msgs)-1]
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	got, err := c.LatestMessages(context.Background(), []string{"1", "2"})
	if err != nil || got["1"].Content != "new" || got["2"] != nil {
		t.Errorf("got %+v, %v", got, err)
	}
}
