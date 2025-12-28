// ABOUTME: Tests for built-in WebSocket handlers.
// ABOUTME: Validates heartbeat, subscribe, unsubscribe, and token login functionality.

package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bingo-project/websocket/jsonrpc"
)

func TestHeartbeatHandler(t *testing.T) {
	client := &Client{}
	c := &Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "heartbeat"},
		Client:  client,
		Method:  "heartbeat",
	}

	before := time.Now().Unix()
	resp := HeartbeatHandler(c)
	after := time.Now().Unix()

	assert.Nil(t, resp.Error)
	result := resp.Result.(map[string]any)
	assert.Equal(t, "ok", result["status"])

	serverTime := result["server_time"].(int64)
	assert.GreaterOrEqual(t, serverTime, before)
	assert.LessOrEqual(t, serverTime, after)
}

func TestSubscribeHandler(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	client := &Client{
		ID:        "test",
		UserID:    "user-1",
		Platform:  "web",
		LoginTime: 1000,
		hub:       hub,
		Send:      make(chan []byte, 256),
	}

	params, _ := json.Marshal(map[string][]string{"topics": {"market.BTC"}})
	c := &Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "subscribe", Params: params},
		Client:  client,
		Method:  "subscribe",
	}

	resp := SubscribeHandler(c)

	assert.Nil(t, resp.Error)
	result := resp.Result.(map[string]any)
	subscribed := result["subscribed"].([]string)
	assert.Contains(t, subscribed, "market.BTC")
}

func TestSubscribeHandler_InvalidParams(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:  "test",
		hub: hub,
	}

	c := &Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "subscribe", Params: json.RawMessage(`invalid`)},
		Client:  client,
		Method:  "subscribe",
	}

	resp := SubscribeHandler(c)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32602, resp.Error.Code) // JSON-RPC Invalid Params
}

func TestSubscribeHandler_EmptyTopics(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:  "test",
		hub: hub,
	}

	params, _ := json.Marshal(map[string][]string{"topics": {}})
	c := &Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "subscribe", Params: params},
		Client:  client,
		Method:  "subscribe",
	}

	resp := SubscribeHandler(c)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32602, resp.Error.Code) // JSON-RPC Invalid Params
}

func TestUnsubscribeHandler(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	client := &Client{
		ID:        "test",
		UserID:    "user-1",
		Platform:  "web",
		LoginTime: 1000,
		hub:       hub,
		Send:      make(chan []byte, 256),
	}

	params, _ := json.Marshal(map[string][]string{"topics": {"market.BTC"}})
	c := &Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "unsubscribe", Params: params},
		Client:  client,
		Method:  "unsubscribe",
	}

	resp := UnsubscribeHandler(c)

	assert.Nil(t, resp.Error)
	result := resp.Result.(map[string]any)
	unsubscribed := result["unsubscribed"].([]string)
	assert.Contains(t, unsubscribed, "market.BTC")
}

func TestUnsubscribeHandler_InvalidParams(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:  "test",
		hub: hub,
	}

	c := &Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "unsubscribe", Params: json.RawMessage(`invalid`)},
		Client:  client,
		Method:  "unsubscribe",
	}

	resp := UnsubscribeHandler(c)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32602, resp.Error.Code) // JSON-RPC Invalid Params
}

func TestUnsubscribeHandler_EmptyTopics(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:  "test",
		hub: hub,
	}

	params, _ := json.Marshal(map[string][]string{"topics": {}})
	c := &Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "unsubscribe", Params: params},
		Client:  client,
		Method:  "unsubscribe",
	}

	resp := UnsubscribeHandler(c)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32602, resp.Error.Code) // JSON-RPC Invalid Params
}

func TestTokenLoginHandler(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:   "test",
		hub:  hub,
		Send: make(chan []byte, 256),
		tokenParser: func(token string) (*TokenInfo, error) {
			return &TokenInfo{
				UserID:    "user-123",
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			}, nil
		},
	}

	params, _ := json.Marshal(map[string]string{
		"accessToken": "valid-token",
		"platform":    "web",
	})
	c := &Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "auth.loginByToken", Params: params},
		Client:  client,
		Method:  "auth.loginByToken",
	}

	resp := TokenLoginHandler(c)

	assert.Nil(t, resp.Error)
	result := resp.Result.(map[string]any)
	assert.Equal(t, "user-123", result["userId"])
	assert.NotZero(t, result["expiresAt"])

	// Verify login info was set
	loginInfo := c.LoginInfo()
	assert.NotNil(t, loginInfo)
	assert.Equal(t, "user-123", loginInfo.TokenInfo.UserID)
	assert.Equal(t, "web", loginInfo.Platform)
}

func TestTokenLoginHandler_MissingToken(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:  "test",
		hub: hub,
	}

	params, _ := json.Marshal(map[string]string{
		"platform": "web",
	})
	c := &Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "auth.loginByToken", Params: params},
		Client:  client,
		Method:  "auth.loginByToken",
	}

	resp := TokenLoginHandler(c)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32602, resp.Error.Code) // JSON-RPC Invalid Params
}

func TestTokenLoginHandler_InvalidPlatform(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:  "test",
		hub: hub,
	}

	params, _ := json.Marshal(map[string]string{
		"accessToken": "valid-token",
		"platform":    "invalid",
	})
	c := &Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "auth.loginByToken", Params: params},
		Client:  client,
		Method:  "auth.loginByToken",
	}

	resp := TokenLoginHandler(c)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32602, resp.Error.Code) // JSON-RPC Invalid Params
}

func TestTokenLoginHandler_InvalidToken(t *testing.T) {
	hub := NewHub()
	client := &Client{
		ID:  "test",
		hub: hub,
		tokenParser: func(token string) (*TokenInfo, error) {
			return nil, NewError(401, "InvalidToken", "token expired")
		},
	}

	params, _ := json.Marshal(map[string]string{
		"accessToken": "invalid-token",
		"platform":    "web",
	})
	c := &Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "auth.loginByToken", Params: params},
		Client:  client,
		Method:  "auth.loginByToken",
	}

	resp := TokenLoginHandler(c)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32001, resp.Error.Code) // JSON-RPC Unauthenticated
}
