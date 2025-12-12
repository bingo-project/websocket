// ABOUTME: Configuration for WebSocket Hub.
// ABOUTME: Defines timeout and cleanup intervals.

package websocket

import "time"

// HubConfig holds configuration for the Hub.
type HubConfig struct {
	// Anonymous connection timeout (must login within this time)
	AnonymousTimeout time.Duration
	// Anonymous connection cleanup interval
	AnonymousCleanup time.Duration

	// Authenticated connection heartbeat timeout
	HeartbeatTimeout time.Duration
	// Authenticated connection cleanup interval
	HeartbeatCleanup time.Duration

	// WebSocket protocol ping period
	PingPeriod time.Duration
	// WebSocket protocol pong wait timeout
	PongWait time.Duration

	// Client connection settings
	MaxMessageSize int64         // Maximum message size allowed from peer
	WriteWait      time.Duration // Time allowed to write a message to the peer

	// Connection limits (0 = unlimited)
	MaxConnections  int // Maximum total connections
	MaxConnsPerUser int // Maximum connections per user (across all platforms)
}

// DefaultHubConfig returns default configuration.
func DefaultHubConfig() *HubConfig {
	return &HubConfig{
		AnonymousTimeout: 10 * time.Second,
		AnonymousCleanup: 2 * time.Second,
		HeartbeatTimeout: 60 * time.Second,
		HeartbeatCleanup: 30 * time.Second,
		PingPeriod:       54 * time.Second,
		PongWait:         60 * time.Second,
		MaxMessageSize:   4096,
		WriteWait:        10 * time.Second,
		MaxConnections:   0, // unlimited
		MaxConnsPerUser:  0, // unlimited
	}
}

// Validate checks if the config values are valid.
func (c *HubConfig) Validate() error {
	if c.AnonymousTimeout <= 0 {
		return NewError(400, "ConfigError", "AnonymousTimeout must be positive")
	}
	if c.AnonymousCleanup <= 0 {
		return NewError(400, "ConfigError", "AnonymousCleanup must be positive")
	}
	if c.HeartbeatTimeout <= 0 {
		return NewError(400, "ConfigError", "HeartbeatTimeout must be positive")
	}
	if c.HeartbeatCleanup <= 0 {
		return NewError(400, "ConfigError", "HeartbeatCleanup must be positive")
	}
	if c.PingPeriod <= 0 {
		return NewError(400, "ConfigError", "PingPeriod must be positive")
	}
	if c.PongWait <= 0 {
		return NewError(400, "ConfigError", "PongWait must be positive")
	}
	if c.PingPeriod >= c.PongWait {
		return NewError(400, "ConfigError", "PingPeriod must be less than PongWait")
	}
	if c.MaxMessageSize <= 0 {
		return NewError(400, "ConfigError", "MaxMessageSize must be positive")
	}
	if c.WriteWait <= 0 {
		return NewError(400, "ConfigError", "WriteWait must be positive")
	}
	if c.MaxConnections < 0 {
		return NewError(400, "ConfigError", "MaxConnections cannot be negative")
	}
	if c.MaxConnsPerUser < 0 {
		return NewError(400, "ConfigError", "MaxConnsPerUser cannot be negative")
	}

	return nil
}
