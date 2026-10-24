package fluxer

import (
	"context"
	"net/http"
)

// LatestMessages fetches the most recent message of each given channel in one request
// (POST /users/@me/preload-messages). The map is keyed by channel ID; a channel without
// messages may map to nil.
func (c *Client) LatestMessages(ctx context.Context, channelIDs []string) (map[string]*Message, error) {
	var out map[string]*Message
	err := c.Do(ctx, http.MethodPost, "/users/@me/preload-messages", map[string][]string{"channels": channelIDs}, &out)
	return out, err
}
