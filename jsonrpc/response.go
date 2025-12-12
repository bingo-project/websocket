// ABOUTME: JSON-RPC 2.0 response and push message constructors.
// ABOUTME: Creates success responses, error responses, stream responses, and push messages.

package jsonrpc

import "github.com/bingo-project/websocket/errors"

// NewResponse creates a success response.
func NewResponse(id any, result any) *Response {
	return &Response{
		JSONRPC: Version,
		Result:  result,
		ID:      id,
	}
}

// NewErrorResponse creates an error response from an error.
func NewErrorResponse(id any, err error) *Response {
	e := errors.FromError(err)

	return &Response{
		JSONRPC: Version,
		Error: &Error{
			Code:    e.JSONRPCCode(),
			Reason:  e.Reason,
			Message: e.Message,
			Data:    e.Metadata,
		},
		ID: id,
	}
}

// NewStreamResponse creates a response for streaming scenarios with method identifier.
func NewStreamResponse(id any, method string, result any) *Response {
	return &Response{
		JSONRPC: Version,
		Method:  method,
		Result:  result,
		ID:      id,
	}
}

// NewPush creates a server-initiated push message (not tied to any request).
func NewPush(method string, data any) *Push {
	return &Push{
		JSONRPC: Version,
		Method:  method,
		Data:    data,
	}
}
