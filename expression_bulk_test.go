package fluxer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestBulkEmojis(t *testing.T) {
	srv, c := setup(t)
	srv.Handle("POST /v1/guilds/{guild}/emojis/bulk", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Emojis []fluxer.NewEmoji `json:"emojis"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		res := fluxer.EmojiBulkResult{Success: []fluxer.Emoji{}, Failed: []fluxer.BulkFailure{}}
		for i, e := range body.Emojis {
			if e.Image == "" {
				res.Failed = append(res.Failed, fluxer.BulkFailure{Name: e.Name, Error: "image required"})
				continue
			}
			res.Success = append(res.Success, fluxer.Emoji{ID: string(rune('a' + i)), Name: e.Name})
		}
		_ = json.NewEncoder(w).Encode(res)
	})
	srv.Handle("GET /v1/emojis/{emoji}/metadata", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(fluxer.ExpressionMetadata{ID: r.PathValue("emoji"), GuildID: "3", Name: "coin", AllowCloning: true})
	})
	srv.Handle("GET /v1/stickers/{sticker}/source", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(fluxer.ExpressionSource{ID: "3", Name: "Test", Features: []string{"VERIFIED"}})
	})

	ctx := context.Background()
	img := fluxer.ImageData("image/png", []byte("png"))
	res, err := c.CreateEmojis(ctx, "3", []fluxer.NewEmoji{{Name: "coin", Image: img}, {Name: "broken"}})
	if err != nil || len(res.Success) != 1 || len(res.Failed) != 1 || res.Failed[0].Name != "broken" {
		t.Fatalf("got %+v, %v", res, err)
	}
	meta, err := c.EmojiMetadata(ctx, "60")
	if err != nil || meta.ID != "60" || !meta.AllowCloning {
		t.Errorf("got %+v, %v", meta, err)
	}
	src, err := c.StickerSource(ctx, "61")
	if err != nil || src.Name != "Test" || src.Features[0] != "VERIFIED" {
		t.Errorf("got %+v, %v", src, err)
	}
}
