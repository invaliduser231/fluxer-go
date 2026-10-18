package fluxer_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/invaliduser231/fluxer-go"
)

func TestRoleHoistPositions(t *testing.T) {
	srv, c := setup(t)
	noContent := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }
	srv.Handle("PATCH /v1/guilds/{guild}/roles/hoist-positions", noContent)
	srv.Handle("DELETE /v1/guilds/{guild}/roles/hoist-positions", noContent)
	ctx := context.Background()
	if err := c.ModifyRoleHoistPositions(ctx, "3", []fluxer.RoleHoistPosition{{ID: "40", HoistPosition: 2}}); err != nil {
		t.Fatal(err)
	}
	if err := c.ResetRoleHoistPositions(ctx, "3"); err != nil {
		t.Fatal(err)
	}
	reqs := srv.Requests()
	if len(reqs) != 2 || strings.TrimSpace(reqs[0].Body) != `[{"id":"40","hoist_position":2}]` || reqs[1].Method != http.MethodDelete {
		t.Errorf("got %+v", reqs)
	}
}
