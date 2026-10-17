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

const (
	DefaultInstance = "https://fluxer.app"
	discoveryLimit  = 256 << 10
)

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

type InstanceFeatures struct {
	VoiceEnabled               bool `json:"voice_enabled"`
	SelfHosted                 bool `json:"self_hosted"`
	PresignedAttachmentUploads bool `json:"presigned_attachment_uploads"`
	DiscoveryEnabled           bool `json:"discovery_enabled"`
}

func (i *Instance) NewClient(token string) *Client {
	c := NewClient(token)
	c.BaseURL = i.API
	if i.HTTP != nil {
		c.HTTP = i.HTTP
	}
	return c
}

func Discover(ctx context.Context, origin string) (*Instance, error) {
	return DiscoverWith(ctx, http.DefaultClient, origin)
}

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
