package state

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/invaliduser231/fluxer-go"
)

var ErrChunkTimeout = errors.New("state: member chunks did not arrive in time")

type MemberRequester interface {
	RequestGuildMembers(ctx context.Context, r fluxer.RequestMembers) error
}

type RateLimitedError struct {
	GuildID    string
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("state: member request for %s is rate limited for %s", e.GuildID, e.RetryAfter)
}

type Chunker struct {
	Requester MemberRequester
	Filter    func(guildID string) bool
	Timeout   time.Duration
	OnError   func(guildID string, err error)

	mu      sync.Mutex
	pending map[string]*chunkRequest
}

type chunkRequest struct {
	guildID   string
	members   []fluxer.Member
	presences []fluxer.Presence
	seen      map[int]bool
	count     int
	err       error
	done      chan struct{}
}

func NewChunker(r MemberRequester) *Chunker {
	return &Chunker{Requester: r, pending: map[string]*chunkRequest{}}
}

func ChunkAll(string) bool { return true }

func ChunkGuilds(ids ...string) func(string) bool {
	return func(id string) bool { return slices.Contains(ids, id) }
}

func newNonce() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (k *Chunker) Request(ctx context.Context, r fluxer.RequestMembers) ([]fluxer.Member, error) {
	r.Nonce = newNonce()
	req := &chunkRequest{guildID: r.GuildID, seen: map[int]bool{}, done: make(chan struct{})}
	k.mu.Lock()
	if k.pending == nil {
		k.pending = map[string]*chunkRequest{}
	}
	k.pending[r.Nonce] = req
	k.mu.Unlock()
	defer func() {
		k.mu.Lock()
		delete(k.pending, r.Nonce)
		k.mu.Unlock()
	}()
	if err := k.Requester.RequestGuildMembers(ctx, r); err != nil {
		return nil, err
	}
	timer := time.NewTimer(cmp.Or(k.Timeout, 30*time.Second))
	defer timer.Stop()
	select {
	case <-req.done:
	case <-timer.C:
		return nil, ErrChunkTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if req.err != nil {
		return nil, req.err
	}
	return req.members, nil
}

func (k *Chunker) RequestAll(ctx context.Context, guildID string) ([]fluxer.Member, error) {
	return k.Request(ctx, fluxer.RequestMembers{GuildID: guildID})
}

func (k *Chunker) Handle(event string, data json.RawMessage) {
	switch event {
	case "GUILD_MEMBERS_CHUNK":
		var e fluxer.GuildMembersChunk
		if json.Unmarshal(data, &e) != nil || e.Nonce == "" {
			return
		}
		k.mu.Lock()
		defer k.mu.Unlock()
		req := k.pending[e.Nonce]
		if req == nil || req.seen[e.ChunkIndex] {
			return
		}
		req.seen[e.ChunkIndex] = true
		req.count = e.ChunkCount
		for _, m := range e.Members {
			m.GuildID = e.GuildID
			req.members = append(req.members, m)
		}
		req.presences = append(req.presences, e.Presences...)
		if len(req.seen) >= req.count {
			close(req.done)
		}
	case "RATE_LIMITED":
		var e fluxer.RateLimited
		if json.Unmarshal(data, &e) != nil || e.Opcode != 8 {
			return
		}
		k.mu.Lock()
		defer k.mu.Unlock()
		for nonce, req := range k.pending {
			if nonce == e.Meta["nonce"] || e.Meta["nonce"] == "" && req.guildID == e.Meta["guild_id"] {
				req.err = &RateLimitedError{GuildID: req.guildID, RetryAfter: time.Duration(e.RetryAfter * float64(time.Second))}
				close(req.done)
				delete(k.pending, nonce)
			}
		}
	case "GUILD_CREATE":
		if k.Filter == nil {
			return
		}
		var e struct {
			ID          string `json:"id"`
			Unavailable bool   `json:"unavailable"`
		}
		if json.Unmarshal(data, &e) != nil || e.Unavailable || !k.Filter(e.ID) {
			return
		}
		go func() {
			if _, err := k.RequestAll(context.Background(), e.ID); err != nil && k.OnError != nil {
				k.OnError(e.ID, err)
			}
		}()
	}
}

type readyState struct {
	mu      sync.Mutex
	pending map[string]bool
	done    chan struct{}
}

func (r *readyState) start(guilds []fluxer.UnavailableGuild) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pending = map[string]bool{}
	for _, g := range guilds {
		r.pending[g.ID] = true
	}
	r.done = make(chan struct{})
	if len(r.pending) == 0 {
		close(r.done)
	}
}

func (r *readyState) resolve(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pending == nil || !r.pending[id] {
		return
	}
	delete(r.pending, id)
	if len(r.pending) == 0 {
		close(r.done)
	}
}

func (r *readyState) wait() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done == nil {
		r.done = make(chan struct{})
	}
	return r.done
}

func (c *Cache) WaitGuilds(ctx context.Context) error {
	select {
	case <-c.ready.wait():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Cache) PendingGuilds() int {
	c.ready.mu.Lock()
	defer c.ready.mu.Unlock()
	return len(c.ready.pending)
}
