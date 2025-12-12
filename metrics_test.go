// ABOUTME: Tests for Prometheus metrics.
// ABOUTME: Validates metric registration and collection.

package websocket_test

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	"github.com/bingo-project/websocket"
)

func TestMetrics_Registration(t *testing.T) {
	metrics := websocket.NewMetrics("test", "hub")
	reg := prometheus.NewRegistry()

	err := metrics.Register(reg)
	assert.NoError(t, err)

	// Verify metrics are registered
	families, err := reg.Gather()
	assert.NoError(t, err)
	assert.NotEmpty(t, families)
}

func TestHub_WithMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metrics := websocket.NewMetrics("test", "hub")
	hub := websocket.NewHub(websocket.WithMetrics(metrics))
	go hub.Run(ctx)

	// Verify metrics are attached
	assert.NotNil(t, hub.Metrics())
	assert.Equal(t, metrics, hub.Metrics())
}

func TestMetrics_ConnectionTracking(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metrics := websocket.NewMetrics("test", "hub")
	reg := prometheus.NewRegistry()
	metrics.MustRegister(reg)

	hub := websocket.NewHub(websocket.WithMetrics(metrics))
	go hub.Run(ctx)

	// Register a client
	client := &websocket.Client{
		ID:   "test-client",
		Addr: "127.0.0.1",
		Send: make(chan []byte, 10),
	}
	hub.Register <- client
	time.Sleep(20 * time.Millisecond)

	// Check metrics
	families, _ := reg.Gather()
	var connTotal, connCurrent, anonConns float64
	for _, f := range families {
		switch f.GetName() {
		case "test_hub_connections_total":
			connTotal = f.GetMetric()[0].GetCounter().GetValue()
		case "test_hub_connections_current":
			connCurrent = f.GetMetric()[0].GetGauge().GetValue()
		case "test_hub_connections_anonymous":
			anonConns = f.GetMetric()[0].GetGauge().GetValue()
		}
	}

	assert.Equal(t, float64(1), connTotal)
	assert.Equal(t, float64(1), connCurrent)
	assert.Equal(t, float64(1), anonConns)

	// Unregister
	hub.Unregister <- client
	time.Sleep(20 * time.Millisecond)

	families, _ = reg.Gather()
	for _, f := range families {
		if f.GetName() == "test_hub_connections_current" {
			connCurrent = f.GetMetric()[0].GetGauge().GetValue()
		}
	}
	assert.Equal(t, float64(0), connCurrent)
}
