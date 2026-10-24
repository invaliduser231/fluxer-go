package state_test

import (
	"encoding/json"
	"testing"

	"github.com/invaliduser231/fluxer-go/state"
)

func TestInstancesKeepIDsApart(t *testing.T) {
	m := state.NewInstances()
	a := m.Handler("https://a.example")
	b := m.Handler("https://b.example")
	a("GUILD_CREATE", json.RawMessage(`{"id":"3","properties":{"id":"3","name":"Guild on A"},"roles":[],"channels":[]}`))
	b("GUILD_CREATE", json.RawMessage(`{"id":"3","properties":{"id":"3","name":"Guild on B"},"roles":[],"channels":[]}`))
	ga, _ := m.For("https://a.example").Guild("3")
	gb, _ := m.For("https://b.example").Guild("3")
	if ga.Name != "Guild on A" || gb.Name != "Guild on B" {
		t.Errorf("got %q and %q", ga.Name, gb.Name)
	}
	if got := m.Origins(); len(got) != 2 || got[0] != "https://a.example" {
		t.Errorf("origins %v", got)
	}
	m.Drop("https://a.example")
	if _, ok := m.For("https://a.example").Guild("3"); ok {
		t.Error("dropped cache kept its data")
	}
}

func TestInstancesApplyOptions(t *testing.T) {
	m := state.NewInstances(state.WithMessages(5, 0))
	m.Options = func(origin string) []state.Option {
		if origin == "https://quiet.example" {
			return []state.Option{state.WithoutFlags(state.FlagMessages)}
		}
		return nil
	}
	msg := json.RawMessage(`{"id":"1","channel_id":"2","content":"x","author":{"id":"7","username":"anna"}}`)
	m.Handler("https://a.example")("MESSAGE_CREATE", msg)
	m.Handler("https://quiet.example")("MESSAGE_CREATE", msg)
	if _, ok := m.For("https://a.example").Message("2", "1"); !ok {
		t.Error("message not cached")
	}
	if _, ok := m.For("https://quiet.example").Message("2", "1"); ok {
		t.Error("per instance option ignored")
	}
}
