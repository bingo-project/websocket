// ABOUTME: Basic example demonstrating websocket framework usage.
// ABOUTME: Shows hub setup, routing, middleware, and message handling.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bingo-project/websocket"
	"github.com/bingo-project/websocket/errors"
	"github.com/bingo-project/websocket/jsonrpc"
	"github.com/bingo-project/websocket/middleware"
	gorillaWS "github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var hub *websocket.Hub

func main() {
	// Create hub config
	cfg := &websocket.HubConfig{
		AnonymousTimeout: 30 * time.Second,
		AnonymousCleanup: 5 * time.Second,
		HeartbeatTimeout: 60 * time.Second,
		HeartbeatCleanup: 30 * time.Second,
		PingPeriod:       54 * time.Second,
		PongWait:         60 * time.Second,
		MaxMessageSize:   4096,
		WriteWait:        10 * time.Second,
		MaxConnections:   1000,
		MaxConnsPerUser:  3,
	}

	if err := cfg.Validate(); err != nil {
		log.Fatal("Invalid config:", err)
	}

	// Create metrics
	metrics := websocket.NewMetrics("example", "websocket")
	metrics.MustRegister(prometheus.DefaultRegisterer)

	// Create rate limiter store
	rateLimiterStore := middleware.NewRateLimiterStore()

	// Create hub with options
	hub = websocket.NewHubWithConfig(cfg,
		websocket.WithMetrics(metrics),
		websocket.WithClientDisconnectCallback(rateLimiterStore.Remove),
	)

	// Create router
	router := websocket.NewRouter()

	// Global middleware
	router.Use(
		middleware.Recovery,
		middleware.RequestID,
		middleware.Logger,
		middleware.RateLimitWithStore(&middleware.RateLimitConfig{
			Default: 10,
			Methods: map[string]float64{
				"heartbeat": 0, // No limit for heartbeat
			},
		}, rateLimiterStore),
	)

	// Public methods (no auth required)
	public := router.Group()
	public.Handle("heartbeat", websocket.HeartbeatHandler)
	public.Handle("echo", echoHandler)
	public.Handle("login", loginHandler)

	// Private methods (require auth)
	private := router.Group(middleware.Auth)
	private.Handle("subscribe", websocket.SubscribeHandler)
	private.Handle("unsubscribe", websocket.UnsubscribeHandler)
	private.Handle("whoami", whoamiHandler)

	// Start hub
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// WebSocket upgrader
	upgrader := gorillaWS.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// WebSocket handler
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Check connection limit before upgrading
		if !hub.CanAcceptConnection() {
			http.Error(w, "Too many connections", http.StatusServiceUnavailable)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Upgrade error: %v", err)
			return
		}

		client := websocket.NewClient(hub, conn, context.Background(),
			websocket.WithRouter(router),
		)
		hub.Register <- client

		go client.WritePump()
		go client.ReadPump()
	})

	// Metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		stats := hub.Stats()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"status":"ok","connections":%d}`, stats.TotalConnections)))
	})

	// Start server
	server := &http.Server{Addr: ":8080"}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down...")
		cancel() // Stop hub

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Println("Server starting on :8080")
	log.Println("WebSocket: ws://localhost:8080/ws")
	log.Println("Metrics: http://localhost:8080/metrics")

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// echoHandler returns the received params as-is
func echoHandler(c *websocket.Context) *jsonrpc.Response {
	return c.JSON(c.Request.Params)
}

// loginHandler simulates user login
func loginHandler(c *websocket.Context) *jsonrpc.Response {
	var req struct {
		Username string `json:"username" validate:"required"`
		Platform string `json:"platform" validate:"required,oneof=web ios android"`
	}
	if err := c.BindValidate(&req); err != nil {
		return c.Error(errors.New(400, "InvalidParams", "%s", err.Error()))
	}

	// In real app, verify credentials and generate token here
	userID := "user_" + req.Username
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	// Check per-user connection limit
	if !hub.CanUserConnect(userID) {
		return c.Error(errors.New(429, "TooManyConnections", "Too many connections for this user"))
	}

	// Update client state
	c.Client.NotifyLogin(userID, req.Platform, expiresAt)

	return c.JSON(map[string]any{
		"user_id":    userID,
		"expires_at": expiresAt,
	})
}

// whoamiHandler returns current user info
func whoamiHandler(c *websocket.Context) *jsonrpc.Response {
	return c.JSON(map[string]any{
		"user_id":  c.Client.UserID,
		"platform": c.Client.Platform,
	})
}
