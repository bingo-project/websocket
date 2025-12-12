# WebSocket

[![Go Reference](https://pkg.go.dev/badge/github.com/bingo-project/websocket.svg)](https://pkg.go.dev/github.com/bingo-project/websocket)
[![Go Report Card](https://goreportcard.com/badge/github.com/bingo-project/websocket)](https://goreportcard.com/report/github.com/bingo-project/websocket)

A production-ready WebSocket framework for Go using JSON-RPC 2.0 protocol with middleware support, grouped routing, and connection management.

[中文文档](README_zh.md)

## Features

- **JSON-RPC 2.0 Protocol** - Industry-standard message format (used by MCP, Ethereum, VSCode LSP)
- **Middleware Pattern** - Familiar programming model like Gin/Echo
- **Grouped Routing** - Support public/private groups with different middleware chains
- **Connection Management** - Hub for client registration, authentication, and topic subscriptions
- **Built-in Handlers** - Heartbeat, subscribe/unsubscribe out of the box
- **Rate Limiting** - Token bucket algorithm with per-method configuration
- **Single Device Login** - Automatic session kick when same user logs in from another device

## Installation

```bash
go get github.com/bingo-project/websocket
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/bingo-project/websocket"
    "github.com/bingo-project/websocket/jsonrpc"
    "github.com/bingo-project/websocket/middleware"
    gorillaWS "github.com/gorilla/websocket"
)

func main() {
    // Create hub and router
    hub := websocket.NewHub()
    router := websocket.NewRouter()

    // Add global middleware
    router.Use(
        middleware.Recovery,
        middleware.RequestID,
        middleware.Logger,
    )

    // Public methods (no auth required)
    public := router.Group()
    public.Handle("heartbeat", websocket.HeartbeatHandler)
    public.Handle("echo", func(c *websocket.Context) *jsonrpc.Response {
        return c.JSON(c.Request.Params)
    })

    // Private methods (require auth)
    private := router.Group(middleware.Auth)
    private.Handle("subscribe", websocket.SubscribeHandler)

    // Start hub
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go hub.Run(ctx)

    // WebSocket upgrader
    upgrader := gorillaWS.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
    }

    // HTTP handler
    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            return
        }

        client := websocket.NewClient(hub, conn, context.Background(),
            websocket.WithRouter(router),
        )
        hub.Register <- client

        go client.WritePump()
        go client.ReadPump()
    })

    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Message Format

### Request

```json
{
    "jsonrpc": "2.0",
    "method": "auth.login",
    "params": {"username": "test", "password": "123456"},
    "id": 1
}
```

### Success Response

```json
{
    "jsonrpc": "2.0",
    "result": {"token": "xxx", "expiresAt": 1234567890},
    "id": 1
}
```

### Error Response

```json
{
    "jsonrpc": "2.0",
    "error": {
        "code": -32001,
        "reason": "Unauthorized",
        "message": "Login required"
    },
    "id": 1
}
```

### Server Push (Notification)

```json
{
    "jsonrpc": "2.0",
    "method": "session.kicked",
    "params": {"reason": "Account logged in elsewhere"}
}
```

## Middleware

### Built-in Middleware

| Middleware | Description |
|------------|-------------|
| `Recovery` / `RecoveryWithLogger` | Catch panics, return 500 error |
| `RequestID` | Inject request-id into context |
| `Logger` / `LoggerWithLogger` | Log requests and latency |
| `Auth` | Verify user is authenticated |
| `RateLimit` / `RateLimitWithStore` | Token bucket rate limiting |
| `LoginStateUpdater` | Update client state after login |

### Custom Middleware

```go
func MyMiddleware(next websocket.Handler) websocket.Handler {
    return func(c *websocket.Context) *jsonrpc.Response {
        // Before handler
        log.Printf("Method: %s", c.Method)

        resp := next(c)

        // After handler
        return resp
    }
}

router.Use(MyMiddleware)
```

## Handler

```go
func Login(c *websocket.Context) *jsonrpc.Response {
    var req LoginRequest
    if err := c.BindValidate(&req); err != nil {
        return c.Error(errors.New(400, "InvalidParams", err.Error()))
    }

    // Business logic...
    token := authenticate(req.Username, req.Password)

    // Update client login state
    c.Client.NotifyLogin(userID, req.Platform, tokenExpiresAt)

    return c.JSON(map[string]any{
        "token":     token,
        "expiresAt": tokenExpiresAt,
    })
}
```

## Connection Management

### Hub API

```go
// Get client by ID
client := hub.GetClient("client-id")

// Get all clients for a user
clients := hub.GetClientsByUser("user-123")

// Kick client
hub.KickClient("client-id", "reason")

// Kick all sessions of a user
hub.KickUser("user-123", "account suspended")

// Get statistics
stats := hub.Stats()
```

### Push Messages

```go
// Push to specific user on specific platform
hub.PushToUser("ios", "user-123", "order.created", data)

// Push to user on all platforms
hub.PushToUserAllPlatforms("user-123", "security.alert", data)

// Push to topic subscribers
hub.PushToTopic("group:123", "message.new", data)

// Broadcast to all authenticated clients
hub.Broadcast <- message
```

### Topic Subscription

```go
// Client subscribes to topics
hub.Subscribe <- &websocket.SubscribeEvent{
    Client: client,
    Topics: []string{"group:123", "room:lobby"},
    Result: resultChan,
}

// Client unsubscribes
hub.Unsubscribe <- &websocket.UnsubscribeEvent{
    Client: client,
    Topics: []string{"group:123"},
}
```

## Rate Limiting

```go
store := middleware.NewRateLimiterStore()

router.Use(middleware.RateLimitWithStore(&middleware.RateLimitConfig{
    Default: 10, // 10 requests/second
    Methods: map[string]float64{
        "heartbeat": 0,  // No limit
        "subscribe": 5,  // 5 requests/second
    },
}, store))

// Clean up when client disconnects
hub := websocket.NewHub(
    websocket.WithClientDisconnectCallback(store.Remove),
)
```

## Configuration

```go
cfg := &websocket.HubConfig{
    AnonymousTimeout: 10 * time.Second,  // Disconnect if not logged in within 10s
    AnonymousCleanup: 2 * time.Second,   // Cleanup interval
    HeartbeatTimeout: 60 * time.Second,  // No heartbeat for 60s -> disconnect
    HeartbeatCleanup: 30 * time.Second,
    PingPeriod:       54 * time.Second,  // WebSocket ping interval
    PongWait:         60 * time.Second,
    MaxMessageSize:   4096,
    WriteWait:        10 * time.Second,
    MaxConnections:   10000,             // Max total connections (0 = unlimited)
    MaxConnsPerUser:  5,                 // Max connections per user (0 = unlimited)
}

// Validate config before use
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

hub := websocket.NewHubWithConfig(cfg)
```

## Prometheus Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

// Create and register metrics
metrics := websocket.NewMetrics("myapp", "websocket")
metrics.MustRegister(prometheus.DefaultRegisterer)

// Attach metrics to hub
hub := websocket.NewHub(websocket.WithMetrics(metrics))

// Available metrics:
// - myapp_websocket_connections_total
// - myapp_websocket_connections_current
// - myapp_websocket_connections_authenticated
// - myapp_websocket_connections_anonymous
// - myapp_websocket_messages_sent_total
// - myapp_websocket_broadcasts_total
// - myapp_websocket_errors_total{type="connection_limit|user_limit|..."}
// - myapp_websocket_topics_current
// - myapp_websocket_subscriptions_total
```

## Connection Limits

```go
// Check before accepting connection (optional, for early rejection)
if !hub.CanAcceptConnection() {
    http.Error(w, "Too many connections", http.StatusServiceUnavailable)
    return
}

// Check before login (optional)
if !hub.CanUserConnect(userID) {
    return c.Error(errors.New(429, "TooManyConnections", "Max connections reached"))
}

// Limits are also enforced automatically in hub
```

## Examples

See [examples/basic](examples/basic) for a complete working example with:
- Hub configuration and validation
- Prometheus metrics integration
- Rate limiting middleware
- Public and private route groups
- Connection limits
- Graceful shutdown

## Error Code Mapping

| HTTP Status | JSON-RPC Code | Description |
|-------------|---------------|-------------|
| 400 | -32602 | Invalid params |
| 401 | -32001 | Unauthorized |
| 403 | -32003 | Permission denied |
| 404 | -32004 | Not found |
| 429 | -32029 | Too many requests |
| 500 | -32603 | Internal error |

## License

MIT License
