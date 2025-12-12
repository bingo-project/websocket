// ABOUTME: Tests for error types and utilities.
// ABOUTME: Validates error creation, formatting, and HTTP-to-JSON-RPC code mapping.

package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	err := New(400, "InvalidParams", "missing field: %s", "name")

	assert.Equal(t, 400, err.Code)
	assert.Equal(t, "InvalidParams", err.Reason)
	assert.Equal(t, "missing field: name", err.Message)
}

func TestError_Error(t *testing.T) {
	err := New(401, "Unauthorized", "invalid token")

	assert.Equal(t, "error: code = 401 reason = Unauthorized message = invalid token", err.Error())
}

func TestError_WithMessage(t *testing.T) {
	err := New(400, "BadRequest", "original message")
	err = err.WithMessage("new message: %d", 123)

	assert.Equal(t, "new message: 123", err.Message)
}

func TestError_WithMetadata(t *testing.T) {
	err := New(400, "BadRequest", "error")
	md := map[string]string{"field": "name", "reason": "required"}
	err = err.WithMetadata(md)

	assert.Equal(t, md, err.Metadata)
}

func TestError_JSONRPCCode(t *testing.T) {
	tests := []struct {
		httpCode     int
		expectedRPC  int
	}{
		{400, -32602}, // Invalid params
		{401, -32001}, // Unauthenticated
		{403, -32003}, // Permission denied
		{404, -32004}, // Not found
		{409, -32009}, // Conflict
		{429, -32029}, // Too many requests
		{500, -32603}, // Internal error
		{503, -32053}, // Service unavailable
		{418, -32603}, // Unknown -> default to Internal error
	}

	for _, tt := range tests {
		err := New(tt.httpCode, "Test", "test error")
		assert.Equal(t, tt.expectedRPC, err.JSONRPCCode(), "HTTP code %d", tt.httpCode)
	}
}

func TestFromError_Nil(t *testing.T) {
	result := FromError(nil)
	assert.Nil(t, result)
}

func TestFromError_AlreadyError(t *testing.T) {
	original := New(403, "Forbidden", "access denied")
	result := FromError(original)

	assert.Same(t, original, result)
}

func TestFromError_StandardError(t *testing.T) {
	stdErr := errors.New("something went wrong")
	result := FromError(stdErr)

	assert.Equal(t, 500, result.Code)
	assert.Equal(t, "InternalError", result.Reason)
	assert.Equal(t, "something went wrong", result.Message)
}

func TestFromError_WrappedError(t *testing.T) {
	original := New(404, "NotFound", "user not found")
	wrapped := errors.New("wrapped: " + original.Error())

	// This won't unwrap because we used string concatenation
	result := FromError(wrapped)
	assert.Equal(t, 500, result.Code) // Falls back to internal error

	// Now test with proper wrapping using fmt.Errorf
	import_wrapped := &wrappedError{err: original}
	result2 := FromError(import_wrapped)
	assert.Same(t, original, result2)
}

// wrappedError implements Unwrap for testing
type wrappedError struct {
	err error
}

func (w *wrappedError) Error() string {
	return "wrapped: " + w.err.Error()
}

func (w *wrappedError) Unwrap() error {
	return w.err
}
