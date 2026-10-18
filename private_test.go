package fluxer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestGroupDMs(t *testing.T) {
	srv, c := setup(t)
	srv.Handle("POST /v1/users/@me/channels", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Recipients []string `json:"recipients"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		users := make([]fluxer.User, len(body.Recipients))
		for i, id := range body.Recipients {
			users[i] = fluxer.User{ID: id}
		}
		_ = json.NewEncoder(w).Encode(fluxer.Channel{ID: "70", Type: fluxer.ChannelGroupDM, Recipients: users})
	})
	noContent := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }
	srv.Handle("PUT /v1/channels/{channel}/recipients/{user}", noContent)
	srv.Handle("DELETE /v1/channels/{channel}/recipients/{user}", noContent)
	srv.Handle("PUT /v1/users/@me/channels/{channel}/pin", noContent)
	srv.Handle("DELETE /v1/users/@me/channels/{channel}/pin", noContent)

	ctx := context.Background()
	ch, err := c.CreateGroupDM(ctx, []string{"7", "8"})
	if err != nil || ch.Type != fluxer.ChannelGroupDM || len(ch.Recipients) != 2 || !ch.IsPrivate() {
		t.Fatalf("got %+v, %v", ch, err)
	}
	for _, err := range []error{
		c.AddRecipient(ctx, "70", "9"),
		c.RemoveRecipient(ctx, "70", "9", true),
		c.PinDM(ctx, "70"),
		c.UnpinDM(ctx, "70"),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	var got []string
	for _, r := range srv.Requests() {
		got = append(got, r.Method+" "+r.Path)
	}
	want := "POST /v1/users/@me/channels,PUT /v1/channels/70/recipients/9,DELETE /v1/channels/70/recipients/9," +
		"PUT /v1/users/@me/channels/70/pin,DELETE /v1/users/@me/channels/70/pin"
	if strings.Join(got, ",") != want {
		t.Errorf("got %v", got)
	}
}
