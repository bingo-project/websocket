// ABOUTME: Benchmark tests for WebSocket hub performance.
// ABOUTME: Measures broadcast, subscription, and connection handling speed.

package websocket_test

import (
	"context"
	"testing"
	"time"

	"github.com/bingo-project/websocket"
)

func BenchmarkHub_Broadcast(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHub()
	go hub.Run(ctx)

	// Setup 1000 clients
	const numClients = 1000
	clients := make([]*websocket.Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = &websocket.Client{
			ID:   string(rune(i)),
			Addr: "client",
			Send: make(chan []byte, 256),
		}
		hub.Register <- clients[i]
	}
	time.Sleep(50 * time.Millisecond)

	// Login all clients
	for i := 0; i < numClients; i++ {
		hub.Login <- &websocket.LoginEvent{
			Client:   clients[i],
			UserID:   "user",
			Platform: websocket.PlatformWeb,
		}
	}
	time.Sleep(50 * time.Millisecond)

	msg := []byte(`{"jsonrpc":"2.0","method":"test","params":{"data":"benchmark"}}`)

	// Drain channels in background
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				for _, c := range clients {
					select {
					case <-c.Send:
					default:
					}
				}
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.Broadcast <- msg
	}
	b.StopTimer()

	close(done)
}

func BenchmarkHub_Subscribe(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHub()
	go hub.Run(ctx)

	client := &websocket.Client{
		ID:   "bench-client",
		Addr: "client",
		Send: make(chan []byte, 256),
	}
	hub.Register <- client
	time.Sleep(10 * time.Millisecond)

	hub.Login <- &websocket.LoginEvent{
		Client:   client,
		UserID:   "user",
		Platform: websocket.PlatformWeb,
	}
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := make(chan []string, 1)
		hub.Subscribe <- &websocket.SubscribeEvent{
			Client: client,
			Topics: []string{"topic-" + string(rune(i%100))},
			Result: result,
		}
		<-result
	}
}

func BenchmarkHub_PushToTopic(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHub()
	go hub.Run(ctx)

	// Setup 100 clients subscribed to topic
	const numClients = 100
	clients := make([]*websocket.Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = &websocket.Client{
			ID:   string(rune(i)),
			Addr: "client",
			Send: make(chan []byte, 256),
		}
		hub.Register <- clients[i]
	}
	time.Sleep(20 * time.Millisecond)

	for i := 0; i < numClients; i++ {
		hub.Login <- &websocket.LoginEvent{
			Client:   clients[i],
			UserID:   "user-" + string(rune(i)),
			Platform: websocket.PlatformWeb,
		}
	}
	time.Sleep(20 * time.Millisecond)

	// Subscribe all to same topic
	for i := 0; i < numClients; i++ {
		hub.Subscribe <- &websocket.SubscribeEvent{
			Client: clients[i],
			Topics: []string{"bench-topic"},
		}
	}
	time.Sleep(20 * time.Millisecond)

	// Drain channels in background
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				for _, c := range clients {
					select {
					case <-c.Send:
					default:
					}
				}
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.PushToTopic("bench-topic", "test.push", map[string]int{"n": i})
	}
	b.StopTimer()

	close(done)
}

func BenchmarkHub_RegisterUnregister(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := websocket.NewHub()
	go hub.Run(ctx)
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := &websocket.Client{
			ID:        "bench-" + string(rune(i)),
			Addr:      "client",
			Send:      make(chan []byte, 10),
			FirstTime: time.Now().Unix(),
		}
		hub.Register <- client
		// Small delay to ensure registration is processed before unregistration
		time.Sleep(time.Microsecond)
		hub.Unregister <- client
	}
}
