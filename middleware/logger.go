// ABOUTME: Request logging middleware for WebSocket handlers.
// ABOUTME: Logs method, latency, and error status.

package middleware

import (
	"time"

	"github.com/bingo-project/websocket"
	"github.com/bingo-project/websocket/jsonrpc"
)

// LoggerWithLogger logs request details after handling using the provided logger.
func LoggerWithLogger(logger websocket.Logger) websocket.Middleware {
	return func(next websocket.Handler) websocket.Handler {
		return func(c *websocket.Context) *jsonrpc.Response {
			resp := next(c)

			fields := []any{
				"method", c.Method,
				"latency", time.Since(c.StartTime),
			}

			if c.Client != nil {
				fields = append(fields, "client_id", c.Client.ID, "client_addr", c.Client.Addr)
				if c.Client.UserID != "" {
					fields = append(fields, "user_id", c.Client.UserID)
				}
			}

			if resp.Error != nil {
				fields = append(fields, "error", resp.Error.Reason)
				logger.WithContext(c.Context).Warnw("WebSocket request failed", fields...)
			} else {
				logger.WithContext(c.Context).Infow("WebSocket request", fields...)
			}

			return resp
		}
	}
}

// Logger logs request details after handling.
// Uses NopLogger by default; use LoggerWithLogger to inject a real logger.
var Logger = LoggerWithLogger(websocket.NopLogger())
