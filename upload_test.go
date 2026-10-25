package fluxer_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestUploadAndSend(t *testing.T) {
	srv, c := setup(t)
	var mu sync.Mutex
	stored := map[string][]byte{}
	var auth []string
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		mu.Lock()
		stored[r.URL.Path] = append(stored[r.URL.Path], data...)
		auth = append(auth, r.Header.Get("Authorization"))
		mu.Unlock()
	}))
	t.Cleanup(storage.Close)

	const partSize = 4
	var completed []string
	srv.Handle("POST /v1/channels/{channel}/attachments", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Attachments []struct {
				ID          int    `json:"id"`
				Filename    string `json:"filename"`
				FileSize    int64  `json:"file_size"`
				ContentType string `json:"content_type"`
			} `json:"attachments"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var plan []fluxer.UploadPlan
		for _, a := range req.Attachments {
			p := fluxer.UploadPlan{ID: a.ID, Filename: a.Filename, FileSize: a.FileSize, ContentType: a.ContentType, UploadFilename: "up-" + a.Filename}
			if a.FileSize <= partSize {
				p.UploadMode, p.UploadURL = "singlepart", storage.URL+"/"+a.Filename
			} else {
				p.UploadMode, p.UploadID, p.PartSize = "multipart", "mp-1", partSize
				for n := 1; int64(n-1)*partSize < a.FileSize; n++ {
					p.Parts = append(p.Parts, fluxer.UploadPart{PartNumber: n, UploadURL: storage.URL + "/" + a.Filename + "?part=" + strconv.Itoa(n)})
				}
			}
			plan = append(plan, p)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"attachments": plan})
	})
	srv.Handle("POST /v1/channels/{channel}/attachments/complete", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		completed = append(completed, string(data))
		_, _ = w.Write([]byte(`{"uploads":[]}`))
	})

	msg, err := c.SendMessageUploads(context.Background(), "20", fluxer.MessageSend{Content: "files"}, []fluxer.File{
		{Name: "a.txt", Data: []byte("tiny")},
		{Name: "video.mp4", Data: []byte("0123456789"), Description: "clip", Title: "Launch", Flags: fluxer.AttachmentSpoiler},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored["/a.txt"], []byte("tiny")) || !bytes.Equal(stored["/video.mp4"], []byte("0123456789")) {
		t.Errorf("stored %q", stored)
	}
	for _, a := range auth {
		if a != "" {
			t.Error("sent the token to the storage")
		}
	}
	if len(completed) != 1 || !strings.Contains(completed[0], `"upload_filename":"up-video.mp4","upload_id":"mp-1"`) {
		t.Errorf("completed %q", completed)
	}
	reqs := srv.Requests()
	send := reqs[len(reqs)-1].Body
	if msg.Content != "files" || !strings.Contains(send, `"upload_filename":"up-a.txt"`) || !strings.Contains(send, `"title":"Launch","description":"clip","flags":8`) {
		t.Errorf("send body %s", send)
	}
	if _, err := c.Upload(context.Background(), "20", make([]fluxer.File, 11)); err == nil {
		t.Error("expected error for eleven files")
	}
}

func TestRefreshAttachmentURLs(t *testing.T) {
	srv, c := setup(t)
	srv.Handle("POST /v1/attachments/refresh-urls", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URLs []string `json:"attachment_urls"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var out []map[string]string
		for _, u := range req.URLs {
			out = append(out, map[string]string{"original": u, "refreshed": u + "?fresh"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"refreshed_urls": out})
	})
	got, err := c.RefreshAttachmentURLs(context.Background(), []string{"https://media/x.png"})
	if err != nil || got["https://media/x.png"] != "https://media/x.png?fresh" {
		t.Errorf("got %v, %v", got, err)
	}
}

func TestUploadPlanMissingFiles(t *testing.T) {
	srv, c := setup(t)
	srv.Handle("POST /v1/channels/{channel}/attachments", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"attachments":[]}`))
	})
	if _, err := c.Upload(context.Background(), "20", []fluxer.File{{Name: "a.txt", Data: []byte("x")}}); err == nil || !strings.Contains(err.Error(), "covers 0 of 1") {
		t.Errorf("got %v", err)
	}
}
