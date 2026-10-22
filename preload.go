package fluxer

import (
	"context"
	"net/http"
)

func (c *Client) LatestMessages(ctx context.Context, channelIDs []string) (map[string]*Message, error) {
	var out map[string]*Message
	err := c.Do(ctx, http.MethodPost, "/users/@me/preload-messages", map[string][]string{"channels": channelIDs}, &out)
	return out, err
}
