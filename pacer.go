package fluxer

import (
	"context"
	"sync"
	"time"
)

type Reserve func(ctx context.Context, key string, step time.Duration) (time.Duration, error)

type Pacer struct {
	Reserve Reserve
	Key     string

	interval time.Duration
	mu       sync.Mutex
	next     time.Time
}

func NewPacer(perMinute int) *Pacer {
	return &Pacer{interval: time.Minute / time.Duration(max(perMinute, 1))}
}

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
