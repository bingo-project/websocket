// ABOUTME: Authentication middleware for WebSocket handlers.
// ABOUTME: Blocks unauthenticated requests with 401 error.

package middleware

import (
	"github.com/bingo-project/websocket"
	"github.com/bingo-project/websocket/jsonrpc"
)

// Auth requires the client to be authenticated.
func Auth(next websocket.Handler) websocket.Handler {
	return func(c *websocket.Context) *jsonrpc.Response {
		if c.Client == nil || !c.Client.IsAuthenticated() {
			return jsonrpc.NewErrorResponse(c.Request.ID,
				websocket.NewError(401, "Unauthorized", "Login required"))
		}

		// Add user ID to context
		c.Context = websocket.WithUserID(c.Context, c.Client.GetUserID())

		return next(c)
	}
}
