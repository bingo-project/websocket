// ABOUTME: Middleware types for WebSocket message handling.
// ABOUTME: Provides middleware chain composition similar to HTTP/gRPC patterns.

package websocket

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/bingo-project/websocket/jsonrpc"
)

// validate is the singleton validator instance.
var validate = validator.New(validator.WithRequiredStructEnabled())

func init() {
	validate.SetTagName("binding")
}

// LoginInfo contains login authentication information.
// Used to pass login state from handler to middleware.
type LoginInfo struct {
	TokenInfo *TokenInfo
	Platform  string
}

// Context contains all information needed by middleware.
// It embeds context.Context so it can be passed directly to business layer methods.
type Context struct {
	context.Context
	Request   *jsonrpc.Request
	Client    *Client
	Method    string
	StartTime time.Time

	// loginInfo stores login state set by handler for middleware to process.
	loginInfo *LoginInfo
}

// RequestID returns the request ID from context.
func (c *Context) RequestID() string {
	if c.Context == nil {
		return ""
	}

	return RequestID(c.Context)
}

// UserID returns the user ID from context.
func (c *Context) UserID() string {
	if c.Context == nil {
		return ""
	}

	return UserID(c.Context)
}

// BindParams unmarshals the request params into the given struct.
func (c *Context) BindParams(v any) error {
	if len(c.Request.Params) == 0 {
		return nil
	}

	return json.Unmarshal(c.Request.Params, v)
}

// BindValidate unmarshals and validates the request params.
func (c *Context) BindValidate(v any) error {
	if err := c.BindParams(v); err != nil {
		return err
	}

	return validate.Struct(v)
}

// JSON returns a successful JSON-RPC response with the given data.
func (c *Context) JSON(data any) *jsonrpc.Response {
	return jsonrpc.NewResponse(c.Request.ID, data)
}

// Error returns a JSON-RPC error response.
func (c *Context) Error(err error) *jsonrpc.Response {
	return jsonrpc.NewErrorResponse(c.Request.ID, err)
}

// SetLoginInfo stores login information for middleware to process.
func (c *Context) SetLoginInfo(tokenInfo *TokenInfo, platform string) {
	c.loginInfo = &LoginInfo{
		TokenInfo: tokenInfo,
		Platform:  platform,
	}
}

// LoginInfo returns the login information set by handler.
func (c *Context) LoginInfo() *LoginInfo {
	return c.loginInfo
}

// Handler is a message handler function.
type Handler func(*Context) *jsonrpc.Response

// Middleware wraps a handler with additional functionality.
type Middleware func(Handler) Handler

// Chain combines multiple middlewares into a single middleware.
func Chain(middlewares ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}

		return next
	}
}
