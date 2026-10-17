package fluxer

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var ErrTokenRejected = errors.New("fluxer: token rejected")

type FieldError struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIError struct {
	Status        int
	Code          string       `json:"code"`
	Message       string       `json:"message"`
	Errors        []FieldError `json:"errors"`
	RequiredScope string       `json:"required_scope"`
	RequestID     string
	RetryAfter    time.Duration
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("fluxer: %d %s: %s", e.Status, e.Code, e.Message)
	if len(e.Errors) > 0 {
		f := e.Errors[0]
		msg += fmt.Sprintf(" (%s: %s)", f.Path, cmpOr(f.Message, f.Code))
	}
	return msg
}

func (e *APIError) Is(target error) bool {
	return target == ErrTokenRejected && e.Status == http.StatusUnauthorized
}

func ErrorCode(err error) string {
	var e *APIError
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

var fallbackCodes = map[int]string{
	400: "BAD_REQUEST", 403: "FORBIDDEN", 404: "NOT_FOUND", 405: "METHOD_NOT_ALLOWED",
	409: "CONFLICT", 410: "GONE", 500: "INTERNAL_SERVER_ERROR", 501: "NOT_IMPLEMENTED",
	502: "BAD_GATEWAY", 503: "SERVICE_UNAVAILABLE", 504: "GATEWAY_TIMEOUT",
}

func apiError(resp *http.Response, data []byte) *APIError {
	e := &APIError{Status: resp.StatusCode, RequestID: resp.Header.Get("X-Request-ID")}
	if json.Unmarshal(data, e) != nil || e.Code == "" {
		e.Code = cmpOr(fallbackCodes[resp.StatusCode], "GENERAL_ERROR")
		e.Message = strings.TrimSpace(string(data))
	}
	return e
}

func cmpOr(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}
