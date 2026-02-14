package dashboard

import (
	"log"
	"sync"
)

// Client represents a connected WebSocket client
type Client struct {
	ID   string
	Send chan interface{}
}

// WebSocketMessage is the envelope for all messages sent to clients
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
}

// Broadcaster manages WebSocket client connections and broadcasts messages
type Broadcaster struct {
	clients      map[*Client]bool
	broadcast    chan interface{}
	register     chan *Client
	unregister   chan *Client
	mu           sync.RWMutex
	logger       *log.Logger
	shutdown     chan struct{}
}

// NewBroadcaster creates a new Broadcaster instance
func NewBroadcaster(logger *log.Logger) *Broadcaster {
	return &Broadcaster{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan interface{}, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
		shutdown:   make(chan struct{}),
	}
}

// Register registers a new client for broadcasts
func (b *Broadcaster) Register(client *Client) {
	b.register <- client
}

// Unregister removes a client from broadcasts
func (b *Broadcaster) Unregister(client *Client) {
	b.unregister <- client
}

// Broadcast sends a message to all connected clients
func (b *Broadcaster) Broadcast(message interface{}) {
	select {
	case b.broadcast <- message:
	case <-b.shutdown:
		// Broadcaster is shutting down, drop message
	}
}

// Run starts the broadcaster loop (should be called in a goroutine)
func (b *Broadcaster) Run() {
	defer func() {
		b.logger.Println("broadcaster: shutting down")
		close(b.shutdown)
	}()

	for {
		select {
		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = true
			b.mu.Unlock()
			b.logger.Printf("broadcaster: client registered (total: %d)", len(b.clients))

		case client := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				// Close the Send channel to prevent sends on closed channel
				close(client.Send)
			}
			b.mu.Unlock()
			b.logger.Printf("broadcaster: client unregistered (total: %d)", len(b.clients))

		case message := <-b.broadcast:
			b.mu.RLock()
			clients := make([]*Client, 0, len(b.clients))
			for client := range b.clients {
				clients = append(clients, client)
			}
			b.mu.RUnlock()

			// Send to all clients (non-blocking)
			for _, client := range clients {
				select {
				case client.Send <- message:
					// Message sent
				default:
					// Client's Send channel is full, skip this client
					// This prevents blocking the broadcaster if one client is slow
					b.logger.Printf("broadcaster: client %s send channel full, skipping", client.ID)
				}
			}

		case <-b.shutdown:
			return
		}
	}
}

// Shutdown gracefully shuts down the broadcaster
func (b *Broadcaster) Shutdown() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Close all client connections
	for client := range b.clients {
		close(client.Send)
	}
	b.clients = make(map[*Client]bool)

	// Signal shutdown
	close(b.broadcast)
}

// ClientCount returns the number of connected clients
func (b *Broadcaster) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}
