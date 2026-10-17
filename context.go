package fluxer

import (
	"context"
	"strings"
	"unicode"
)

type reasonKey struct{}

func WithReason(ctx context.Context, reason string) context.Context {
	reason = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, reason))
	if r := []rune(reason); len(r) > 512 {
		reason = strings.TrimSpace(string(r[:512]))
	}
	if reason == "" {
		return ctx
	}
	return context.WithValue(ctx, reasonKey{}, reason)
}
