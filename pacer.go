package fluxer

import (
	"context"
	"sync"
	"time"
)

// Reserve claims the next free slot for key in a rate limiter shared between
// processes, for example one backed by Redis, and returns how long the caller
// has to wait for that slot. step is the spacing between slots.
type Reserve func(ctx context.Context, key string, step time.Duration) (time.Duration, error)

// Pacer spaces out events, such as gateway identifies, at a fixed interval.
// When Reserve is set, Wait asks it for a slot under Key and falls back to
// pacing within the process if Reserve fails or takes longer than two
// seconds.
type Pacer struct {
	Reserve Reserve
	Key     string

	interval time.Duration
	mu       sync.Mutex
	next     time.Time
}

// NewPacer returns a Pacer that allows perMinute events per minute, evenly
// spaced. Values below 1 count as 1.
func NewPacer(perMinute int) *Pacer {
	return &Pacer{interval: time.Minute / time.Duration(max(perMinute, 1))}
}

// Wait blocks until the caller's slot has come or ctx is done. Every call
// takes the next slot, so callers proceed one interval apart.
func (p *Pacer) Wait(ctx context.Context) error {
	if p.Reserve != nil {
		rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		wait, err := p.Reserve(rctx, p.Key, p.interval)
		cancel()
		if err == nil {
			return sleep(ctx, wait)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	p.mu.Lock()
	slot := time.Now()
	if p.next.After(slot) {
		slot = p.next
	}
	p.next = slot.Add(p.interval)
	p.mu.Unlock()
	return sleep(ctx, time.Until(slot))
}
