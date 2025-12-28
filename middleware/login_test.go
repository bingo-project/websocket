// ABOUTME: Tests for login state updater middleware.
// ABOUTME: Verifies client state is updated after successful login.

package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bingo-project/websocket"
	"github.com/bingo-project/websocket/jsonrpc"
)

func TestLoginStateUpdater_WithLoginInfo(t *testing.T) {
	hub := websocket.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	client := websocket.NewClient(hub, nil, context.Background())

	handler := func(c *websocket.Context) *jsonrpc.Response {
		// Simulate handler setting login info
		c.SetLoginInfo(&websocket.TokenInfo{
			UserID:    "user-123",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}, "web")

		return jsonrpc.NewResponse(c.Request.ID, map[string]any{
			"accessToken": "token",
		})
	}

	wrapped := LoginStateUpdater(handler)

	c := &websocket.Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "auth.login"},
		Client:  client,
		Method:  "auth.login",
	}

	resp := wrapped(c)

	assert.Nil(t, resp.Error)

	// Wait for hub to process login event
	time.Sleep(50 * time.Millisecond)

	// Verify client state was updated
	assert.True(t, client.IsAuthenticated())
	assert.Equal(t, "user-123", client.GetUserID())
	assert.Equal(t, "web", client.GetPlatform())
}

func TestLoginStateUpdater_WithoutLoginInfo(t *testing.T) {
	hub := websocket.NewHub()
	client := websocket.NewClient(hub, nil, context.Background())

	handler := func(c *websocket.Context) *jsonrpc.Response {
		// Handler does not set login info
		return jsonrpc.NewResponse(c.Request.ID, map[string]any{
			"message": "ok",
		})
	}

	wrapped := LoginStateUpdater(handler)

	c := &websocket.Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "test"},
		Client:  client,
		Method:  "test",
	}

	resp := wrapped(c)

	assert.Nil(t, resp.Error)

	// Client should remain unauthenticated
	assert.False(t, client.IsAuthenticated())
}

func TestLoginStateUpdater_HandlerError(t *testing.T) {
	hub := websocket.NewHub()
	client := websocket.NewClient(hub, nil, context.Background())

	handler := func(c *websocket.Context) *jsonrpc.Response {
		// Handler returns error
		return jsonrpc.NewErrorResponse(c.Request.ID,
			websocket.NewError(401, "InvalidCredentials", "wrong password"))
	}

	wrapped := LoginStateUpdater(handler)

	c := &websocket.Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "auth.login"},
		Client:  client,
		Method:  "auth.login",
	}

	resp := wrapped(c)

	assert.NotNil(t, resp.Error)

	// Client should remain unauthenticated
	assert.False(t, client.IsAuthenticated())
}

func TestLoginStateUpdater_NilClient(t *testing.T) {
	handler := func(c *websocket.Context) *jsonrpc.Response {
		c.SetLoginInfo(&websocket.TokenInfo{
			UserID:    "user-123",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}, "web")

		return jsonrpc.NewResponse(c.Request.ID, "ok")
	}

	wrapped := LoginStateUpdater(handler)

	c := &websocket.Context{
		Context: context.Background(),
		Request: &jsonrpc.Request{ID: 1, Method: "auth.login"},
		Client:  nil,
		Method:  "auth.login",
	}

	resp := wrapped(c)

	// Should not panic and just return the response
	assert.Nil(t, resp.Error)
}
