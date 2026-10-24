package fluxer

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// TemporaryBanRetry is the wait used for a temporary IP ban that comes without
// an expiry time. PermanentBanRetry is the minimum wait used for a permanent IP
// ban.
const (
	TemporaryBanRetry = time.Hour
	PermanentBanRetry = 24 * time.Hour
)

// BanError reports that the server refused the connection because the
// client's IP address is banned. Until is the earliest time a new attempt makes
// sense. A Session returns it when the gateway handshake is answered with 403
// and a ban code.
type BanError struct {
	Until     time.Time
	Permanent bool
}

// Error describes the ban and, for a temporary ban, when it ends.
func (e *BanError) Error() string {
	if e.Permanent {
		return "fluxer: this address is banned"
	}
	return "fluxer: this address is blocked until " + e.Until.UTC().Format(time.RFC3339)
}

// Ban builds a BanError from a Fluxer error code and expiry time. It returns
// nil unless code is GLOBAL_IP_TEMPORARILY_BANNED or GLOBAL_IP_BANNED. A
// temporary ban without expiry lasts TemporaryBanRetry from now; a permanent ban
// lasts until expires or PermanentBanRetry from now, whichever is later.
func Ban(code string, expires time.Time) *BanError {
	var b BanError
	switch code {
	case "GLOBAL_IP_TEMPORARILY_BANNED":
		b.Until = expires
		if b.Until.IsZero() {
			b.Until = time.Now().Add(TemporaryBanRetry)
		}
	case "GLOBAL_IP_BANNED":
		b.Permanent = true
		b.Until = time.Now().Add(PermanentBanRetry)
		if expires.After(b.Until) {
			b.Until = expires
		}
	default:
		return nil
	}
	return &b
}

func banned(resp *http.Response) *BanError {
	if resp == nil || resp.StatusCode != http.StatusForbidden || resp.Body == nil {
		return nil
	}
	var body struct {
		Code      string    `json:"code"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if json.Unmarshal(data, &body) != nil {
		return nil
	}
	return Ban(body.Code, body.ExpiresAt)
}
