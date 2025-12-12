// ABOUTME: Panic recovery middleware for WebSocket handlers.
// ABOUTME: Catches panics and returns a JSON-RPC error response.

package middleware

import (
	"runtime/debug"

	"github.com/bingo-project/websocket"
	"github.com/bingo-project/websocket/jsonrpc"
)

// RecoveryWithLogger catches panics and returns an error response using the provided logger.
func RecoveryWithLogger(logger websocket.Logger) websocket.Middleware {
	return func(next websocket.Handler) websocket.Handler {
		return func(c *websocket.Context) (resp *jsonrpc.Response) {
			defer func() {
				if r := recover(); r != nil {
					logger.WithContext(c.Context).Errorw("WebSocket panic recovered",
						"method", c.Method,
						"panic", r,
						"stack", string(debug.Stack()),
					)
					resp = jsonrpc.NewErrorResponse(c.Request.ID,
						websocket.NewError(500, "InternalError", "Internal server error"))
				}
			}()

			return next(c)
		}
	}
}

// Recovery catches panics and returns an error response.
// Uses NopLogger by default; use RecoveryWithLogger to inject a real logger.
var Recovery = RecoveryWithLogger(websocket.NopLogger())
