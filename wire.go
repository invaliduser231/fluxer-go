package fluxer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"github.com/coder/websocket"
	"github.com/klauspost/compress/zstd"
)

type wire struct {
	ws       *websocket.Conn
	compress bool
	mu       sync.Mutex
	enc      *zstd.Encoder
	out      bytes.Buffer
	in       *inflater
}

func (w *wire) write(ctx context.Context, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if w.enc == nil {
		return w.ws.Write(context.WithoutCancel(ctx), websocket.MessageText, data)
	}
	w.out.Reset()
	if _, err := w.enc.Write(data); err != nil {
		return err
	}
	if err := w.enc.Flush(); err != nil {
		return err
	}
	return w.ws.Write(context.WithoutCancel(ctx), websocket.MessageBinary, w.out.Bytes())
}

func (w *wire) read(ctx context.Context) (payload, error) {
	if w.in != nil {
		return w.in.next(ctx)
	}
	typ, data, err := w.ws.Read(ctx)
	if err != nil {
		return payload{}, closeError(err)
	}
	if typ == websocket.MessageBinary {
		if !w.compress {
			return payload{}, errors.New("fluxer: unexpected binary frame")
		}
		if err := w.startCompression(ctx, data); err != nil {
			return payload{}, err
		}
		return w.in.next(ctx)
	}
	var p payload
	return p, json.Unmarshal(data, &p)
}

func (w *wire) startCompression(ctx context.Context, first []byte) error {
	enc, err := zstd.NewWriter(&w.out, zstd.WithEncoderConcurrency(1), zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return err
	}
	w.mu.Lock()
	w.enc = enc
	w.mu.Unlock()
	w.in = inflate(ctx, w.ws, first)
	return nil
}

func (w *wire) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.enc != nil {
		_ = w.enc.Close()
	}
}

type inflater struct {
	frames chan payload
	done   chan struct{}
	err    error
}

func inflate(ctx context.Context, ws *websocket.Conn, first []byte) *inflater {
	pr, pw := io.Pipe()
	in := &inflater{frames: make(chan payload), done: make(chan struct{})}
	var mu sync.Mutex
	var readErr error
	go func() {
		data := first
		for {
			if _, err := pw.Write(data); err != nil {
				return
			}
			var err error
			if _, data, err = ws.Read(ctx); err != nil {
				mu.Lock()
				readErr = closeError(err)
				mu.Unlock()
				pw.CloseWithError(readErr)
				return
			}
		}
	}()
	go func() {
		defer close(in.done)
		zr, err := zstd.NewReader(pr, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(64<<20))
		if err != nil {
			in.err = err
			pr.CloseWithError(err)
			return
		}
		defer zr.Close()
		dec := json.NewDecoder(zr)
		for {
			var p payload
			if err := dec.Decode(&p); err != nil {
				mu.Lock()
				if readErr != nil {
					err = readErr
				}
				mu.Unlock()
				in.err = err
				pr.CloseWithError(err)
				return
			}
			select {
			case in.frames <- p:
			case <-ctx.Done():
				in.err = ctx.Err()
				pr.CloseWithError(in.err)
				return
			}
		}
	}()
	return in
}

func (in *inflater) next(ctx context.Context) (payload, error) {
	select {
	case p := <-in.frames:
		return p, nil
	case <-in.done:
		return payload{}, in.err
	case <-ctx.Done():
		return payload{}, ctx.Err()
	}
}

func closeError(err error) error {
	var ce websocket.CloseError
	if errors.As(err, &ce) {
		return &CloseError{Code: int(ce.Code), Reason: ce.Reason}
	}
	return err
}
