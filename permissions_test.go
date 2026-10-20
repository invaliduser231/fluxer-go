package fluxer_test

import (
	"encoding/json"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestPermissionsJSON(t *testing.T) {
	var r fluxer.Role
	if err := json.Unmarshal([]byte(`{"id":"1","name":"x","position":0,"permissions":"18014398509481984"}`), &r); err != nil {
		t.Fatal(err)
	}
	if r.Permissions != fluxer.PermViewChannelMembers {
		t.Errorf("got %d", r.Permissions)
	}
	data, _ := json.Marshal(fluxer.NewRole{Name: "mods", Permissions: fluxer.PermKickMembers | fluxer.PermBanMembers})
	if string(data) != `{"name":"mods","permissions":"6"}` {
		t.Errorf("got %s", data)
	}
	var p fluxer.Permissions
	if err := json.Unmarshal([]byte(`1024`), &p); err != nil || p != fluxer.PermViewChannel {
		t.Errorf("got %d, %v", p, err)
	}
	if err := json.Unmarshal([]byte(`"-1"`), &p); err == nil {
		t.Error("expected error")
	}
}

func TestPermissionsHelpers(t *testing.T) {
	p := fluxer.PermSendMessages.Add(fluxer.PermViewChannel)
	if !p.Has(fluxer.PermViewChannel|fluxer.PermSendMessages) || p.Has(fluxer.PermAttachFiles) {
		t.Errorf("got %s", p)
	}
	if missing := p.Missing(fluxer.PermViewChannel | fluxer.PermEmbedLinks); missing != fluxer.PermEmbedLinks {
		t.Errorf("missing %s", missing)
	}
	if s := p.String(); s != "VIEW_CHANNEL|SEND_MESSAGES" {
		t.Errorf("got %s", s)
	}
	if s := p.Remove(fluxer.PermSendMessages).String(); s != "VIEW_CHANNEL" {
		t.Errorf("got %s", s)
	}
}

func TestChannelPermissions(t *testing.T) {
	g := fluxer.Guild{ID: "1", OwnerID: "99", Roles: []fluxer.Role{
		{ID: "1", Permissions: fluxer.PermViewChannel | fluxer.PermSendMessages},
		{ID: "10", Permissions: fluxer.PermManageMessages},
		{ID: "11", Permissions: fluxer.PermAdministrator},
		{ID: "12"},
	}}
	ch := fluxer.Channel{ID: "5", PermissionOverwrites: []fluxer.Overwrite{
		{ID: "10", Type: fluxer.OverwriteRole, Deny: fluxer.PermSendMessages},
		{ID: "1", Type: fluxer.OverwriteRole, Deny: fluxer.PermViewChannel},
		{ID: "12", Type: fluxer.OverwriteRole, Allow: fluxer.PermViewChannel | fluxer.PermSendMessages},
		{ID: "7", Type: fluxer.OverwriteMember, Allow: fluxer.PermAttachFiles},
	}}
	tests := []struct {
		name  string
		user  string
		roles []string
		want  fluxer.Permissions
	}{
		{"everyone denied view", "6", nil, fluxer.PermSendMessages},
		{"role allow beats role deny", "6", []string{"10", "12"}, fluxer.PermViewChannel | fluxer.PermSendMessages | fluxer.PermManageMessages},
		{"member overwrite", "7", []string{"10"}, fluxer.PermManageMessages | fluxer.PermAttachFiles},
		{"administrator", "6", []string{"11"}, fluxer.PermAll},
		{"owner", "99", nil, fluxer.PermAll},
	}
	for _, tt := range tests {
		if got := fluxer.ChannelPermissions(g, tt.user, tt.roles, ch); got != tt.want {
			t.Errorf("%s: got %s, want %s", tt.name, got, tt.want)
		}
	}
	m := fluxer.Member{User: &fluxer.User{ID: "6"}, Roles: []string{"10"}}
	if got := fluxer.MemberPermissions(g, m, nil); got != fluxer.PermViewChannel|fluxer.PermSendMessages|fluxer.PermManageMessages {
		t.Errorf("base got %s", got)
	}
}
