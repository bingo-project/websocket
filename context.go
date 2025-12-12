// ABOUTME: Context utilities for storing and retrieving request-scoped values.
// ABOUTME: Provides type-safe context keys for request ID and user ID.

package websocket

import (
	"context"
)

type (
	requestIDKey struct{}
	userIDKey    struct{}
)

// WithRequestID stores the request ID in the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestID retrieves the request ID from the context.
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

// WithUserID stores the user ID in the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserID retrieves the user ID from the context.
func UserID(ctx context.Context) string {
	userID, _ := ctx.Value(userIDKey{}).(string)
	return userID
}
