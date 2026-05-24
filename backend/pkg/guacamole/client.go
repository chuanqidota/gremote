package guacamole

import (
    "fmt"
    "net"
    "sync"
)

// Client manages a connection to guacd.
type Client struct {
    conn    net.Conn
    mu      sync.Mutex
    closed  bool
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

    // Send protocol selection
    if err := WriteInstruction(c.conn, "select", protocol); err != nil {
        return err
    }

    // Read server version response
    _, err := ReadInstruction(c.conn)
    if err != nil {
        return fmt.Errorf("failed to read server version: %w", err)
    }

    // Send connection parameters
    for key, value := range params {
        if err := WriteInstruction(c.conn, key, value); err != nil {
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
        // Server may send supported instructions list
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
func (c *Client) Read() (*Instruction, error) {
    return ReadInstruction(c.conn)
}

// Conn returns the underlying net.Conn for direct I/O (used for streaming).
func (c *Client) Conn() net.Conn {
    return c.conn
}

// Close closes the connection to guacd.
func (c *Client) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.closed {
        return nil
    }
    c.closed = true
    return c.conn.Close()
}
