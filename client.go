package fluxer

import (
	"bytes"
	"cmp"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultAPI, DefaultGateway and DefaultMedia are the endpoints of the public
// fluxer.app instance. Use Discover to find the endpoints of a self-hosted instance.
const (
	DefaultAPI       = "https://api.fluxer.app/v1"
	DefaultGateway   = "wss://gateway.fluxer.app"
	DefaultMedia     = "https://fluxerusercontent.com"
	defaultBodyLimit = 8 << 20
)

// Client is a REST client for the Fluxer API. It learns rate limit buckets from
// response headers and waits before sending a request into a bucket that is
// exhausted or while a global rate limit is active. A Client is safe for
// concurrent use once its fields are set.
//
// After the API answers a request with 401, the Client considers its token
// rejected and fails every later call with an *APIError matching
// ErrTokenRejected without contacting the API.
//
// MaxRetries limits retries of transient failures, see Do. BodyLimit caps how
// many bytes of a response body are read and defaults to 8 MiB. Observe, when
// set, is called after every HTTP round trip with the route in MetricRoute form,
// the status code (0 on a transport error) and the duration.
type Client struct {
	BaseURL    string
	Token      string
	UserAgent  string
	HTTP       *http.Client
	MaxRetries int
	BodyLimit  int64
	Observe    func(route string, status int, took time.Duration)

	rejected    atomic.Bool
	mu          sync.Mutex
	routes      map[string]string
	buckets     map[string]time.Time
	globalUntil time.Time
}

// NewClient returns a Client for the public fluxer.app API that authenticates
// as the bot with the given token. Surrounding whitespace and a leading "Bot "
// prefix are removed from token. The client uses a 15 second HTTP timeout and
// retries up to 3 times.
func NewClient(token string) *Client {
	return &Client{
		BaseURL:    DefaultAPI,
		Token:      strings.TrimPrefix(strings.TrimSpace(token), "Bot "),
		UserAgent:  "fluxer-go/" + Version + " (+https://github.com/invaliduser231/fluxer-go)",
		HTTP:       &http.Client{Timeout: 15 * time.Second},
		MaxRetries: 3,
	}
}

type nonced interface{ nonce() string }

// Do sends a request to path, relative to BaseURL, with body encoded as JSON,
// and decodes the response into out when out is non-nil. A nil body sends no
// request body. Responses with a status of 300 or higher are returned as
// *APIError.
//
// Do waits for known rate limits before sending. It retries 429 responses after
// the advertised delay, and 423, 502, 503 and 504 responses after Retry-After or
// one second, up to MaxRetries times. A POST is only retried on 429, unless body
// carries a nonce, as a MessageSend with Nonce set does. Transport errors are not
// retried. A reason set with WithReason is sent as X-Audit-Log-Reason.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	return c.send(ctx, method, path, "application/json", payload, body, out)
}

func (c *Client) send(ctx context.Context, method, path, contentType string, payload []byte, body, out any) error {
	if c.rejected.Load() {
		return &APIError{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: "token was rejected earlier"}
	}
	n, hasNonce := body.(nonced)
	safe := method != http.MethodPost || hasNonce && n.nonce() != ""
	route := routeKey(method, path)
	for attempt := 0; ; attempt++ {
		if err := c.wait(ctx, route); err != nil {
			return err
		}
		retry, err := c.once(ctx, method, path, route, contentType, payload, out)
		var apiErr *APIError
		transient := retry > 0 && (safe || errors.As(err, &apiErr) && apiErr.Status == http.StatusTooManyRequests)
		if !transient || attempt >= c.MaxRetries {
			return err
		}
		if err := sleep(ctx, retry); err != nil {
			return err
		}
	}
}

func (c *Client) once(ctx context.Context, method, path, route, contentType string, payload []byte, out any) (time.Duration, error) {
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(c.BaseURL, "/")+path, reader)
	if err != nil {
		return 0, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bot "+c.Token)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	if payload != nil {
		req.Header.Set("Content-Type", contentType)
	}
	if reason, ok := ctx.Value(reasonKey{}).(string); ok {
		req.Header.Set("X-Audit-Log-Reason", reason)
	}
	if h, ok := ctx.Value(headersKey{}).(http.Header); ok {
		for k, v := range h {
			req.Header[k] = v
		}
	}
	start := time.Now()
	resp, err := c.httpClient().Do(req)
	if c.Observe != nil {
		status := 0
		if err == nil {
			status = resp.StatusCode
		}
		c.Observe(MetricRoute(route), status, time.Since(start))
	}
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, cmp.Or(c.BodyLimit, defaultBodyLimit)))
	if err != nil {
		return 0, err
	}
	c.track(route, resp.Header)

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		var rl struct {
			RetryAfter float64 `json:"retry_after"`
			Global     bool    `json:"global"`
		}
		_ = json.Unmarshal(data, &rl)
		wait := max(cmp.Or(seconds(rl.RetryAfter), seconds(headerFloat(resp.Header, "Retry-After"))), 50*time.Millisecond)
		if rl.Global {
			c.mu.Lock()
			c.globalUntil = time.Now().Add(wait)
			c.mu.Unlock()
		}
		e := apiError(resp, data)
		e.RetryAfter = wait
		return wait, e
	case http.StatusLocked, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return max(seconds(headerFloat(resp.Header, "Retry-After")), time.Second), apiError(resp, data)
	case http.StatusUnauthorized:
		c.rejected.Store(true)
		return 0, apiError(resp, data)
	}
	if resp.StatusCode >= 300 {
		return 0, apiError(resp, data)
	}
	if out != nil && len(data) > 0 {
		return 0, json.Unmarshal(data, out)
	}
	return 0, nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) track(route string, h http.Header) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.routes == nil {
		c.routes, c.buckets = map[string]string{}, map[string]time.Time{}
	}
	key := route
	if b := h.Get("X-RateLimit-Bucket"); b != "" {
		c.routes[route] = b
		key = b
	}
	if h.Get("X-RateLimit-Remaining") != "0" {
		return
	}
	after := seconds(headerFloat(h, "X-RateLimit-Reset-After"))
	if limit := headerFloat(h, "X-RateLimit-Limit"); limit > 1 {
		after = time.Duration(float64(after) / limit)
	}
	if after > 0 {
		c.buckets[key] = time.Now().Add(after)
	}
}

func (c *Client) wait(ctx context.Context, route string) error {
	c.mu.Lock()
	key := route
	if b, ok := c.routes[route]; ok {
		key = b
	}
	until := c.buckets[key]
	if c.globalUntil.After(until) {
		until = c.globalUntil
	}
	c.mu.Unlock()
	return sleep(ctx, time.Until(until))
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func seconds(f float64) time.Duration { return time.Duration(f * float64(time.Second)) }

func headerFloat(h http.Header, key string) float64 {
	f, _ := strconv.ParseFloat(h.Get(key), 64)
	return f
}

func routeKey(method, path string) string {
	path, _, _ = strings.Cut(path, "?")
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if i == 0 || p == "" {
			continue
		}
		prev := parts[i-1]
		switch {
		case prev == "reactions":
			parts[i] = ":emoji"
		case i >= 2 && parts[i-2] == "webhooks":
			parts[i] = ":token"
		case prev != "channels" && prev != "guilds" && prev != "webhooks" && isSnowflake(p):
			parts[i] = ":id"
		}
	}
	return method + " " + strings.Join(parts, "/")
}

func isSnowflake(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

func newNonce() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
