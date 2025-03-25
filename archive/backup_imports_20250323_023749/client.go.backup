// client.go - Client handling

package websocket

import (
    "bytes"
    "encoding/json"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/gorilla/websocket"
)

// Client represents a WebSocket connection
type Client struct {
    ID          string
    Hub         *Hub
    Conn        *websocket.Conn
    Send        chan Message
    Filters     []string
    Subscriptions map[string]bool
    UserData    map[string]interface{}
    created     time.Time
    lastActive  time.Time
    mu          sync.RWMutex
    closed      bool
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
    return &Client{
        ID:           uuid.New().String(),
        Hub:          hub,
        Conn:         conn,
        Send:         make(chan Message, 256),
        Subscriptions: make(map[string]bool),
        UserData:     make(map[string]interface{}),
        created:      time.Now(),
        lastActive:   time.Now(),
    }
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
    defer func() {
        c.Hub.unregister <- c
        c.Close()
    }()

    c.Conn.SetReadLimit(c.Hub.config.MaxMessageSize)
    c.Conn.SetReadDeadline(time.Now().Add(c.Hub.config.PongWait))
    c.Conn.SetPongHandler(func(string) error {
        c.Conn.SetReadDeadline(time.Now().Add(c.Hub.config.PongWait))
        return nil
    })

    for {
        _, message, err := c.Conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                // Log unexpected close
            }
            break
        }

        c.mu.Lock()
        c.lastActive = time.Now()
        c.mu.Unlock()

        message = bytes.TrimSpace(bytes.Replace(message, []byte{'\n'}, []byte{' '}, -1))
        
        // Process incoming message
        c.processIncomingMessage(message)
    }
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
    ticker := time.NewTicker(c.Hub.config.PingPeriod)
    defer func() {
        ticker.Stop()
        c.Close()
    }()

    for {
        select {
        case message, ok := <-c.Send:
            c.Conn.SetWriteDeadline(time.Now().Add(c.Hub.config.WriteWait))
            if !ok {
                // The hub closed the channel
                c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            w, err := c.Conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }

            // Serialize message to JSON
            messageData, err := json.Marshal(message)
            if err != nil {
                // Handle error
                continue
            }

            w.Write(messageData)

            // Add queued messages to the current websocket message
            n := len(c.Send)
            for i := 0; i < n; i++ {
                w.Write([]byte{'\n'})
                nextMsg := <-c.Send
                nextData, err := json.Marshal(nextMsg)
                if err != nil {
                    continue
                }
                w.Write(nextData)
            }

            if err := w.Close(); err != nil {
                return
            }
        case <-ticker.C:
            c.Conn.SetWriteDeadline(time.Now().Add(c.Hub.config.WriteWait))
            if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

// Close closes the client connection
func (c *Client) Close() {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.closed {
        return
    }

    c.closed = true
    c.Conn.Close()
}

// Subscribe adds a channel subscription
func (c *Client) Subscribe(channel string) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.Subscriptions[channel] = true
}

// Unsubscribe removes a channel subscription
func (c *Client) Unsubscribe(channel string) {
    c.mu.Lock()
    defer c.mu.Unlock()

    delete(c.Subscriptions, channel)
}

// IsSubscribed checks if client is subscribed to a channel
func (c *Client) IsSubscribed(channel string) bool {
    c.mu.RLock()
    defer c.mu.RUnlock()

    // Empty channel means broadcast to all clients
    if channel == "" {
        return true
    }

    return c.Subscriptions[channel]
}

// processIncomingMessage handles client messages
func (c *Client) processIncomingMessage(data []byte) {
    var clientMessage struct {
        Type    string          `json:"type"`
        Channel string          `json:"channel,omitempty"`
        Data    json.RawMessage `json:"data,omitempty"`
    }

    if err := json.Unmarshal(data, &clientMessage); err != nil {
        // Handle parse error
        return
    }

    switch clientMessage.Type {
    case "subscribe":
        var channels []string
        if err := json.Unmarshal(clientMessage.Data, &channels); err != nil {
            return
        }

        for _, channel := range channels {
            c.Subscribe(channel)
        }

    case "unsubscribe":
        var channels []string
        if err := json.Unmarshal(clientMessage.Data, &channels); err != nil {
            return
        }

        for _, channel := range channels {
            c.Unsubscribe(channel)
        }

    case "set_filters":
        var filters struct {
            Filters []string `json:"filters"`
        }
        if err := json.Unmarshal(clientMessage.Data, &filters); err != nil {
            return
        }

        c.mu.Lock()
        c.Filters = filters.Filters
        c.mu.Unlock()

    default:
        // Handle custom message types by notifying the hub
        // We could implement a message handler system here
    }
}
