// ABOUTME: Built-in handlers for common WebSocket methods.
// ABOUTME: Provides heartbeat, subscribe, unsubscribe, and token login handlers.

package websocket

import (
	"time"

	"github.com/bingo-project/websocket/jsonrpc"
)

// HeartbeatHandler responds to heartbeat requests.
func HeartbeatHandler(c *Context) *jsonrpc.Response {
	return jsonrpc.NewResponse(c.Request.ID, map[string]any{
		"status":      "ok",
		"server_time": time.Now().Unix(),
	})
}

// SubscribeHandler handles topic subscription.
func SubscribeHandler(c *Context) *jsonrpc.Response {
	var params struct {
		Topics []string `json:"topics"`
	}

	if err := c.BindParams(&params); err != nil {
		return jsonrpc.NewErrorResponse(c.Request.ID,
			NewError(400, "InvalidParams", "Invalid subscribe params"))
	}

	if len(params.Topics) == 0 {
		return jsonrpc.NewErrorResponse(c.Request.ID,
			NewError(400, "InvalidParams", "Topics required"))
	}

	result := make(chan []string, 1)
	c.Client.hub.Subscribe <- &SubscribeEvent{
		Client: c.Client,
		Topics: params.Topics,
		Result: result,
	}

	subscribed := <-result

	return jsonrpc.NewResponse(c.Request.ID, map[string]any{
		"subscribed": subscribed,
	})
}

// UnsubscribeHandler handles topic unsubscription.
func UnsubscribeHandler(c *Context) *jsonrpc.Response {
	var params struct {
		Topics []string `json:"topics"`
	}

	if err := c.BindParams(&params); err != nil {
		return jsonrpc.NewErrorResponse(c.Request.ID,
			NewError(400, "InvalidParams", "Invalid unsubscribe params"))
	}

	if len(params.Topics) == 0 {
		return jsonrpc.NewErrorResponse(c.Request.ID,
			NewError(400, "InvalidParams", "Topics required"))
	}

	c.Client.hub.Unsubscribe <- &UnsubscribeEvent{
		Client: c.Client,
		Topics: params.Topics,
	}

	return jsonrpc.NewResponse(c.Request.ID, map[string]any{
		"unsubscribed": params.Topics,
	})
}

// TokenLoginHandler handles token-based authentication.
// It validates the provided access token and updates client state.
// Use with LoginStateUpdater middleware to complete the login process.
func TokenLoginHandler(c *Context) *jsonrpc.Response {
	var params struct {
		AccessToken string `json:"accessToken"`
		Platform    string `json:"platform"`
	}

	if err := c.BindParams(&params); err != nil {
		return jsonrpc.NewErrorResponse(c.Request.ID,
			NewError(400, "InvalidParams", "Invalid login params"))
	}

	if params.AccessToken == "" {
		return jsonrpc.NewErrorResponse(c.Request.ID,
			NewError(400, "InvalidParams", "accessToken required"))
	}

	if !IsValidPlatform(params.Platform) {
		return jsonrpc.NewErrorResponse(c.Request.ID,
			NewError(400, "InvalidPlatform", "Invalid platform: %s", params.Platform))
	}

	// Parse token using client's token parser
	tokenInfo, err := c.Client.ParseToken(params.AccessToken)
	if err != nil {
		return jsonrpc.NewErrorResponse(c.Request.ID,
			NewError(401, "InvalidToken", "Token validation failed"))
	}

	// Set login info for middleware to process
	c.SetLoginInfo(tokenInfo, params.Platform)

	return jsonrpc.NewResponse(c.Request.ID, map[string]any{
		"userId":    tokenInfo.UserID,
		"expiresAt": tokenInfo.ExpiresAt,
	})
}
