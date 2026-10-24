package fluxer

import (
	"context"
	"net/http"
	"net/url"
)

// CreateGroupDM creates a group DM with the given users.
func (c *Client) CreateGroupDM(ctx context.Context, recipientIDs []string) (*Channel, error) {
	var ch Channel
	return &ch, c.Do(ctx, http.MethodPost, "/users/@me/channels", map[string][]string{"recipients": recipientIDs}, &ch)
}

// AddRecipient adds a user to a group DM.
func (c *Client) AddRecipient(ctx context.Context, channelID, userID string) error {
	return c.Do(ctx, http.MethodPut, channelPath(channelID)+"/recipients/"+userID, nil, nil)
}

// RemoveRecipient removes a user from a group DM. With silent set, the request carries
// silent=true.
func (c *Client) RemoveRecipient(ctx context.Context, channelID, userID string, silent bool) error {
	path := channelPath(channelID) + "/recipients/" + userID
	if silent {
		path += "?" + url.Values{"silent": {"true"}}.Encode()
	}
	return c.Do(ctx, http.MethodDelete, path, nil, nil)
}

// PinDM pins a DM or group DM in the current user's channel list.
func (c *Client) PinDM(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodPut, "/users/@me/channels/"+channelID+"/pin", nil, nil)
}

// UnpinDM unpins a DM or group DM from the current user's channel list.
func (c *Client) UnpinDM(ctx context.Context, channelID string) error {
	return c.Do(ctx, http.MethodDelete, "/users/@me/channels/"+channelID+"/pin", nil, nil)
}
