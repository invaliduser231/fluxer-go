package fluxer_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

func TestVoiceStateCommandJSON(t *testing.T) {
	data, _ := json.Marshal(fluxer.VoiceStateCommand{GuildID: "3", ConnectionID: "c1"})
	want := `{"channel_id":null,"connection_id":"c1","guild_id":"3","self_deaf":false,"self_mute":false,"self_stream":false,"self_video":false}`
	if string(data) != want {
		t.Errorf("got %s", data)
	}
}

func TestVoiceJoinAndLeave(t *testing.T) {
	srv, _ := guildSetup(t)
	srv.SetVoiceEndpoint("wss://livekit.example")
	r := fluxer.NewRouter()
	ready := make(chan struct{}, 1)
	states := make(chan *fluxer.VoiceStateUpdate, 4)
	servers := make(chan *fluxer.VoiceServerUpdate, 4)
	refused := make(chan *fluxer.GatewayError, 1)
	fluxer.On(r, func(*fluxer.Ready) { ready <- struct{}{} })
	fluxer.On(r, func(e *fluxer.VoiceStateUpdate) { states <- e })
	fluxer.On(r, func(e *fluxer.VoiceServerUpdate) { servers <- e })
	fluxer.On(r, func(e *fluxer.GatewayError) { refused <- e })
	s := &fluxer.Session{URL: srv.GatewayURL(), Token: srv.Token, Handler: r.Handle}
	runSession(t, s)
	<-ready

	ctx := context.Background()
	if err := s.UpdateVoiceState(ctx, fluxer.VoiceStateCommand{GuildID: "3", ChannelID: "31", SelfDeaf: true}); err != nil {
		t.Fatal(err)
	}
	var grant *fluxer.VoiceServerUpdate
	select {
	case st := <-states:
		if st.ChannelID != "31" || !st.SelfDeaf || st.ConnectionID == "" {
			t.Errorf("state %+v", st)
		}
		grant = <-servers
	case <-time.After(3 * time.Second):
		t.Fatal("no voice state")
	}
	if grant.Endpoint != "wss://livekit.example" || grant.Token == "" || grant.GuildID != "3" {
		t.Errorf("grant %+v", grant)
	}

	if err := s.UpdateVoiceState(ctx, fluxer.VoiceStateCommand{GuildID: "3", ConnectionID: grant.ConnectionID}); err != nil {
		t.Fatal(err)
	}
	if st := <-states; st.ChannelID != "" || len(srv.VoiceStates()) != 0 {
		t.Errorf("left %+v", st)
	}

	srv.RefuseNextVoice(fluxer.VoiceChannelFull)
	if err := s.UpdateVoiceState(ctx, fluxer.VoiceStateCommand{GuildID: "3", ChannelID: "31"}); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-refused:
		var ge *fluxer.GatewayError
		if !errors.As(error(e), &ge) || ge.Code != fluxer.VoiceChannelFull {
			t.Errorf("got %v", e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no gateway error")
	}
}
