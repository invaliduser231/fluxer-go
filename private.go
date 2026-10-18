package fluxer

import (
	"context"
	"net/http"
	"net/url"
)

func (c *Client) CreateGroupDM(ctx context.Context, recipientIDs []string) (*Channel, error) {
	var ch Channel
	return &ch, c.Do(ctx, http.MethodPost, "/users/@me/channels", map[string][]string{"recipients": recipientIDs}, &ch)
}

func (c *Client) AddRecipient(ctx context.Context, channelID, userID string) error {
	return c.Do(ctx, http.MethodPut, channelPath(channelID)+"/recipients/"+userID, nil, nil)
}

func (c *Client) RemoveRecipient(ctx context.Context, channelID, userID string, silent bool) error {
	path := channelPath(channelID) + "/recipients/" + userID
	if silent {
		path += "?" + url.Values{"silent": {"true"}}.Encode()
	}
	return c.Do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) PinDM(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodPut, "/users/@me/channels/"+channelID+"/pin", nil, nil)
}

func (c *Client) UnpinDM(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodDelete, "/users/@me/channels/"+channelID+"/pin", nil, nil)
}
