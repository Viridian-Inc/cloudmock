package cloudmock

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	maxBufferSize    = 100
	reconnectDelay   = 5 * time.Second
	connectTimeout   = 3 * time.Second
	writeTimeout     = 2 * time.Second
)

// event is the wire format sent to the devtools source server.
type event struct {
	Type      string         `json:"type"`
	Data      map[string]any `json:"data"`
	Source    string         `json:"source"`
	Runtime   string         `json:"runtime"`
	Timestamp int64          `json:"timestamp"`
}

// connection is a TCP JSON-line client that connects to the devtools source server.
// It auto-reconnects every 5 seconds and buffers up to 100 messages when disconnected.
// All operations are non-blocking and goroutine-safe.
type connection struct {
	host    string
	port    int
	appName string

	mu        sync.Mutex
	conn      net.Conn
	connected bool
	closed    bool
	buffer    []string

	done chan struct{}
}

func newConnection(host string, port int, appName string) *connection {
	c := &connection{
		host:    host,
		port:    port,
		appName: appName,
		buffer:  make([]string, 0, maxBufferSize),
		done:    make(chan struct{}),
	}
	go c.connectLoop()
	return c
}

func (c *connection) connectLoop() {
	// Attempt first connection immediately
	c.tryConnect()

	for {
		select {
		case <-c.done:
			return
		case <-time.After(reconnectDelay):
			c.mu.Lock()
			needsConnect := !c.connected && !c.closed
			c.mu.Unlock()
			if needsConnect {
				c.tryConnect()
			}
		}
	}
}

func (c *connection) tryConnect() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	addr := net.JoinHostPort(c.host, fmt.Sprintf("%d", c.port))
	conn, err := net.DialTimeout("tcp", addr, connectTimeout)
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		conn.Close()
		return
	}

	c.conn = conn
	c.connected = true

	// Flush buffered messages
	remaining := make([]string, 0)
	for _, msg := range c.buffer {
		if err := c.writeLocked(msg); err != nil {
			remaining = append(remaining, msg)
			break
		}
	}
	c.buffer = remaining

	// Start a goroutine to detect disconnection by reading
	go c.readUntilClose(conn)
}

func (c *connection) readUntilClose(conn net.Conn) {
	buf := make([]byte, 256)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			c.mu.Lock()
			if c.conn == conn {
				c.connected = false
				c.conn = nil
				conn.Close()
			}
			c.mu.Unlock()
			return
		}
	}
}

// send serializes and sends an event to the devtools server.
// Non-blocking: buffers the message if not connected.
func (c *connection) send(typ string, data map[string]any) {
	e := event{
		Type:      typ,
		Data:      data,
		Source:    c.appName,
		Runtime:   "go",
		Timestamp: time.Now().UnixMilli(),
	}

	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	msg := string(b) + "\n"

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	if c.connected && c.conn != nil {
		if err := c.writeLocked(msg); err == nil {
			return
		}
		// Write failed; fall through to buffer
		c.connected = false
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
	}

	// Buffer up to maxBufferSize messages
	if len(c.buffer) < maxBufferSize {
		c.buffer = append(c.buffer, msg)
	}
}

// writeLocked writes a message to the TCP connection. Must be called with mu held.
func (c *connection) writeLocked(msg string) error {
	if c.conn == nil {
		return fmt.Errorf("no connection")
	}
	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	_, err := c.conn.Write([]byte(msg))
	return err
}

// close shuts down the connection and stops reconnection attempts.
func (c *connection) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}
	c.closed = true
	close(c.done)

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connected = false
	c.buffer = nil
}
