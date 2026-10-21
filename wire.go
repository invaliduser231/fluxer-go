package fluxer

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/coder/websocket"
)

type wire struct {
	ws *websocket.Conn
	mu sync.Mutex
}

func (w *wire) write(ctx context.Context, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	return w.ws.Write(context.WithoutCancel(ctx), websocket.MessageText, data)
}

func (w *wire) read(ctx context.Context) (payload, error) {
	typ, data, err := w.ws.Read(ctx)
	if err != nil {
		return payload{}, closeError(err)
	}
	if typ != websocket.MessageText {
		return payload{}, errors.New("fluxer: unexpected binary frame")
	}
	var p payload
	return p, json.Unmarshal(data, &p)
}

func closeError(err error) error {
	var ce websocket.CloseError
	if errors.As(err, &ce) {
		return &CloseError{Code: int(ce.Code), Reason: ce.Reason}
	}
	return err
}
