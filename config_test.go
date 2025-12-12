// ABOUTME: Tests for Hub configuration.
// ABOUTME: Validates config validation and defaults.

package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultHubConfig(t *testing.T) {
	cfg := DefaultHubConfig()

	assert.Equal(t, 10*time.Second, cfg.AnonymousTimeout)
	assert.Equal(t, 2*time.Second, cfg.AnonymousCleanup)
	assert.Equal(t, 60*time.Second, cfg.HeartbeatTimeout)
	assert.Equal(t, 30*time.Second, cfg.HeartbeatCleanup)
	assert.Equal(t, 54*time.Second, cfg.PingPeriod)
	assert.Equal(t, 60*time.Second, cfg.PongWait)
	assert.Equal(t, int64(4096), cfg.MaxMessageSize)
	assert.Equal(t, 10*time.Second, cfg.WriteWait)
	assert.Equal(t, 0, cfg.MaxConnections)
	assert.Equal(t, 0, cfg.MaxConnsPerUser)
}

func TestHubConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*HubConfig)
		wantErr bool
	}{
		{
			name:    "default config is valid",
			modify:  func(c *HubConfig) {},
			wantErr: false,
		},
		{
			name:    "zero AnonymousTimeout",
			modify:  func(c *HubConfig) { c.AnonymousTimeout = 0 },
			wantErr: true,
		},
		{
			name:    "negative AnonymousTimeout",
			modify:  func(c *HubConfig) { c.AnonymousTimeout = -1 },
			wantErr: true,
		},
		{
			name:    "zero HeartbeatTimeout",
			modify:  func(c *HubConfig) { c.HeartbeatTimeout = 0 },
			wantErr: true,
		},
		{
			name:    "PingPeriod >= PongWait",
			modify:  func(c *HubConfig) { c.PingPeriod = c.PongWait },
			wantErr: true,
		},
		{
			name:    "zero MaxMessageSize",
			modify:  func(c *HubConfig) { c.MaxMessageSize = 0 },
			wantErr: true,
		},
		{
			name:    "negative MaxConnections",
			modify:  func(c *HubConfig) { c.MaxConnections = -1 },
			wantErr: true,
		},
		{
			name:    "zero MaxConnections is valid (unlimited)",
			modify:  func(c *HubConfig) { c.MaxConnections = 0 },
			wantErr: false,
		},
		{
			name:    "positive MaxConnections is valid",
			modify:  func(c *HubConfig) { c.MaxConnections = 100 },
			wantErr: false,
		},
		{
			name:    "negative MaxConnsPerUser",
			modify:  func(c *HubConfig) { c.MaxConnsPerUser = -1 },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultHubConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
