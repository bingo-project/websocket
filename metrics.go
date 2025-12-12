// ABOUTME: Prometheus metrics for WebSocket hub monitoring.
// ABOUTME: Tracks connections, messages, and errors.

package websocket

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all Prometheus metrics for the WebSocket hub.
type Metrics struct {
	// Connection metrics
	ConnectionsTotal   prometheus.Counter
	ConnectionsCurrent prometheus.Gauge
	AuthenticatedConns prometheus.Gauge
	AnonymousConns     prometheus.Gauge

	// Message metrics
	MessagesReceived prometheus.Counter
	MessagesSent     prometheus.Counter
	BroadcastsSent   prometheus.Counter

	// Error metrics
	ErrorsTotal *prometheus.CounterVec

	// Subscription metrics
	TopicsTotal        prometheus.Gauge
	SubscriptionsTotal prometheus.Counter
}

// NewMetrics creates a new Metrics instance with default metric definitions.
func NewMetrics(namespace, subsystem string) *Metrics {
	if namespace == "" {
		namespace = "websocket"
	}
	if subsystem == "" {
		subsystem = "hub"
	}

	return &Metrics{
		ConnectionsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "connections_total",
			Help:      "Total number of WebSocket connections established",
		}),
		ConnectionsCurrent: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "connections_current",
			Help:      "Current number of active WebSocket connections",
		}),
		AuthenticatedConns: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "connections_authenticated",
			Help:      "Current number of authenticated connections",
		}),
		AnonymousConns: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "connections_anonymous",
			Help:      "Current number of anonymous connections",
		}),
		MessagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_received_total",
			Help:      "Total number of messages received from clients",
		}),
		MessagesSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_sent_total",
			Help:      "Total number of messages sent to clients",
		}),
		BroadcastsSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "broadcasts_total",
			Help:      "Total number of broadcast messages sent",
		}),
		ErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "errors_total",
			Help:      "Total number of errors by type",
		}, []string{"type"}),
		TopicsTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "topics_current",
			Help:      "Current number of active topics",
		}),
		SubscriptionsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "subscriptions_total",
			Help:      "Total number of topic subscriptions",
		}),
	}
}

// Register registers all metrics with the given registerer.
func (m *Metrics) Register(reg prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.ConnectionsTotal,
		m.ConnectionsCurrent,
		m.AuthenticatedConns,
		m.AnonymousConns,
		m.MessagesReceived,
		m.MessagesSent,
		m.BroadcastsSent,
		m.ErrorsTotal,
		m.TopicsTotal,
		m.SubscriptionsTotal,
	}

	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			return err
		}
	}

	return nil
}

// MustRegister registers all metrics and panics on error.
func (m *Metrics) MustRegister(reg prometheus.Registerer) {
	if err := m.Register(reg); err != nil {
		panic(err)
	}
}

// Collectors returns all metric collectors for custom registration.
func (m *Metrics) Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.ConnectionsTotal,
		m.ConnectionsCurrent,
		m.AuthenticatedConns,
		m.AnonymousConns,
		m.MessagesReceived,
		m.MessagesSent,
		m.BroadcastsSent,
		m.ErrorsTotal,
		m.TopicsTotal,
		m.SubscriptionsTotal,
	}
}
