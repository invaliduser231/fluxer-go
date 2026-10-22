package voice_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/invaliduser231/fluxer-go"
	"github.com/invaliduser231/fluxer-go/fluxertest"
	"github.com/invaliduser231/fluxer-go/voice"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

type fakeRoom struct {
	mu           sync.Mutex
	endpoint     string
	token        string
	published    []string
	unpublished  int
	disconnected bool
}

func (r *fakeRoom) Publish(track *lksdk.LocalTrack, opts lksdk.TrackPublicationOptions) (func() error, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.published = append(r.published, opts.Name)
	return func() error {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.unpublished++
		return nil
	}, nil
}

func (r *fakeRoom) Disconnect() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.disconnected = true
}

type rooms struct {
	mu  sync.Mutex
	all []*fakeRoom
}

func (rs *rooms) connect(ctx context.Context, endpoint, token string, cb *lksdk.RoomCallback, receive bool) (voice.Room, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	r := &fakeRoom{endpoint: endpoint, token: token}
	rs.all = append(rs.all, r)
	return r, nil
}

func (rs *rooms) last() *fakeRoom {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.all[len(rs.all)-1]
}

func setup(t *testing.T) (*fluxertest.Server, *voice.Manager, *rooms) {
	t.Helper()
	srv := fluxertest.NewServer("1000.secret")
	t.Cleanup(srv.Close)
	srv.SetVoiceEndpoint("wss://livekit.example")
	r := fluxer.NewRouter()
	ready := make(chan struct{}, 1)
	fluxer.On(r, func(*fluxer.Ready) { ready <- struct{}{} })
	s := &fluxer.Session{URL: srv.GatewayURL(), Token: srv.Token, Handler: r.Handle, MinBackoff: 10 * time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		t.Fatal("no ready")
	}
	rs := &rooms{}
	m := voice.NewManager(s, r)
	m.Connect = rs.connect
	m.Timeout = 2 * time.Second
	t.Cleanup(m.Close)
	return srv, m, rs
}

func TestJoinMoveLeave(t *testing.T) {
	srv, m, rs := setup(t)
	ctx := context.Background()
	c, err := m.Join(ctx, "3", "31", voice.Options{SelfDeaf: true})
	if err != nil {
		t.Fatal(err)
	}
	room := rs.last()
	if room.endpoint != "wss://livekit.example" || room.token != "voice-token-"+c.ConnectionID() || c.ChannelID() != "31" {
		t.Errorf("room %+v, conn %s %s", room, c.ConnectionID(), c.ChannelID())
	}
	if again, err := m.Join(ctx, "3", "31", voice.Options{}); err != nil || again != c {
		t.Errorf("second join gave %v, %v", again, err)
	}

	if err := c.SetMute(ctx, true); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "mute", func() bool {
		st := srv.VoiceStates()
		return len(st) == 1 && st[0].SelfMute && st[0].SelfDeaf && st[0].ConnectionID == c.ConnectionID()
	})

	if err := c.Move(ctx, "32"); err != nil {
		t.Fatal(err)
	}
	if c.ChannelID() != "32" || len(rs.all) != 2 || !rs.all[0].disconnected {
		t.Errorf("after move: channel %s, %d rooms", c.ChannelID(), len(rs.all))
	}

	if err := c.Leave(ctx); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "leave", func() bool { return len(srv.VoiceStates()) == 0 })
	if !rs.last().disconnected || m.Conn("3") != nil {
		t.Error("connection kept after leaving")
	}
	if err := c.Move(ctx, "31"); !errors.Is(err, voice.ErrClosed) {
		t.Errorf("got %v", err)
	}
}

func TestJoinRefused(t *testing.T) {
	srv, m, _ := setup(t)
	srv.RefuseNextVoice(fluxer.VoiceChannelFull)
	_, err := m.Join(context.Background(), "3", "31", voice.Options{})
	var ge *fluxer.GatewayError
	if !errors.As(err, &ge) || ge.Code != fluxer.VoiceChannelFull {
		t.Errorf("got %v", err)
	}
	if m.Conn("3") != nil {
		t.Error("refused join left a connection")
	}
}

func TestPlayPublishesAndStops(t *testing.T) {
	_, m, rs := setup(t)
	c, err := m.Join(context.Background(), "3", "31", voice.Options{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = c.Play(ctx, io.NopCloser(strings.NewReader("not really ogg")))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("got %v", err)
	}
	room := rs.last()
	if len(room.published) != 1 || room.published[0] != "audio" || room.unpublished != 1 {
		t.Errorf("room %+v", room)
	}
	track, stop, err := c.OpusTrack("music")
	if err != nil || track == nil {
		t.Fatal(err)
	}
	if err := stop(); err != nil || room.unpublished != 2 {
		t.Errorf("got %v, %d", err, room.unpublished)
	}
}

func TestParseIdentity(t *testing.T) {
	user, conn, ok := voice.ParseIdentity("user_1189375284394692700_b7f3c2a94e1d4c0f")
	if !ok || user != "1189375284394692700" || conn != "b7f3c2a94e1d4c0f" {
		t.Errorf("got %s %s %v", user, conn, ok)
	}
	for _, bad := range []string{"", "bot_1_2", "user_1", "user__x"} {
		if _, _, ok := voice.ParseIdentity(bad); ok {
			t.Errorf("%q parsed", bad)
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
