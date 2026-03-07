package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func httpToWS(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

func TestNewWSClient(t *testing.T) {
	wsc := NewWSClient("ws://localhost:8546")
	require.NotNil(t, wsc)
	assert.Equal(t, "ws://localhost:8546", wsc.url)
	assert.NotNil(t, wsc.subs)
	assert.Empty(t, wsc.subs)
	assert.NotNil(t, wsc.done)
	assert.Nil(t, wsc.conn)
}

func TestWSClient_Connect_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsc := NewWSClient(httpToWS(server.URL))
	err := wsc.Connect(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, wsc.conn)
	defer wsc.Close()
}

func TestWSClient_Connect_InvalidURL(t *testing.T) {
	wsc := NewWSClient("ws://127.0.0.1:1")
	err := wsc.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect")
}

func TestWSClient_Subscribe_NotConnected(t *testing.T) {
	wsc := NewWSClient("ws://localhost:9999")
	ch, subID, err := wsc.Subscribe(context.Background(), "newHeads")
	assert.Nil(t, ch)
	assert.Empty(t, subID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestWSClient_Subscribe_Success(t *testing.T) {
	subIDValue := "0xabc123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var req Request
		if err := conn.ReadJSON(&req); err != nil {
			return
		}
		assert.Equal(t, "eth_subscribe", req.Method)

		subIDJSON, _ := json.Marshal(subIDValue)
		resp := Response{
			JSONRPC: "2.0",
			Result:  json.RawMessage(subIDJSON),
			ID:      req.ID,
		}
		conn.WriteJSON(resp)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsc := NewWSClient(httpToWS(server.URL))
	err := wsc.Connect(context.Background())
	require.NoError(t, err)
	defer wsc.Close()

	ch, subID, err := wsc.Subscribe(context.Background(), "newHeads")
	require.NoError(t, err)
	assert.Equal(t, subIDValue, subID)
	assert.NotNil(t, ch)
}

func TestWSClient_Subscribe_RPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var req Request
		conn.ReadJSON(&req)

		resp := Response{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: -32600, Message: "Invalid subscription"},
			ID:      req.ID,
		}
		conn.WriteJSON(resp)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsc := NewWSClient(httpToWS(server.URL))
	require.NoError(t, wsc.Connect(context.Background()))
	defer wsc.Close()

	ch, subID, err := wsc.Subscribe(context.Background(), "badSub")
	assert.Nil(t, ch)
	assert.Empty(t, subID)
	require.Error(t, err)
}

func TestWSClient_Subscribe_WithParams(t *testing.T) {
	subIDValue := "0xdef456"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var req Request
		conn.ReadJSON(&req)
		assert.Equal(t, "eth_subscribe", req.Method)

		paramsJSON, _ := json.Marshal(req.Params)
		assert.Contains(t, string(paramsJSON), "logs")

		subIDJSON, _ := json.Marshal(subIDValue)
		resp := Response{JSONRPC: "2.0", Result: json.RawMessage(subIDJSON), ID: req.ID}
		conn.WriteJSON(resp)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsc := NewWSClient(httpToWS(server.URL))
	require.NoError(t, wsc.Connect(context.Background()))
	defer wsc.Close()

	filter := map[string]any{"address": "0x1234"}
	ch, subID, err := wsc.Subscribe(context.Background(), "logs", filter)
	require.NoError(t, err)
	assert.Equal(t, subIDValue, subID)
	assert.NotNil(t, ch)
}

func TestWSClient_ReadLoop_DeliversNotifications(t *testing.T) {
	subIDValue := "0xsub1"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var req Request
		conn.ReadJSON(&req)

		subIDJSON, _ := json.Marshal(subIDValue)
		resp := Response{JSONRPC: "2.0", Result: json.RawMessage(subIDJSON), ID: req.ID}
		conn.WriteJSON(resp)

		time.Sleep(50 * time.Millisecond)
		notif := subscriptionNotification{
			JSONRPC: "2.0",
			Method:  "eth_subscription",
		}
		notif.Params.Subscription = subIDValue
		notif.Params.Result = json.RawMessage(`{"number":"0x1"}`)
		conn.WriteJSON(notif)

		time.Sleep(50 * time.Millisecond)
		notif.Params.Result = json.RawMessage(`{"number":"0x2"}`)
		conn.WriteJSON(notif)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsc := NewWSClient(httpToWS(server.URL))
	require.NoError(t, wsc.Connect(context.Background()))
	defer wsc.Close()

	ch, subID, err := wsc.Subscribe(context.Background(), "newHeads")
	require.NoError(t, err)
	assert.Equal(t, subIDValue, subID)

	select {
	case msg := <-ch:
		assert.Contains(t, string(msg), "0x1")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first notification")
	}

	select {
	case msg := <-ch:
		assert.Contains(t, string(msg), "0x2")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second notification")
	}
}

func TestWSClient_ReadLoop_IgnoresNonSubscriptionMessages(t *testing.T) {
	subIDValue := "0xsub2"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var req Request
		conn.ReadJSON(&req)

		subIDJSON, _ := json.Marshal(subIDValue)
		resp := Response{JSONRPC: "2.0", Result: json.RawMessage(subIDJSON), ID: req.ID}
		conn.WriteJSON(resp)

		time.Sleep(50 * time.Millisecond)
		conn.WriteJSON(map[string]any{
			"jsonrpc": "2.0",
			"method":  "some_other_method",
			"params":  map[string]any{"foo": "bar"},
		})

		time.Sleep(50 * time.Millisecond)
		notif := subscriptionNotification{
			JSONRPC: "2.0",
			Method:  "eth_subscription",
		}
		notif.Params.Subscription = subIDValue
		notif.Params.Result = json.RawMessage(`"real_data"`)
		conn.WriteJSON(notif)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsc := NewWSClient(httpToWS(server.URL))
	require.NoError(t, wsc.Connect(context.Background()))
	defer wsc.Close()

	ch, _, err := wsc.Subscribe(context.Background(), "newHeads")
	require.NoError(t, err)

	select {
	case msg := <-ch:
		assert.Equal(t, `"real_data"`, string(msg))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification")
	}
}

func TestWSClient_ReadLoop_UnknownSubscriptionDropped(t *testing.T) {
	subIDValue := "0xknown"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var req Request
		conn.ReadJSON(&req)

		subIDJSON, _ := json.Marshal(subIDValue)
		resp := Response{JSONRPC: "2.0", Result: json.RawMessage(subIDJSON), ID: req.ID}
		conn.WriteJSON(resp)

		time.Sleep(50 * time.Millisecond)
		notif := subscriptionNotification{JSONRPC: "2.0", Method: "eth_subscription"}
		notif.Params.Subscription = "0xunknown"
		notif.Params.Result = json.RawMessage(`"should_be_dropped"`)
		conn.WriteJSON(notif)

		time.Sleep(50 * time.Millisecond)
		notif.Params.Subscription = subIDValue
		notif.Params.Result = json.RawMessage(`"should_be_delivered"`)
		conn.WriteJSON(notif)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsc := NewWSClient(httpToWS(server.URL))
	require.NoError(t, wsc.Connect(context.Background()))
	defer wsc.Close()

	ch, _, err := wsc.Subscribe(context.Background(), "newHeads")
	require.NoError(t, err)

	select {
	case msg := <-ch:
		assert.Equal(t, `"should_be_delivered"`, string(msg))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification")
	}
}

func TestWSClient_Unsubscribe_Success(t *testing.T) {
	subIDValue := "0xunsub"
	var unsubReceived sync.WaitGroup
	unsubReceived.Add(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var req Request
		conn.ReadJSON(&req)

		subIDJSON, _ := json.Marshal(subIDValue)
		resp := Response{JSONRPC: "2.0", Result: json.RawMessage(subIDJSON), ID: req.ID}
		conn.WriteJSON(resp)

		var unsubReq Request
		conn.ReadJSON(&unsubReq)
		assert.Equal(t, "eth_unsubscribe", unsubReq.Method)
		unsubReceived.Done()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsc := NewWSClient(httpToWS(server.URL))
	require.NoError(t, wsc.Connect(context.Background()))
	defer wsc.Close()

	ch, subID, err := wsc.Subscribe(context.Background(), "newHeads")
	require.NoError(t, err)
	assert.NotNil(t, ch)

	err = wsc.Unsubscribe(subID)
	require.NoError(t, err)

	unsubReceived.Wait()

	_, ok := <-ch
	assert.False(t, ok, "subscription channel should be closed after unsubscribe")

	wsc.mu.RLock()
	_, exists := wsc.subs[subID]
	wsc.mu.RUnlock()
	assert.False(t, exists)
}

func TestWSClient_Unsubscribe_UnknownSubID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsc := NewWSClient(httpToWS(server.URL))
	require.NoError(t, wsc.Connect(context.Background()))
	defer wsc.Close()

	err := wsc.Unsubscribe("0xnonexistent")
	require.NoError(t, err)
}

func TestWSClient_Unsubscribe_NilConn(t *testing.T) {
	wsc := NewWSClient("ws://localhost:9999")
	err := wsc.Unsubscribe("0xany")
	assert.NoError(t, err)
}

func TestWSClient_Close_ConnectedClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var req Request
		conn.ReadJSON(&req)

		subIDJSON, _ := json.Marshal("0xclose")
		resp := Response{JSONRPC: "2.0", Result: json.RawMessage(subIDJSON), ID: req.ID}
		conn.WriteJSON(resp)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsc := NewWSClient(httpToWS(server.URL))
	require.NoError(t, wsc.Connect(context.Background()))

	ch, _, err := wsc.Subscribe(context.Background(), "newHeads")
	require.NoError(t, err)

	err = wsc.Close()
	require.NoError(t, err)

	_, ok := <-ch
	assert.False(t, ok)

	wsc.mu.RLock()
	assert.Empty(t, wsc.subs)
	wsc.mu.RUnlock()
}

func TestWSClient_Close_NotConnected(t *testing.T) {
	wsc := NewWSClient("ws://localhost:9999")
	err := wsc.Close()
	assert.NoError(t, err)
}

func TestWSClient_Close_ClosesMultipleSubscriptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testWSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for i := 0; i < 2; i++ {
			var req Request
			conn.ReadJSON(&req)

			subID := "0xsub" + string(rune('a'+i))
			subIDJSON, _ := json.Marshal(subID)
			resp := Response{JSONRPC: "2.0", Result: json.RawMessage(subIDJSON), ID: req.ID}
			conn.WriteJSON(resp)
		}

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsc := NewWSClient(httpToWS(server.URL))
	require.NoError(t, wsc.Connect(context.Background()))

	ch1, _, err1 := wsc.Subscribe(context.Background(), "newHeads")
	require.NoError(t, err1)
	ch2, _, err2 := wsc.Subscribe(context.Background(), "logs")
	require.NoError(t, err2)

	err := wsc.Close()
	require.NoError(t, err)

	_, ok1 := <-ch1
	assert.False(t, ok1)
	_, ok2 := <-ch2
	assert.False(t, ok2)
}

func TestSubscriptionNotification_JSONRoundTrip(t *testing.T) {
	original := subscriptionNotification{
		JSONRPC: "2.0",
		Method:  "eth_subscription",
	}
	original.Params.Subscription = "0xabc"
	original.Params.Result = json.RawMessage(`{"blockNumber":"0x1"}`)

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded subscriptionNotification
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "2.0", decoded.JSONRPC)
	assert.Equal(t, "eth_subscription", decoded.Method)
	assert.Equal(t, "0xabc", decoded.Params.Subscription)
	assert.JSONEq(t, `{"blockNumber":"0x1"}`, string(decoded.Params.Result))
}
