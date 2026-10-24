package fluxer

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ErrTokenRejected matches, through errors.Is, every *APIError with status 401.
// Once a Client has seen a 401 it returns such an error for all later calls.
var ErrTokenRejected = errors.New("fluxer: token rejected")

// FieldError describes a validation problem with one field of a request body.
// Path names the field.
type FieldError struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// APIError is an error response from the Fluxer API. Code is the error code
// from the response body; when the body has none, Code is derived from the HTTP
// status and Message holds the raw body. RequiredScope names the OAuth2 scope
// the token lacks, if the API reports one. RequestID comes from the X-Request-ID
// header and RetryAfter is set for 429 responses.
type APIError struct {
	Status        int
	Code          string       `json:"code"`
	Message       string       `json:"message"`
	Errors        []FieldError `json:"errors"`
	RequiredScope string       `json:"required_scope"`
	RequestID     string
	RetryAfter    time.Duration
}

// Error formats the status, code and message, followed by the first field
// error if there is one.
func (e *APIError) Error() string {
	msg := fmt.Sprintf("fluxer: %d %s: %s", e.Status, e.Code, e.Message)
	if len(e.Errors) > 0 {
		f := e.Errors[0]
		msg += fmt.Sprintf(" (%s: %s)", f.Path, cmpOr(f.Message, f.Code))
	}
	return msg
}

// Is reports whether target is ErrTokenRejected and the error has status 401.
func (e *APIError) Is(target error) bool {
	return target == ErrTokenRejected && e.Status == http.StatusUnauthorized
}

// ErrorCode returns the Code of the first *APIError in err's chain, or an
// empty string if there is none.
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
