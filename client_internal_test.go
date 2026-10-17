package fluxer

import "testing"

func TestRouteKey(t *testing.T) {
	tests := map[string]string{
		"/channels/1/messages/2":                 "POST /channels/1/messages/:id",
		"/channels/1/messages/2/reactions/x/@me": "POST /channels/1/messages/:id/reactions/:emoji/@me",
		"/webhooks/5/secret-token":               "POST /webhooks/5/:token",
		"/guilds/3/members/4/roles/5":            "POST /guilds/3/members/:id/roles/:id",
		"/channels/1/messages?limit=5&before=9":  "POST /channels/1/messages",
	}
	for path, want := range tests {
		if got := routeKey("POST", path); got != want {
			t.Errorf("%s: got %s, want %s", path, got, want)
		}
	}
}

func TestMetricRoute(t *testing.T) {
	tests := map[string]string{
		"GET /channels/1/messages/:id":          "GET /channels/:id/messages/:id",
		"POST /webhooks/5/:token":               "POST /webhooks/:id/:?",
		"GET /users/@me":                        "GET /users/@me",
		"DELETE /guilds/3/bans/:id":             "DELETE /guilds/:id/bans/:id",
		"POST /channels/1/messages/bulk-delete": "POST /channels/:id/messages/bulk-delete",
	}
	for route, want := range tests {
		if got := MetricRoute(route); got != want {
			t.Errorf("%s: got %s, want %s", route, got, want)
		}
	}
}
