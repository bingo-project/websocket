// ABOUTME: Lightweight error type for WebSocket/JSON-RPC responses.
// ABOUTME: Provides Error struct with Code, Reason, Message, and HTTP-to-JSON-RPC code mapping.

package errors

import (
	"errors"
	"fmt"
)

// Error represents an error with HTTP-style code, reason, and message.
type Error struct {
	// Code is the HTTP-style status code (e.g., 400, 401, 500).
	Code int `json:"code,omitempty"`

	// Reason is a short error identifier (e.g., "InvalidParams", "Unauthorized").
	Reason string `json:"reason,omitempty"`

	// Message is a human-readable error description.
	Message string `json:"message,omitempty"`

	// Metadata holds additional key-value data about the error.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// httpToJSONRPC maps HTTP status codes to JSON-RPC error codes.
var httpToJSONRPC = map[int]int{
	400: -32602, // Invalid params
	401: -32001, // Unauthenticated
	403: -32003, // Permission denied
	404: -32004, // Not found
	409: -32009, // Conflict
	429: -32029, // Too many requests
	500: -32603, // Internal error
	503: -32053, // Service unavailable
}

// New creates a new Error with the given code, reason, and formatted message.
func New(code int, reason string, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Reason:  reason,
		Message: fmt.Sprintf(format, args...),
	}
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("error: code = %d reason = %s message = %s", e.Code, e.Reason, e.Message)
}

// WithMessage sets the error message.
func (e *Error) WithMessage(format string, args ...any) *Error {
	e.Message = fmt.Sprintf(format, args...)
	return e
}

// WithMetadata sets the error metadata.
func (e *Error) WithMetadata(md map[string]string) *Error {
	e.Metadata = md
	return e
}

// JSONRPCCode returns the JSON-RPC error code for this error.
func (e *Error) JSONRPCCode() int {
	if code, ok := httpToJSONRPC[e.Code]; ok {
		return code
	}
	return -32603 // Default to Internal error
}

// FromError converts an error to *Error.
// If the error is already an *Error, it returns it directly.
// Otherwise, it wraps it as an internal error.
func FromError(err error) *Error {
	if err == nil {
		return nil
	}

	var e *Error
	if errors.As(err, &e) {
		return e
	}

	return New(500, "InternalError", "%s", err.Error())
}
