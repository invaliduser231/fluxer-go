package fluxer_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/fluxertest"
)

type event struct {
	name string
	data json.RawMessage
}

func start(t *testing.T, srv *fluxertest.Server, token string) (*fluxer.Session, chan event, chan error) {
	t.Helper()
	events := make(chan event, 100)
	s := &fluxer.Session{
		URL:           srv.GatewayURL(),
		Token:         token,
		IgnoredEvents: []string{"TYPING_START"},
		Handler:       func(name string, data json.RawMessage) { events <- event{name, data} },
		MinBackoff:    10 * time.Millisecond,
		MaxBackoff:    50 * time.Millisecond,
		ReadyTimeout:  200 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return s, events, done
}

func expect(t *testing.T, events chan event, name string) event {
	t.Helper()
	timeout := time.After(3 * time.Second)
	for {
		select {
		case e := <-events:
			if e.name == name {
				return e
			}
		case <-timeout:
			t.Fatalf("no %s event", name)
		}
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestGatewayIdentifyAndDispatch(t *testing.T) {
	srv, _ := setup(t)
	s, events, _ := start(t, srv, "1000.secret")
	expect(t, events, "READY")

	srv.Dispatch("MESSAGE_CREATE", fluxer.Message{ID: "1", ChannelID: "2", Content: "!ping"})
	var m fluxer.Message
	if err := json.Unmarshal(expect(t, events, "MESSAGE_CREATE").data, &m); err != nil || m.Content != "!ping" {
		t.Errorf("got %+v, %v", m, err)
	}
	waitFor(t, "heartbeats", func() bool { return srv.Stats().Heartbeats >= 2 })
	if s.Latency() <= 0 {
		t.Error("latency not measured")
	}
	if c := s.Checkpoint(); c.SessionID == "" || c.Seq != 2 {
		t.Errorf("got checkpoint %+v", c)
	}
	if !slices.Contains(srv.Stats().Ignored, "TYPING_START") {
		t.Error("ignored events not sent")
	}
}

func TestGatewayResumesAfterDrop(t *testing.T) {
	srv, _ := setup(t)
	_, events, _ := start(t, srv, "1000.secret")
	expect(t, events, "READY")

	srv.Drop()
	srv.Dispatch("MESSAGE_CREATE", fluxer.Message{ID: "1", Content: "while away"})
	expect(t, events, "MESSAGE_CREATE")
	expect(t, events, "RESUMED")
	if st := srv.Stats(); st.Identifies != 1 || st.Resumes != 1 {
		t.Errorf("got %+v", st)
	}
}

func TestGatewayReconnectRequest(t *testing.T) {
	srv, _ := setup(t)
	_, events, _ := start(t, srv, "1000.secret")
	expect(t, events, "READY")

	srv.RequestReconnect()
	expect(t, events, "RESUMED")
	srv.Dispatch("MESSAGE_CREATE", fluxer.Message{ID: "2"})
	expect(t, events, "MESSAGE_CREATE")
}

func TestGatewayHandsOverWithoutOverlap(t *testing.T) {
	srv, _ := setup(t)
	events := make(chan event, 100)
	handler := func(name string, data json.RawMessage) { events <- event{name, data} }
	run := func(s *fluxer.Session) func() {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- s.Run(ctx) }()
		return func() {
			cancel()
			<-done
		}
	}
	old := &fluxer.Session{URL: srv.GatewayURL(), Token: "1000.secret", Handler: handler}
	stop := run(old)
	expect(t, events, "READY")
	stop()
	for deadline := time.Now().Add(5 * time.Second); srv.Connections() > 0 && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}

	next := &fluxer.Session{URL: srv.GatewayURL(), Token: "1000.secret", Handler: handler}
	next.Restore(old.Checkpoint())
	defer run(next)()
	expect(t, events, "RESUMED")
	if st := srv.Stats(); st.Resumes != 1 || st.Overlaps != 0 {
		t.Errorf("got %+v", st)
	}
}

func TestGatewayInvalidSession(t *testing.T) {
	srv, _ := setup(t)
	_, events, _ := start(t, srv, "1000.secret")
	expect(t, events, "READY")

	srv.InvalidateSessions()
	expect(t, events, "READY")
	if st := srv.Stats(); st.Identifies != 2 {
		t.Errorf("got %+v", st)
	}
}

func TestGatewayInvalidToken(t *testing.T) {
	srv, _ := setup(t)
	_, _, done := start(t, srv, "wrong")
	select {
	case err := <-done:
		if !errors.Is(err, fluxer.ErrInvalidToken) {
			t.Errorf("got %v", err)
		}
		done <- err
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not stop on an invalid token")
	}
}

func TestGatewayStopsWhenAddressIsBanned(t *testing.T) {
	srv, _ := setup(t)
	_, events, done := start(t, srv, "1000.secret")
	expect(t, events, "READY")

	until := time.Now().Add(time.Hour).Truncate(time.Second)
	srv.Ban("GLOBAL_IP_TEMPORARILY_BANNED", until)
	srv.Drop()
	select {
	case err := <-done:
		b, ok := errors.AsType[*fluxer.BanError](err)
		if !ok || !b.Until.Equal(until) || b.Permanent {
			t.Errorf("got %v", err)
		}
		done <- err
	case <-time.After(3 * time.Second):
		t.Fatal("Run kept reconnecting while banned")
	}
	if st := srv.Stats(); st.Dials != 2 {
		t.Errorf("%d dials", st.Dials)
	}
}

func TestGatewayURLFromREST(t *testing.T) {
	srv, c := setup(t)
	g, err := c.GatewayBot(context.Background())
	if err != nil || g.URL != srv.GatewayURL() || g.Shards != 1 {
		t.Errorf("got %+v, %v", g, err)
	}
}

func TestGatewayResendsDroppedIdentify(t *testing.T) {
	srv, _ := setup(t)
	srv.DropNextIdentify()
	_, events, _ := start(t, srv, "1000.secret")
	expect(t, events, "READY")

	srv.DropNextIdentify()
	srv.InvalidateSessions()
	expect(t, events, "READY")
	if st := srv.Stats(); st.Identifies != 2 {
		t.Errorf("got %+v", st)
	}
}

func TestGatewayIdentifiesAfterRejectedResume(t *testing.T) {
	srv, _ := setup(t)
	_, events, done := start(t, srv, "1000.secret")
	expect(t, events, "READY")

	srv.RejectNextResume()
	srv.Drop()
	expect(t, events, "READY")
	select {
	case err := <-done:
		t.Fatalf("Run stopped: %v", err)
	default:
	}
}

func TestGatewaySurvivesOverflowingHeartbeatInterval(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer ws.CloseNow()
		ctx := r.Context()
		_ = ws.Write(ctx, websocket.MessageText, []byte(`{"op":10,"d":{"heartbeat_interval":9223372036855}}`))
		if _, _, err := ws.Read(ctx); err != nil {
			return
		}
		_ = ws.Write(ctx, websocket.MessageText, []byte(`{"op":0,"s":1,"t":"READY","d":{"session_id":"s1","version":1,"guilds":[]}}`))
		for {
			if _, _, err := ws.Read(ctx); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	events := make(chan event, 100)
	s := &fluxer.Session{
		URL:     "ws" + strings.TrimPrefix(srv.URL, "http") + "/gateway",
		Token:   "1000.secret",
		Handler: func(name string, data json.RawMessage) { events <- event{name, data} },
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	expect(t, events, "READY")
	select {
	case err := <-done:
		t.Fatalf("Run stopped: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
}

func setup(t *testing.T) (*fluxertest.Server, *fluxer.Client) {
	t.Helper()
	srv := fluxertest.NewServer("1000.secret")
	t.Cleanup(srv.Close)
	return srv, srv.Client()
}
