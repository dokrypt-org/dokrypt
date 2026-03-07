package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"net/http"

	"github.com/gorilla/websocket"
)

type WSClient struct {
	url       string
	conn      *websocket.Conn
	nextID    atomic.Int64
	subs      map[string]chan json.RawMessage
	mu        sync.RWMutex
	done      chan struct{}
	responses chan json.RawMessage // channel for RPC responses (non-subscription messages)
}

func NewWSClient(url string) *WSClient {
	return &WSClient{
		url:       url,
		subs:      make(map[string]chan json.RawMessage),
		done:      make(chan struct{}),
		responses: make(chan json.RawMessage, 16),
	}
}

func (w *WSClient) Connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: http.DefaultClient.Timeout,
	}
	conn, _, err := dialer.DialContext(ctx, w.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", w.url, err)
	}
	w.conn = conn
	go w.readLoop()
	return nil
}

func (w *WSClient) Subscribe(ctx context.Context, method string, params ...any) (<-chan json.RawMessage, string, error) {
	if w.conn == nil {
		return nil, "", fmt.Errorf("not connected")
	}

	id := w.nextID.Add(1)
	allParams := []any{method}
	allParams = append(allParams, params...)

	req := Request{
		JSONRPC: "2.0",
		Method:  "eth_subscribe",
		Params:  allParams,
		ID:      id,
	}

	if err := w.conn.WriteJSON(req); err != nil {
		return nil, "", fmt.Errorf("failed to send subscribe request: %w", err)
	}

	var raw json.RawMessage
	select {
	case raw = <-w.responses:
	case <-ctx.Done():
		return nil, "", ctx.Err()
	case <-w.done:
		return nil, "", fmt.Errorf("client closed")
	}

	var resp Response
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, "", fmt.Errorf("failed to parse subscribe response: %w", err)
	}
	if resp.Error != nil {
		return nil, "", resp.Error
	}

	var subID string
	if err := json.Unmarshal(resp.Result, &subID); err != nil {
		return nil, "", fmt.Errorf("failed to parse subscription ID: %w", err)
	}

	ch := make(chan json.RawMessage, 100)
	w.mu.Lock()
	w.subs[subID] = ch
	w.mu.Unlock()

	return ch, subID, nil
}

func (w *WSClient) Unsubscribe(subID string) error {
	w.mu.Lock()
	ch, ok := w.subs[subID]
	if ok {
		close(ch)
		delete(w.subs, subID)
	}
	w.mu.Unlock()

	if w.conn == nil {
		return nil
	}

	id := w.nextID.Add(1)
	req := Request{
		JSONRPC: "2.0",
		Method:  "eth_unsubscribe",
		Params:  []any{subID},
		ID:      id,
	}
	return w.conn.WriteJSON(req)
}

func (w *WSClient) Close() error {
	close(w.done)
	w.mu.Lock()
	for id, ch := range w.subs {
		close(ch)
		delete(w.subs, id)
	}
	w.mu.Unlock()
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

type subscriptionNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Subscription string          `json:"subscription"`
		Result       json.RawMessage `json:"result"`
	} `json:"params"`
}

func (w *WSClient) readLoop() {
	for {
		select {
		case <-w.done:
			return
		default:
		}

		_, message, err := w.conn.ReadMessage()
		if err != nil {
			slog.Debug("websocket read error", "error", err)
			return
		}

		var notif subscriptionNotification
		if err := json.Unmarshal(message, &notif); err == nil && notif.Method == "eth_subscription" {
			w.mu.RLock()
			ch, ok := w.subs[notif.Params.Subscription]
			w.mu.RUnlock()

			if ok {
				select {
				case ch <- notif.Params.Result:
				default:
					slog.Warn("subscription channel full, dropping message", "sub", notif.Params.Subscription)
				}
			}
			continue
		}

		select {
		case w.responses <- message:
		case <-w.done:
			return
		}
	}
}
