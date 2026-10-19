package fluxer_test

import (
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestRouterDispatchesTypedEvents(t *testing.T) {
	r := fluxer.NewRouter()
	var got []string
	fluxer.On(r, func(m *fluxer.MessageCreate) { got = append(got, "a:"+m.Content) })
	remove := fluxer.On(r, func(m *fluxer.MessageCreate) { got = append(got, "b:"+m.Content) })
	fluxer.On(r, func(d *fluxer.MessageDelete) { got = append(got, "del:"+d.ID) })

	r.Handle("MESSAGE_CREATE", json.RawMessage(`{"id":"1","channel_id":"2","content":"hi","author":{"id":"3"}}`))
	remove()
	r.Handle("MESSAGE_CREATE", json.RawMessage(`{"id":"4","channel_id":"2","content":"again","author":{"id":"3"}}`))
	r.Handle("MESSAGE_DELETE", json.RawMessage(`{"id":"4","channel_id":"2"}`))
	r.Handle("TYPING_START", json.RawMessage(`{}`))

	want := []string{"a:hi", "b:hi", "a:again", "del:4"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v", got)
	}
}

func TestRouterOnAny(t *testing.T) {
	r := fluxer.NewRouter()
	var n atomic.Int32
	remove := r.OnAny(func(event string, data json.RawMessage) { n.Add(1) })
	r.Handle("SOMETHING_NEW", nil)
	r.Handle("READY", json.RawMessage(`{"session_id":"s"}`))
	remove()
	r.Handle("READY", json.RawMessage(`{"session_id":"s"}`))
	if n.Load() != 2 {
		t.Errorf("got %d calls", n.Load())
	}
}

func TestRouterReportsErrors(t *testing.T) {
	r := fluxer.NewRouter()
	var errs []string
	r.OnError = func(event string, err error) { errs = append(errs, event+": "+err.Error()) }
	fluxer.On(r, func(*fluxer.MessageDelete) { panic("boom") })
	fluxer.On(r, func(*fluxer.GuildDelete) {})
	r.Handle("MESSAGE_DELETE", json.RawMessage(`{"id":"1"}`))
	r.Handle("GUILD_DELETE", json.RawMessage(`{"id":1}`))
	if len(errs) != 2 || !strings.Contains(errs[0], "panicked: boom") || !strings.Contains(errs[1], "decoding GUILD_DELETE") {
		t.Errorf("got %q", errs)
	}
}

func TestRouterWithSession(t *testing.T) {
	srv, _ := setup(t)
	r := fluxer.NewRouter()
	ready := make(chan *fluxer.Ready, 1)
	msgs := make(chan *fluxer.MessageCreate, 1)
	fluxer.On(r, func(e *fluxer.Ready) { ready <- e })
	fluxer.On(r, func(e *fluxer.MessageCreate) { msgs <- e })
	runSession(t, &fluxer.Session{URL: srv.GatewayURL(), Token: srv.Token, Handler: r.Handle})

	if e := <-ready; e.SessionID == "" || e.User.ID != "1000" {
		t.Errorf("got %+v", e)
	}
	srv.Dispatch("MESSAGE_CREATE", fluxer.Message{ID: "1", ChannelID: "2", Content: "!ping"})
	if m := <-msgs; m.Content != "!ping" {
		t.Errorf("got %+v", m)
	}
}
