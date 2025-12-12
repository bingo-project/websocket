# WebSocket

[![Go Reference](https://pkg.go.dev/badge/github.com/bingo-project/websocket.svg)](https://pkg.go.dev/github.com/bingo-project/websocket)
[![Go Report Card](https://goreportcard.com/badge/github.com/bingo-project/websocket)](https://goreportcard.com/report/github.com/bingo-project/websocket)

一个生产就绪的 Go WebSocket 框架，采用 JSON-RPC 2.0 协议，支持中间件、分组路由和连接管理。

[English](README.md)

## 特性

- **JSON-RPC 2.0 协议** - 行业标准消息格式（MCP、以太坊、VSCode LSP 等广泛使用）
- **中间件模式** - 类似 Gin/Echo 的熟悉编程模型
- **分组路由** - 支持 public/private 分组，不同方法使用不同中间件链
- **连接管理** - Hub 管理客户端注册、认证和主题订阅
- **内置处理器** - 开箱即用的心跳、订阅/取消订阅
- **限流** - 令牌桶算法，支持按方法配置
- **单设备登录** - 同一用户再次登录时自动踢掉旧会话

## 安装

```bash
go get github.com/bingo-project/websocket
```

## 快速开始

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
    // 创建 hub 和 router
    hub := websocket.NewHub()
    router := websocket.NewRouter()

    // 添加全局中间件
    router.Use(
        middleware.Recovery,
        middleware.RequestID,
        middleware.Logger,
    )

    // 公开方法（无需认证）
    public := router.Group()
    public.Handle("heartbeat", websocket.HeartbeatHandler)
    public.Handle("echo", func(c *websocket.Context) *jsonrpc.Response {
        return c.JSON(c.Request.Params)
    })

    // 私有方法（需要认证）
    private := router.Group(middleware.Auth)
    private.Handle("subscribe", websocket.SubscribeHandler)

    // 启动 hub
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go hub.Run(ctx)

    // WebSocket upgrader
    upgrader := gorillaWS.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
    }

    // HTTP 处理器
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

    log.Println("服务器启动于 :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## 消息格式

### 请求

```json
{
    "jsonrpc": "2.0",
    "method": "auth.login",
    "params": {"username": "test", "password": "123456"},
    "id": 1
}
```

### 成功响应

```json
{
    "jsonrpc": "2.0",
    "result": {"token": "xxx", "expiresAt": 1234567890},
    "id": 1
}
```

### 错误响应

```json
{
    "jsonrpc": "2.0",
    "error": {
        "code": -32001,
        "reason": "Unauthorized",
        "message": "需要登录"
    },
    "id": 1
}
```

### 服务端推送（通知）

```json
{
    "jsonrpc": "2.0",
    "method": "session.kicked",
    "params": {"reason": "您的账号已在其他设备登录"}
}
```

## 中间件

### 内置中间件

| 中间件 | 说明 |
|-------|------|
| `Recovery` / `RecoveryWithLogger` | 捕获 panic，返回 500 错误 |
| `RequestID` | 注入 request-id 到 context |
| `Logger` / `LoggerWithLogger` | 记录请求日志和延迟 |
| `Auth` | 验证用户已认证 |
| `RateLimit` / `RateLimitWithStore` | 令牌桶限流 |
| `LoginStateUpdater` | 登录成功后更新客户端状态 |

### 自定义中间件

```go
func MyMiddleware(next websocket.Handler) websocket.Handler {
    return func(c *websocket.Context) *jsonrpc.Response {
        // 处理器之前
        log.Printf("Method: %s", c.Method)

        resp := next(c)

        // 处理器之后
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

    // 业务逻辑...
    token := authenticate(req.Username, req.Password)

    // 更新客户端登录状态
    c.Client.NotifyLogin(userID, req.Platform, tokenExpiresAt)

    return c.JSON(map[string]any{
        "token":     token,
        "expiresAt": tokenExpiresAt,
    })
}
```

## 连接管理

### Hub API

```go
// 根据 ID 获取客户端
client := hub.GetClient("client-id")

// 获取用户的所有客户端
clients := hub.GetClientsByUser("user-123")

// 踢出客户端
hub.KickClient("client-id", "reason")

// 踢出用户的所有会话
hub.KickUser("user-123", "账号已被封禁")

// 获取统计信息
stats := hub.Stats()
```

### 推送消息

```go
// 推送给特定平台的特定用户
hub.PushToUser("ios", "user-123", "order.created", data)

// 推送给用户的所有平台
hub.PushToUserAllPlatforms("user-123", "security.alert", data)

// 推送给主题订阅者
hub.PushToTopic("group:123", "message.new", data)

// 广播给所有已认证客户端
hub.Broadcast <- message
```

### 主题订阅

```go
// 客户端订阅主题
hub.Subscribe <- &websocket.SubscribeEvent{
    Client: client,
    Topics: []string{"group:123", "room:lobby"},
    Result: resultChan,
}

// 客户端取消订阅
hub.Unsubscribe <- &websocket.UnsubscribeEvent{
    Client: client,
    Topics: []string{"group:123"},
}
```

## 限流

```go
store := middleware.NewRateLimiterStore()

router.Use(middleware.RateLimitWithStore(&middleware.RateLimitConfig{
    Default: 10, // 每秒 10 个请求
    Methods: map[string]float64{
        "heartbeat": 0,  // 不限制
        "subscribe": 5,  // 每秒 5 个请求
    },
}, store))

// 客户端断开时清理
hub := websocket.NewHub(
    websocket.WithClientDisconnectCallback(store.Remove),
)
```

## 配置

```go
cfg := &websocket.HubConfig{
    AnonymousTimeout: 10 * time.Second,  // 10s 内未登录则断开
    AnonymousCleanup: 2 * time.Second,   // 清理间隔
    HeartbeatTimeout: 60 * time.Second,  // 60s 无心跳则断开
    HeartbeatCleanup: 30 * time.Second,
    PingPeriod:       54 * time.Second,  // WebSocket ping 间隔
    PongWait:         60 * time.Second,
    MaxMessageSize:   4096,
    WriteWait:        10 * time.Second,
}

hub := websocket.NewHubWithConfig(cfg)
```

## 错误码映射

| HTTP 状态码 | JSON-RPC 码 | 说明 |
|------------|-------------|------|
| 400 | -32602 | 无效参数 |
| 401 | -32001 | 未授权 |
| 403 | -32003 | 权限拒绝 |
| 404 | -32004 | 未找到 |
| 429 | -32029 | 请求过多 |
| 500 | -32603 | 内部错误 |

## 许可证

MIT License
