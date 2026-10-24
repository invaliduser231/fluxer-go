package fluxer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultInstance is the origin of the public Fluxer instance.
const (
	DefaultInstance = "https://fluxer.app"
	discoveryLimit  = 256 << 10
)

// Instance holds the endpoints and features of a Fluxer instance, as read from
// its discovery document at /.well-known/fluxer. API already includes the /v1
// suffix. HTTP, when non-nil, is used by clients and sessions created from the
// Instance.
type Instance struct {
	Origin      string
	API         string
	Gateway     string
	Media       string
	Webapp      string
	Invite      string
	UploadRelay string
	CodeVersion int
	Features    InstanceFeatures
	HTTP        *http.Client
}

// InstanceFeatures lists the optional features an instance reports in its
// discovery document.
type InstanceFeatures struct {
	VoiceEnabled               bool `json:"voice_enabled"`
	SelfHosted                 bool `json:"self_hosted"`
	PresignedAttachmentUploads bool `json:"presigned_attachment_uploads"`
	DiscoveryEnabled           bool `json:"discovery_enabled"`
}

// NewClient returns a Client for the instance's API, using the instance's HTTP
// client if it has one.
func (i *Instance) NewClient(token string) *Client {
	c := NewClient(token)
	c.BaseURL = i.API
	if i.HTTP != nil {
		c.HTTP = i.HTTP
	}
	return c
}

// Discover reads the discovery document of the instance at origin with
// http.DefaultClient. See DiscoverWith.
func Discover(ctx context.Context, origin string) (*Instance, error) {
	return DiscoverWith(ctx, http.DefaultClient, origin)
}

// DiscoverWith fetches origin + "/.well-known/fluxer" with client and returns
// the instance it describes. origin should already be normalized, see
// NormalizeOrigin. The request times out after 10 seconds. It fails when the
// server does not answer 200, the document is larger than 256 KiB, or it lacks
// the api_public or gateway endpoint. For an https origin it also fails when a
// redirect leaves https or when the API is not served over https or the gateway
// over wss, so the token never travels in plain text. A client other than
// http.DefaultClient is kept in Instance.HTTP.
func DiscoverWith(ctx context.Context, client *http.Client, origin string) (*Instance, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	origin = strings.TrimSuffix(origin, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+"/.well-known/fluxer", nil)
	if err != nil {
		return nil, err
	}
	host := req.URL.Host
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	secure := req.URL.Scheme == "https"
	if secure && resp.Request.URL.Scheme != "https" {
		return nil, fmt.Errorf("fluxer: the discovery request to %s was redirected away from https", host)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fluxer: %s has no discovery document, it answered with status %d", host, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, discoveryLimit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > discoveryLimit {
		return nil, fmt.Errorf("fluxer: the discovery document of %s is too large", host)
	}
	var doc struct {
		APICodeVersion int `json:"api_code_version"`
		Endpoints      struct {
			APIPublic   string `json:"api_public"`
			Gateway     string `json:"gateway"`
			Media       string `json:"media"`
			Webapp      string `json:"webapp"`
			Invite      string `json:"invite"`
			UploadRelay string `json:"upload_relay"`
		} `json:"endpoints"`
		Features InstanceFeatures `json:"features"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("fluxer: %s did not answer with a discovery document", host)
	}
	e := doc.Endpoints
	if e.APIPublic == "" || e.Gateway == "" {
		return nil, fmt.Errorf("fluxer: the discovery document of %s has no api_public or gateway endpoint", host)
	}
	if err := checkEndpoint(host, "api_public", e.APIPublic, secure, "https", "http"); err != nil {
		return nil, err
	}
	if err := checkEndpoint(host, "gateway", e.Gateway, secure, "wss", "ws"); err != nil {
		return nil, err
	}
	inst := &Instance{
		Origin:      origin,
		API:         strings.TrimSuffix(e.APIPublic, "/") + "/v1",
		Gateway:     e.Gateway,
		Media:       strings.TrimSuffix(e.Media, "/"),
		Webapp:      strings.TrimSuffix(e.Webapp, "/"),
		Invite:      strings.TrimSuffix(e.Invite, "/"),
		UploadRelay: strings.TrimSuffix(e.UploadRelay, "/"),
		CodeVersion: doc.APICodeVersion,
		Features:    doc.Features,
	}
	if client != http.DefaultClient {
		inst.HTTP = client
	}
	return inst, nil
}

func checkEndpoint(host, name, raw string, secure bool, tls, plain string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("fluxer: the %s endpoint of %s is not a web address", name, host)
	}
	if u.Scheme == tls || u.Scheme == plain && !secure {
		return nil
	}
	if secure {
		return fmt.Errorf("fluxer: the %s endpoint of %s must use %s", name, host, tls)
	}
	return fmt.Errorf("fluxer: the %s endpoint of %s must use %s or %s", name, host, tls, plain)
}

// NormalizeOrigin turns user input such as "Chat.Example.com" into a lowercase
// origin like "https://chat.example.com". It adds https:// when no scheme is
// given and rejects schemes other than http and https as well as addresses with
// a path, query, fragment or user info.
func NormalizeOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("fluxer: %q is not a web address", raw)
	}
	if scheme := strings.ToLower(u.Scheme); scheme != "https" && scheme != "http" {
		return "", errors.New("fluxer: an instance must be served over http or https")
	}
	if u.User != nil || strings.Trim(u.Path, "/") != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("fluxer: an instance address has no path, query or credentials")
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host), nil
}
