package fluxer

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const (
	TemporaryBanRetry = time.Hour
	PermanentBanRetry = 24 * time.Hour
)

type BanError struct {
	Until     time.Time
	Permanent bool
}

func (e *BanError) Error() string {
	if e.Permanent {
		return "fluxer: this address is banned"
	}
	return "fluxer: this address is blocked until " + e.Until.UTC().Format(time.RFC3339)
}

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
