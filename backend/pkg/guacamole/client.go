package guacamole

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Client manages a connection to guacd.
type Client struct {
	conn   net.Conn
	mu     sync.Mutex
	closed bool
}

// Connect creates a new TCP connection to guacd.
func Connect(host string, port int) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to guacd at %s: %w", addr, err)
	}
	return &Client{conn: conn}, nil
}

// Handshake sends the Guacamole protocol handshake (select + connection params).
func (c *Client) Handshake(protocol string, params map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	// Send protocol selection
	if err := WriteInstruction(c.conn, "select", protocol); err != nil {
		return err
	}

	// Read server version response
	_, err := ReadInstruction(c.conn)
	if err != nil {
		return fmt.Errorf("failed to read server version: %w", err)
	}

	// Send connection parameters in sorted order for determinism
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, key := range keys {
		if err := WriteInstruction(c.conn, key, params[key]); err != nil {
			return err
		}
	}

	// Send ready signal
	if err := WriteInstruction(c.conn, "ready"); err != nil {
		return err
	}

	// Read connection parameters from server
	for {
		instr, err := ReadInstruction(c.conn)
		if err != nil {
			return fmt.Errorf("failed to read handshake response: %w", err)
		}
		if instr.Op == "ready" {
			break
		}
	}

	return nil
}

// Write sends an instruction to guacd.
func (c *Client) Write(op string, args ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("client is closed")
	}
	return WriteInstruction(c.conn, op, args...)
}

// Read reads one instruction from guacd.
// Uses a deadline to avoid blocking forever if Close is called concurrently.
func (c *Client) Read() (*Instruction, error) {
	c.conn.SetReadDeadline(time.Now().Add(24 * time.Hour))
	return ReadInstruction(c.conn)
}

// Close closes the connection to guacd.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	// Set a short deadline to unblock any pending Read
	c.conn.SetDeadline(time.Now())
	return c.conn.Close()
}
