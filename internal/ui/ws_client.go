package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"nhooyr.io/websocket"
)

// WSMessage is the generic WebSocket message format.
type WSMessage struct {
	ID      string          `json:"id"`
	Action  string          `json:"action"`
	Status  string          `json:"status,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// WSClient manages the WebSocket connection to the daemon.
type WSClient struct {
	url       string
	conn      *websocket.Conn
	mu        sync.Mutex
	connected atomic.Bool
	msgID     atomic.Int64
	pending   map[string]chan WSMessage
	pendingMu sync.Mutex
	onEvent   func(WSMessage)
}

func NewWSClient(url string) *WSClient {
	return &WSClient{
		url:     url,
		pending: make(map[string]chan WSMessage),
	}
}

// SetEventHandler sets the callback for push events from the daemon.
func (c *WSClient) SetEventHandler(handler func(WSMessage)) {
	c.onEvent = handler
}

// Connect establishes the WebSocket connection.
func (c *WSClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, c.url, nil)
	if err != nil {
		c.connected.Store(false)
		return err
	}
	c.conn = conn
	c.connected.Store(true)

	// Start read loop
	go c.readLoop()

	return nil
}

// IsConnected returns true if the WebSocket is connected.
func (c *WSClient) IsConnected() bool {
	return c.connected.Load()
}

// Close closes the WebSocket connection.
func (c *WSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "bye")
		c.conn = nil
	}
	c.connected.Store(false)
}

// Send sends a request and waits for the response (with timeout).
func (c *WSClient) Send(action string, payload interface{}) (*WSMessage, error) {
	if !c.connected.Load() {
		return nil, fmt.Errorf("not connected")
	}

	id := fmt.Sprintf("%d", c.msgID.Add(1))

	var payloadData json.RawMessage
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		payloadData = data
	}

	msg := WSMessage{
		ID:      id,
		Action:  action,
		Payload: payloadData,
	}

	// Register pending response channel
	respChan := make(chan WSMessage, 1)
	c.pendingMu.Lock()
	c.pending[id] = respChan
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	// Send
	data, _ := json.Marshal(msg)
	c.mu.Lock()
	err := c.conn.Write(context.Background(), websocket.MessageText, data)
	c.mu.Unlock()
	if err != nil {
		c.connected.Store(false)
		return nil, err
	}

	// Wait for response
	select {
	case resp := <-respChan:
		if resp.Status == "error" {
			return &resp, fmt.Errorf("%s", resp.Error)
		}
		return &resp, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// SendFire sends a message without waiting for response.
func (c *WSClient) SendFire(action string, payload interface{}) {
	if !c.connected.Load() {
		return
	}

	id := fmt.Sprintf("%d", c.msgID.Add(1))
	var payloadData json.RawMessage
	if payload != nil {
		data, _ := json.Marshal(payload)
		payloadData = data
	}

	msg := WSMessage{ID: id, Action: action, Payload: payloadData}
	data, _ := json.Marshal(msg)

	c.mu.Lock()
	c.conn.Write(context.Background(), websocket.MessageText, data)
	c.mu.Unlock()
}

func (c *WSClient) readLoop() {
	for {
		_, data, err := c.conn.Read(context.Background())
		if err != nil {
			c.connected.Store(false)
			log.Printf("[WSClient] read error: %v", err)
			return
		}

		var msg WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		// Check if it's a response to a pending request
		if msg.ID != "" {
			c.pendingMu.Lock()
			ch, ok := c.pending[msg.ID]
			c.pendingMu.Unlock()
			if ok {
				ch <- msg
				continue
			}
		}

		// It's a push event
		if c.onEvent != nil {
			c.onEvent(msg)
		}
	}
}
