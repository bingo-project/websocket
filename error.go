// ABOUTME: Re-exports error types from the errors subpackage.
// ABOUTME: Provides convenient access to Error, NewError, and FromError at the package level.

package websocket

import "github.com/bingo-project/websocket/errors"

// Error is an alias for errors.Error.
type Error = errors.Error

// NewError creates a new Error with the given code, reason, and formatted message.
var NewError = errors.New

// FromError converts an error to *Error.
var FromError = errors.FromError
