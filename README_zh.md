# WebSocket

[![Go Reference](https://pkg.go.dev/badge/github.com/bingo-project/websocket.svg)](https://pkg.go.dev/github.com/bingo-project/websocket)
[![Go Report Card](https://goreportcard.com/badge/github.com/bingo-project/websocket)](https://goreportcard.com/report/github.com/bingo-project/websocket)
[![CI](https://github.com/bingo-project/websocket/actions/workflows/test.yml/badge.svg)](https://github.com/bingo-project/websocket/actions/workflows/test.yml)

一个生产就绪的 Go WebSocket 框架，采用 JSON-RPC 2.0 协议，支持中间件、分组路由和连接管理。[Bingo](https://bingoctl.dev) 生态组件。

📖 **文档**: [bingoctl.dev/advanced/websocket](https://bingoctl.dev/advanced/websocket)

[English](README.md)

## 特性

- **JSON-RPC 2.0 协议** - 行业标准消息格式（MCP、以太坊、VSCode LSP 等广泛使用）
- **中间件模式** - 类似 Gin/Echo 的熟悉编程模型
- **分组路由** - 支持 public/private 分组，不同方法使用不同中间件链
- **连接管理** - Hub 管理客户端注册、认证和主题订阅
- **内置处理器** - 开箱即用的心跳、订阅/取消订阅、Token 登录
- **限流** - 令牌桶算法，支持按方法配置
- **单设备登录** - 同一用户再次登录时自动踢掉旧会话
- **Prometheus 指标** - 内置可观测性，包含连接、消息和错误指标
- **连接限制** - 可配置的总连接数和单用户连接数限制
- **优雅关闭** - 关闭前通知客户端

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

### 自定义登录处理器

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

### Token 认证登录

对于基于 token 的认证，可使用内置的 `TokenLoginHandler` 配合 `LoginStateUpdater` 中间件：

```go
// 配置 token 解析器
client := websocket.NewClient(hub, conn, ctx,
    websocket.WithRouter(router),
    websocket.WithTokenParser(func(token string) (*websocket.TokenInfo, error) {
        // 在这里验证和解析你的 JWT/token
        claims, err := jwt.Parse(token)
        if err != nil {
            return nil, err
        }
        return &websocket.TokenInfo{
            UserID:    claims.UserID,
            ExpiresAt: claims.ExpiresAt,
        }, nil
    }),
)

// 注册处理器和中间件
router.Handle("login", websocket.TokenLoginHandler, middleware.LoginStateUpdater)
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
    MaxConnections:   10000,             // 最大总连接数 (0 = 不限制)
    MaxConnsPerUser:  5,                 // 每用户最大连接数 (0 = 不限制)
}

// 使用前验证配置
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

hub := websocket.NewHubWithConfig(cfg)
```

## Prometheus 指标

```go
import "github.com/prometheus/client_golang/prometheus"

// 创建并注册指标
metrics := websocket.NewMetrics("myapp", "websocket")
metrics.MustRegister(prometheus.DefaultRegisterer)

// 将指标附加到 hub
hub := websocket.NewHub(websocket.WithMetrics(metrics))

// 可用指标:
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

## 连接限制

```go
// 接受连接前检查（可选，用于提前拒绝）
if !hub.CanAcceptConnection() {
    http.Error(w, "连接数过多", http.StatusServiceUnavailable)
    return
}

// 登录前检查（可选）
if !hub.CanUserConnect(userID) {
    return c.Error(errors.New(429, "TooManyConnections", "已达到最大连接数"))
}

// 限制也会在 hub 中自动执行
```

## 优雅关闭

```go
ctx, cancel := context.WithCancel(context.Background())
go hub.Run(ctx)

// 处理关闭信号
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
<-sigCh

// 取消 context 触发优雅关闭
// Hub 会在关闭前通知所有客户端
cancel()
```

## 示例

查看 [examples/basic](examples/basic) 获取完整示例，包括：
- Hub 配置和验证
- Prometheus 指标集成
- 限流中间件
- 公开和私有路由分组
- 连接限制
- 优雅关闭

## 错误码映射

| HTTP 状态码 | JSON-RPC 码 | 说明 |
|------------|-------------|------|
| 400 | -32602 | 无效参数 |
| 401 | -32001 | 未授权 |
| 403 | -32003 | 权限拒绝 |
| 404 | -32004 | 未找到 |
| 429 | -32029 | 请求过多 |
| 500 | -32603 | 内部错误 |

## 性能

### Benchmark 结果

在 Apple M1 Pro 上的测试结果：

| 操作 | 耗时 | 内存分配 |
|------|------|----------|
| Broadcast (1000 clients) | ~1.7μs | 0 allocs |
| Subscribe | ~1.3μs | 10 allocs |
| PushToTopic (100 clients) | ~6.3μs | 7 allocs |
| Register/Unregister | ~2.8μs | 9 allocs |

运行 benchmark：

```bash
go test -bench=. -benchmem ./...
```

### 容量估算

基于 [gorilla/websocket](https://github.com/gorilla/websocket) 和 Go 运行时特性，单机容量主要受内存限制：

| 服务器配置 | 预估连接数 | 说明 |
|-----------|-----------|------|
| 4核 8GB | 10,000 - 30,000 | 开发/测试环境 |
| 8核 16GB | 50,000 - 100,000 | 生产环境起步 |
| 16核 32GB | 100,000 - 200,000 | 中型生产环境 |
| 32核 64GB | 200,000 - 500,000 | 大型生产环境 |

**内存估算**：每连接约 20-30KB（含 2 个 goroutine、读写缓冲区、应用数据）

### 适用场景

✅ **推荐场景**：
- 即时通讯（IM）
- 实时通知推送
- 在线协作（文档、白板）
- 实时数据展示（股票、监控）
- 游戏状态同步

⚠️ **需要额外优化的场景**：
- 超大规模（100万+连接）：考虑使用 [gnet](https://github.com/panjf2000/gnet) 或 [nbio](https://github.com/lesismal/nbio) 等异步 I/O 库
- 超高频消息（10万+ msg/s）：考虑消息批量合并、压缩

### 生产环境调优

#### Linux 内核参数

```bash
# /etc/sysctl.conf

# 增加文件描述符限制
fs.file-max = 1000000

# TCP 连接优化
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.core.netdev_max_backlog = 65535

# 内存优化
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
```

#### 进程限制

```bash
# /etc/security/limits.conf
* soft nofile 1000000
* hard nofile 1000000
```

## 相关链接

- [Bingo 可插拔协议层](https://bingoctl.dev/advanced/protocol-layer) - 在 Bingo 中使用 WebSocket 作为可插拔协议

## 许可证

Apache License 2.0
