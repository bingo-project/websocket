// ABOUTME: Tests for WebSocket connection hub.
// ABOUTME: Validates client registration, login, and unregistration.

package websocket_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bingo-project/websocket"
)

func TestHub_RegisterAndUnregister(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHub()
	go hub.Run(ctx)

	// Create mock client
	client := &websocket.Client{
		Addr: "127.0.0.1:8080",
		Send: make(chan []byte, 10),
	}

	// Register client (goes to anonymous)
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.AnonymousCount())

	// Unregister client
	hub.Unregister <- client
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, hub.AnonymousCount())
}

func TestHub_Login(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHub()
	go hub.Run(ctx)

	client := &websocket.Client{
		Addr: "127.0.0.1:8080",
		Send: make(chan []byte, 10),
	}

	// Register first
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Login
	hub.Login <- &websocket.LoginEvent{
		Client:   client,
		UserID:   "user-123",
		Platform: websocket.PlatformIOS,
	}
	time.Sleep(10 * time.Millisecond)

	// Verify user is tracked
	assert.Equal(t, 1, hub.UserCount())
	assert.NotNil(t, hub.GetUserClient(websocket.PlatformIOS, "user-123"))
}

func TestHub_Broadcast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHub()
	go hub.Run(ctx)

	client1 := &websocket.Client{Addr: "client1", Send: make(chan []byte, 10)}
	client2 := &websocket.Client{Addr: "client2", Send: make(chan []byte, 10)}

	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	// Login both clients to make them authenticated
	hub.Login <- &websocket.LoginEvent{Client: client1, UserID: "user1", Platform: websocket.PlatformIOS}
	hub.Login <- &websocket.LoginEvent{Client: client2, UserID: "user2", Platform: websocket.PlatformWeb}
	time.Sleep(10 * time.Millisecond)

	// Broadcast message
	hub.Broadcast <- []byte("hello")
	time.Sleep(10 * time.Millisecond)

	// Both clients should receive
	assert.Equal(t, 1, len(client1.Send))
	assert.Equal(t, 1, len(client2.Send))
}

func TestHub_AnonymousCount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHubWithConfig(websocket.DefaultHubConfig())
	go hub.Run(ctx)

	client := &websocket.Client{
		Addr: "127.0.0.1:8080",
		Send: make(chan []byte, 10),
	}

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Client is in anonymous state
	assert.Equal(t, 1, hub.AnonymousCount())
	assert.Equal(t, 0, hub.ClientCount())
}

func TestHub_AnonymousToAuthenticated(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHubWithConfig(websocket.DefaultHubConfig())
	go hub.Run(ctx)

	client := &websocket.Client{
		Addr: "127.0.0.1:8080",
		Send: make(chan []byte, 10),
	}

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.AnonymousCount())

	// Login moves client from anonymous to authenticated
	hub.Login <- &websocket.LoginEvent{
		Client:   client,
		UserID:   "user-123",
		Platform: websocket.PlatformIOS,
	}
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 0, hub.AnonymousCount())
	assert.Equal(t, 1, hub.ClientCount())
	assert.Equal(t, 1, hub.UserCount())
}

func TestHub_KickPreviousSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHubWithConfig(websocket.DefaultHubConfig())
	go hub.Run(ctx)

	// First client logs in
	client1 := &websocket.Client{
		Addr:      "client1",
		Send:      make(chan []byte, 10),
		FirstTime: time.Now().Unix(),
	}
	hub.Register <- client1
	time.Sleep(10 * time.Millisecond)

	hub.Login <- &websocket.LoginEvent{
		Client:   client1,
		UserID:   "user-123",
		Platform: websocket.PlatformIOS,
	}
	time.Sleep(10 * time.Millisecond)

	// Second client logs in with same user/platform
	client2 := &websocket.Client{
		Addr:      "client2",
		Send:      make(chan []byte, 10),
		FirstTime: time.Now().Unix(),
	}
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	hub.Login <- &websocket.LoginEvent{
		Client:   client2,
		UserID:   "user-123",
		Platform: websocket.PlatformIOS,
	}
	time.Sleep(150 * time.Millisecond) // Wait for kick delay

	// First client should receive kick notification
	select {
	case msg := <-client1.Send:
		assert.Contains(t, string(msg), "session.kicked")
	default:
		t.Error("client1 should receive kick notification")
	}

	// Only client2 should remain
	assert.Equal(t, 1, hub.ClientCount())
	assert.Equal(t, 1, hub.UserCount())
	assert.Equal(t, client2, hub.GetUserClient(websocket.PlatformIOS, "user-123"))
}

func TestHub_AnonymousTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use short timeout for testing
	cfg := &websocket.HubConfig{
		AnonymousTimeout: 50 * time.Millisecond,
		AnonymousCleanup: 20 * time.Millisecond,
		HeartbeatTimeout: 60 * time.Second,
		HeartbeatCleanup: 30 * time.Second,
		PingPeriod:       54 * time.Second,
		PongWait:         60 * time.Second,
	}

	hub := websocket.NewHubWithConfig(cfg)
	go hub.Run(ctx)

	client := &websocket.Client{
		Addr:      "127.0.0.1:8080",
		Send:      make(chan []byte, 10),
		FirstTime: time.Now().Unix(),
	}

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.AnonymousCount())

	// Wait for timeout + cleanup
	time.Sleep(100 * time.Millisecond)

	// Should be cleaned up
	assert.Equal(t, 0, hub.AnonymousCount())
}

func TestHub_Subscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHubWithConfig(websocket.DefaultHubConfig())
	go hub.Run(ctx)

	client := &websocket.Client{
		Addr:      "127.0.0.1:8080",
		Send:      make(chan []byte, 10),
		FirstTime: time.Now().Unix(),
	}
	hub.Register <- client
	hub.Login <- &websocket.LoginEvent{
		Client:   client,
		UserID:   "user-123",
		Platform: websocket.PlatformIOS,
	}
	time.Sleep(10 * time.Millisecond)

	// Subscribe to topics
	result := make(chan []string, 1)
	hub.Subscribe <- &websocket.SubscribeEvent{
		Client: client,
		Topics: []string{"group:123", "room:lobby"},
		Result: result,
	}

	subscribed := <-result
	assert.ElementsMatch(t, []string{"group:123", "room:lobby"}, subscribed)

	// Verify topic count
	assert.Equal(t, 2, hub.TopicCount())
}

func TestHub_PushToTopic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHubWithConfig(websocket.DefaultHubConfig())
	go hub.Run(ctx)

	// Create and login two clients
	client1 := &websocket.Client{Addr: "client1", Send: make(chan []byte, 10), FirstTime: time.Now().Unix()}
	client2 := &websocket.Client{Addr: "client2", Send: make(chan []byte, 10), FirstTime: time.Now().Unix()}

	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	hub.Login <- &websocket.LoginEvent{Client: client1, UserID: "user1", Platform: websocket.PlatformIOS}
	hub.Login <- &websocket.LoginEvent{Client: client2, UserID: "user2", Platform: websocket.PlatformWeb}
	time.Sleep(10 * time.Millisecond)

	// Subscribe client1 to topic
	result := make(chan []string, 1)
	hub.Subscribe <- &websocket.SubscribeEvent{Client: client1, Topics: []string{"group:123"}, Result: result}
	<-result

	// Push to topic
	hub.PushToTopic("group:123", "message.new", map[string]string{"content": "hello"})
	time.Sleep(10 * time.Millisecond)

	// Only client1 should receive
	assert.Equal(t, 1, len(client1.Send))
	assert.Equal(t, 0, len(client2.Send))

	msg := <-client1.Send
	assert.Contains(t, string(msg), "message.new")
	assert.Contains(t, string(msg), "hello")
}

func TestHub_PushToUser(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHubWithConfig(websocket.DefaultHubConfig())
	go hub.Run(ctx)

	client := &websocket.Client{Addr: "client1", Send: make(chan []byte, 10), FirstTime: time.Now().Unix()}
	hub.Register <- client
	hub.Login <- &websocket.LoginEvent{Client: client, UserID: "user-123", Platform: websocket.PlatformIOS}
	time.Sleep(10 * time.Millisecond)

	hub.PushToUser(websocket.PlatformIOS, "user-123", "order.created", map[string]string{"order_id": "123"})
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, len(client.Send))
	msg := <-client.Send
	assert.Contains(t, string(msg), "order.created")
}

func TestHub_PushToUserAllPlatforms(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHubWithConfig(websocket.DefaultHubConfig())
	go hub.Run(ctx)

	// Same user on two platforms
	client1 := &websocket.Client{Addr: "client1", Send: make(chan []byte, 10), FirstTime: time.Now().Unix()}
	client2 := &websocket.Client{Addr: "client2", Send: make(chan []byte, 10), FirstTime: time.Now().Unix()}

	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	hub.Login <- &websocket.LoginEvent{Client: client1, UserID: "user-123", Platform: websocket.PlatformIOS}
	hub.Login <- &websocket.LoginEvent{Client: client2, UserID: "user-123", Platform: websocket.PlatformWeb}
	time.Sleep(10 * time.Millisecond)

	hub.PushToUserAllPlatforms("user-123", "security.alert", map[string]string{"message": "new login"})
	time.Sleep(10 * time.Millisecond)

	// Both clients should receive
	assert.Equal(t, 1, len(client1.Send))
	assert.Equal(t, 1, len(client2.Send))
}

func TestHub_Unsubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHubWithConfig(websocket.DefaultHubConfig())
	go hub.Run(ctx)

	client := &websocket.Client{
		Addr:      "127.0.0.1:8080",
		Send:      make(chan []byte, 10),
		FirstTime: time.Now().Unix(),
	}
	hub.Register <- client
	hub.Login <- &websocket.LoginEvent{
		Client:   client,
		UserID:   "user-123",
		Platform: websocket.PlatformIOS,
	}
	time.Sleep(10 * time.Millisecond)

	// Subscribe
	result := make(chan []string, 1)
	hub.Subscribe <- &websocket.SubscribeEvent{
		Client: client,
		Topics: []string{"group:123", "room:lobby"},
		Result: result,
	}
	<-result

	// Unsubscribe one topic
	hub.Unsubscribe <- &websocket.UnsubscribeEvent{
		Client: client,
		Topics: []string{"group:123"},
	}
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.TopicCount())
}

func TestHub_TokenExpiration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &websocket.HubConfig{
		AnonymousTimeout: 10 * time.Second,
		AnonymousCleanup: 2 * time.Second,
		HeartbeatTimeout: 60 * time.Second,
		HeartbeatCleanup: 50 * time.Millisecond, // Fast for testing
		PingPeriod:       54 * time.Second,
		PongWait:         60 * time.Second,
	}

	hub := websocket.NewHubWithConfig(cfg)
	go hub.Run(ctx)

	now := time.Now().Unix()
	client := &websocket.Client{Addr: "client1", Send: make(chan []byte, 10), FirstTime: now, HeartbeatTime: now}
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Login with token that expires immediately
	hub.Login <- &websocket.LoginEvent{
		Client:         client,
		UserID:         "user-123",
		Platform:       websocket.PlatformIOS,
		TokenExpiresAt: time.Now().Unix() - 1, // Already expired
	}
	time.Sleep(150 * time.Millisecond)

	// Should receive session.expired notification
	select {
	case msg := <-client.Send:
		assert.Contains(t, string(msg), "session.expired")
	default:
		t.Error("Should receive session.expired notification")
	}
}

func TestHub_UnsubscribeAllOnDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHubWithConfig(websocket.DefaultHubConfig())
	go hub.Run(ctx)
	time.Sleep(10 * time.Millisecond) // Wait for hub to start

	now := time.Now().Unix()
	client := &websocket.Client{Addr: "client1", Send: make(chan []byte, 10), FirstTime: now, HeartbeatTime: now}
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)
	hub.Login <- &websocket.LoginEvent{Client: client, UserID: "user-123", Platform: websocket.PlatformIOS}
	time.Sleep(10 * time.Millisecond)

	// Subscribe to topics
	result := make(chan []string, 1)
	hub.Subscribe <- &websocket.SubscribeEvent{Client: client, Topics: []string{"group:123", "room:456"}, Result: result}
	<-result
	assert.Equal(t, 2, hub.TopicCount())

	// Disconnect
	hub.Unregister <- client
	time.Sleep(50 * time.Millisecond) // Increase wait time

	// Topics should be cleaned up
	assert.Equal(t, 0, hub.TopicCount())
}

func TestHub_GracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	hub := websocket.NewHub()
	go hub.Run(ctx)

	client := &websocket.Client{
		Addr: "127.0.0.1:8080",
		Send: make(chan []byte, 10),
	}

	hub.Register <- client
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.AnonymousCount())

	// Cancel context to trigger shutdown
	cancel()
	time.Sleep(20 * time.Millisecond)

	// Verify client is cleaned up
	assert.Equal(t, 0, hub.AnonymousCount())

	// Should receive shutdown notification
	select {
	case msg := <-client.Send:
		assert.Contains(t, string(msg), "session.shutdown")
	default:
		// May have already been drained or not sent
	}

	// Drain any remaining messages and verify channel is eventually closed
	for {
		_, ok := <-client.Send
		if !ok {
			break // Channel closed
		}
	}
	// If we get here, channel was successfully closed
}

func TestHub_GetClient(t *testing.T) {
	hub := websocket.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Create and register client
	client := &websocket.Client{
		ID:   "test-client-id",
		Send: make(chan []byte, 256),
	}
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	// Get by ID
	found := hub.GetClient("test-client-id")
	assert.Equal(t, client, found)

	// Not found
	notFound := hub.GetClient("unknown")
	assert.Nil(t, notFound)
}

func TestHub_GetClientsByUser(t *testing.T) {
	hub := websocket.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Create clients for same user on different platforms
	client1 := &websocket.Client{
		ID:   "c1",
		Addr: "client1",
		Send: make(chan []byte, 256),
	}
	client2 := &websocket.Client{
		ID:   "c2",
		Addr: "client2",
		Send: make(chan []byte, 256),
	}

	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	hub.Login <- &websocket.LoginEvent{Client: client1, UserID: "user-123", Platform: websocket.PlatformWeb}
	hub.Login <- &websocket.LoginEvent{Client: client2, UserID: "user-123", Platform: websocket.PlatformIOS}
	time.Sleep(10 * time.Millisecond)

	clients := hub.GetClientsByUser("user-123")
	assert.Len(t, clients, 2)
}

func TestHub_Stats(t *testing.T) {
	hub := websocket.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Create anonymous client
	anonClient := &websocket.Client{
		ID:   "anon",
		Addr: "anon",
		Send: make(chan []byte, 256),
	}
	hub.Register <- anonClient
	time.Sleep(10 * time.Millisecond)

	// Create authenticated clients
	client1 := &websocket.Client{
		ID:   "c1",
		Addr: "client1",
		Send: make(chan []byte, 256),
	}
	client2 := &websocket.Client{
		ID:   "c2",
		Addr: "client2",
		Send: make(chan []byte, 256),
	}
	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	hub.Login <- &websocket.LoginEvent{Client: client1, UserID: "user1", Platform: websocket.PlatformWeb}
	hub.Login <- &websocket.LoginEvent{Client: client2, UserID: "user2", Platform: websocket.PlatformIOS}
	time.Sleep(10 * time.Millisecond)

	stats := hub.Stats()

	assert.Equal(t, int64(1), stats.AnonymousConns)
	assert.Equal(t, int64(2), stats.AuthenticatedConns)
	assert.Equal(t, int64(3), stats.TotalConnections)
	assert.Equal(t, 1, stats.ConnectionsByPlatform[websocket.PlatformWeb])
	assert.Equal(t, 1, stats.ConnectionsByPlatform[websocket.PlatformIOS])
}

func TestHub_KickClient(t *testing.T) {
	hub := websocket.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	client := &websocket.Client{
		ID:   "test-client",
		Addr: "client1",
		Send: make(chan []byte, 256),
	}
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	hub.Login <- &websocket.LoginEvent{Client: client, UserID: "user-123", Platform: websocket.PlatformWeb}
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount())

	// Kick client
	kicked := hub.KickClient("test-client", "test kick")
	assert.True(t, kicked)

	// Should receive kick notification
	time.Sleep(20 * time.Millisecond)
	select {
	case msg := <-client.Send:
		assert.Contains(t, string(msg), "session.kicked")
	default:
		t.Error("Should receive kick notification")
	}

	// Wait for kick to complete
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount())
}

func TestHub_KickUser(t *testing.T) {
	hub := websocket.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// User on multiple platforms
	client1 := &websocket.Client{
		ID:   "c1",
		Addr: "client1",
		Send: make(chan []byte, 256),
	}
	client2 := &websocket.Client{
		ID:   "c2",
		Addr: "client2",
		Send: make(chan []byte, 256),
	}

	hub.Register <- client1
	hub.Register <- client2
	time.Sleep(10 * time.Millisecond)

	hub.Login <- &websocket.LoginEvent{Client: client1, UserID: "user-123", Platform: websocket.PlatformWeb}
	hub.Login <- &websocket.LoginEvent{Client: client2, UserID: "user-123", Platform: websocket.PlatformIOS}
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 2, hub.ClientCount())

	// Kick all user's connections
	kicked := hub.KickUser("user-123", "account suspended")
	assert.Equal(t, 2, kicked)

	// Wait for kicks to complete
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount())
}

// Concurrent tests for race detection

func TestHub_ConcurrentRegisterUnregister(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHub()
	go hub.Run(ctx)

	const numClients = 100
	clients := make([]*websocket.Client, numClients)

	// Create clients
	for i := 0; i < numClients; i++ {
		clients[i] = &websocket.Client{
			ID:   string(rune('a' + i%26)) + string(rune(i)),
			Addr: "client",
			Send: make(chan []byte, 10),
		}
	}

	// Concurrent register
	for i := 0; i < numClients; i++ {
		go func(c *websocket.Client) {
			hub.Register <- c
		}(clients[i])
	}
	time.Sleep(50 * time.Millisecond)

	// Concurrent unregister
	for i := 0; i < numClients; i++ {
		go func(c *websocket.Client) {
			hub.Unregister <- c
		}(clients[i])
	}
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.AnonymousCount())
}

func TestHub_ConcurrentLoginAndBroadcast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHub()
	go hub.Run(ctx)

	const numClients = 50
	clients := make([]*websocket.Client, numClients)
	platforms := []string{websocket.PlatformWeb, websocket.PlatformIOS, websocket.PlatformAndroid}

	// Register and login clients concurrently
	for i := 0; i < numClients; i++ {
		clients[i] = &websocket.Client{
			ID:   string(rune('a'+i%26)) + string(rune(i)),
			Addr: "client",
			Send: make(chan []byte, 256),
		}
		hub.Register <- clients[i]
	}
	time.Sleep(20 * time.Millisecond)

	// Concurrent logins
	for i := 0; i < numClients; i++ {
		go func(idx int) {
			hub.Login <- &websocket.LoginEvent{
				Client:   clients[idx],
				UserID:   "user-" + string(rune('0'+idx%10)),
				Platform: platforms[idx%3],
			}
		}(i)
	}
	time.Sleep(50 * time.Millisecond)

	// Concurrent broadcasts while clients are connected
	for i := 0; i < 10; i++ {
		go func() {
			hub.Broadcast <- []byte(`{"jsonrpc":"2.0","method":"test","params":{}}`)
		}()
	}
	time.Sleep(50 * time.Millisecond)

	// Drain client send channels
	for _, c := range clients {
		for len(c.Send) > 0 {
			<-c.Send
		}
	}

	assert.True(t, hub.ClientCount() > 0)
}

func TestHub_ConcurrentSubscribeAndPush(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHub()
	go hub.Run(ctx)

	const numClients = 30
	clients := make([]*websocket.Client, numClients)
	topics := []string{"topic1", "topic2", "topic3"}

	// Setup clients
	for i := 0; i < numClients; i++ {
		clients[i] = &websocket.Client{
			ID:   string(rune('a'+i%26)) + string(rune(i)),
			Addr: "client",
			Send: make(chan []byte, 256),
		}
		hub.Register <- clients[i]
	}
	time.Sleep(20 * time.Millisecond)

	for i := 0; i < numClients; i++ {
		hub.Login <- &websocket.LoginEvent{
			Client:   clients[i],
			UserID:   "user-" + string(rune('0'+i%10)),
			Platform: websocket.PlatformWeb,
		}
	}
	time.Sleep(20 * time.Millisecond)

	// Concurrent subscriptions
	for i := 0; i < numClients; i++ {
		go func(idx int) {
			hub.Subscribe <- &websocket.SubscribeEvent{
				Client: clients[idx],
				Topics: []string{topics[idx%3]},
			}
		}(i)
	}
	time.Sleep(50 * time.Millisecond)

	// Concurrent push to topics while subscriptions may still be processing
	for i := 0; i < 20; i++ {
		go func(idx int) {
			hub.PushToTopic(topics[idx%3], "test.push", map[string]int{"n": idx})
		}(i)
	}
	time.Sleep(50 * time.Millisecond)

	// Concurrent unsubscribe
	for i := 0; i < numClients; i++ {
		go func(idx int) {
			hub.Unsubscribe <- &websocket.UnsubscribeEvent{
				Client: clients[idx],
				Topics: []string{topics[idx%3]},
			}
		}(i)
	}
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.TopicCount())
}
