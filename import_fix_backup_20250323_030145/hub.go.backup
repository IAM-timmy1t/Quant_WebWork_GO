// hub.go - Centralized WebSocket management

package websocket

import (
    "context"
    "sync"
    "time"
)

// Hub manages WebSocket connections with advanced features
type Hub struct {
    // Client management
    clients     map[*Client]bool
    register    chan *Client
    unregister  chan *Client
    
    // Messaging
    broadcast   chan Message
    
    // Channel subscriptions with metadata
    channels    map[string]ChannelConfig
    
    // Configuration and statistics
    config      Config
    stats       HubStats
    
    // Synchronization
    mu          sync.RWMutex
    
    // Context for graceful shutdown
    ctx         context.Context
    cancel      context.CancelFunc
}

// NewHub creates a new WebSocket hub
func NewHub(config Config) *Hub {
    if config.MessageBufferSize == 0 {
        config.MessageBufferSize = 256
    }
    
    ctx, cancel := context.WithCancel(context.Background())
    
    return &Hub{
        clients:    make(map[*Client]bool),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        broadcast:  make(chan Message, config.MessageBufferSize),
        channels:   make(map[string]ChannelConfig),
        config:     config,
        stats:      HubStats{},
        ctx:        ctx,
        cancel:     cancel,
    }
}

// Broadcast sends a message to subscribers
func (h *Hub) Broadcast(channel string, messageType MessageType, data interface{}) {
    message := Message{
        Channel: channel,
        Type:    messageType,
        Data:    data,
        Time:    time.Now(),
    }
    
    h.broadcast <- message
}

// Start begins the hub's operation
func (h *Hub) Start(ctx context.Context) {
    parentCtx, cancel := context.WithCancel(ctx)
    h.ctx = parentCtx
    h.cancel = cancel
    
    go h.run()
}

// Stop gracefully shuts down the hub
func (h *Hub) Stop() {
    h.cancel()
}

// run processes hub events in a loop
func (h *Hub) run() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-h.ctx.Done():
            // Gracefully close all client connections
            h.mu.Lock()
            for client := range h.clients {
                client.Close()
            }
            h.mu.Unlock()
            return
            
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.stats.CurrentConnections++
            h.stats.TotalConnections++
            h.mu.Unlock()
            
        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                h.stats.CurrentConnections--
                close(client.Send)
            }
            h.mu.Unlock()
            
        case message := <-h.broadcast:
            h.mu.RLock()
            for client := range h.clients {
                // Check if client is subscribed to this channel
                if client.IsSubscribed(message.Channel) {
                    select {
                    case client.Send <- message:
                        h.stats.MessagesSent++
                    default:
                        // Channel buffer is full, client is likely slow or dead
                        h.mu.RUnlock()
                        h.mu.Lock()
                        close(client.Send)
                        delete(h.clients, client)
                        h.stats.CurrentConnections--
                        h.stats.DroppedConnections++
                        h.mu.Unlock()
                        h.mu.RLock()
                    }
                }
            }
            h.mu.RUnlock()
            
        case <-ticker.C:
            // Periodic maintenance and stats updates
            h.mu.Lock()
            h.stats.Timestamp = time.Now()
            // Could add additional maintenance tasks here
            h.mu.Unlock()
        }
    }
}

// RegisterChannel adds a new channel configuration
func (h *Hub) RegisterChannel(name string, config ChannelConfig) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    h.channels[name] = config
}

// GetStats returns current hub statistics
func (h *Hub) GetStats() HubStats {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    return h.stats
}

// BroadcastToClient sends a message to a specific client
func (h *Hub) BroadcastToClient(clientID string, messageType MessageType, data interface{}) {
    message := Message{
        Type: messageType,
        Data: data,
        Time: time.Now(),
    }
    
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    for client := range h.clients {
        if client.ID == clientID {
            select {
            case client.Send <- message:
                h.stats.MessagesSent++
            default:
                // Channel is full, skip this message
            }
            break
        }
    }
}
