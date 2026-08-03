package websocket

import "sync"

type Client struct {
	hub  *Hub
	send chan []byte
	mu   sync.Mutex
	done bool
}

func NewClient(hub *Hub) *Client {
	return &Client{
		hub:  hub,
		send: make(chan []byte, 256),
	}
}

func (c *Client) SendChannel() <-chan []byte {
	return c.send
}

// safeSend returns false if the client is closed or too slow to consume.
func (c *Client) safeSend(data []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.done {
		return false
	}
	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}

// close marks the client done and releases the send channel.
func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.done {
		c.done = true
		close(c.send)
	}
}
