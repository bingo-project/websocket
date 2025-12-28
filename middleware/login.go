// ABOUTME: Login middleware for WebSocket handlers.
// ABOUTME: Updates client authentication state after successful login.

package middleware

import (
	"github.com/bingo-project/websocket"
	"github.com/bingo-project/websocket/jsonrpc"
)

// LoginStateUpdater updates client state after successful login.
// It reads login info from context (set by handler via SetLoginInfo) and
// updates client state and notifies the hub.
func LoginStateUpdater(next websocket.Handler) websocket.Handler {
	return func(c *websocket.Context) *jsonrpc.Response {
		// Call next handler
		resp := next(c)

		// Only process successful responses
		if resp.Error != nil || c.Client == nil {
			return resp
		}

		// Read login info from context (set by handler)
		loginInfo := c.LoginInfo()
		if loginInfo == nil || loginInfo.TokenInfo == nil {
			return resp
		}

		// Update client context
		c.Client.UpdateContext(loginInfo.TokenInfo.UserID)

		// Notify hub about login
		c.Client.NotifyLogin(loginInfo.TokenInfo.UserID, loginInfo.Platform, loginInfo.TokenInfo.ExpiresAt)

		return resp
	}
}
