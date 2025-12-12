// ABOUTME: Request ID middleware for WebSocket handlers.
// ABOUTME: Uses client-provided ID or generates UUID.

package middleware

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/bingo-project/websocket"
	"github.com/bingo-project/websocket/jsonrpc"
)

// RequestID adds request ID to context.
// Uses client-provided ID if present, otherwise generates UUID.
func RequestID(next websocket.Handler) websocket.Handler {
	return func(c *websocket.Context) *jsonrpc.Response {
		requestID := ""
		if c.Request.ID != nil {
			requestID = fmt.Sprintf("%v", c.Request.ID)
		}
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Context = websocket.WithRequestID(c.Context, requestID)

		return next(c)
	}
}
