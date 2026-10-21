package fluxer_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

func TestCompressedGateway(t *testing.T) {
	srv, _ := setup(t)
	events := make(chan event, 100)
	s := &fluxer.Session{
		URL: srv.GatewayURL(), Token: srv.Token, Compress: true,
		Handler:      func(name string, data json.RawMessage) { events <- event{name, data} },
		ReadyTimeout: time.Second,
	}
	runSession(t, s)
	expect(t, events, "READY")
	if st := srv.Stats(); st.Compressed != 1 {
		t.Fatalf("got %+v", st)
	}

	big := strings.Repeat("compressible ", 2000)
	for i := range 3 {
		srv.Dispatch("MESSAGE_CREATE", fluxer.Message{ID: string(rune('1' + i)), ChannelID: "2", Content: big})
	}
	for range 3 {
		var m fluxer.Message
		if err := json.Unmarshal(expect(t, events, "MESSAGE_CREATE").data, &m); err != nil || m.Content != big {
			t.Fatalf("got %d bytes, %v", len(m.Content), err)
		}
	}
	waitFor(t, "heartbeats", func() bool { return srv.Stats().Heartbeats >= 2 })

	srv.Drop()
	srv.Dispatch("MESSAGE_CREATE", fluxer.Message{ID: "9", ChannelID: "2", Content: "while away"})
	expect(t, events, "MESSAGE_CREATE")
	expect(t, events, "RESUMED")
	if st := srv.Stats(); st.Compressed != 2 || st.Resumes != 1 {
		t.Errorf("got %+v", st)
	}
}
