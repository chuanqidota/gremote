package guacamole

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
)

// Client manages a connection to guacd.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
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
	return &Client{conn: conn, reader: bufio.NewReader(conn)}, nil
}

// Handshake sends the Guacamole protocol handshake (select + connection params).
func (c *Client) Handshake(protocol string, params map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	// Step 1: Send protocol selection
	if err := WriteInstruction(c.conn, "select", protocol); err != nil {
		return err
	}

	// Step 2: Read server args response (protocol version + supported params)
	argsInstr, err := ReadInstruction(c.reader)
	if err != nil {
		return fmt.Errorf("failed to read args: %w", err)
	}
	if argsInstr.Op != "args" {
		return fmt.Errorf("expected args instruction, got %s", argsInstr.Op)
	}
	// args: VERSION, param1, param2, ...
	protoVersion := argsInstr.Args[0]
	argNames := argsInstr.Args[1:]

	// Step 3: Send client capabilities
	width := params["width"]
	height := params["height"]
	dpi := params["dpi"]
	if width == "" {
		width = "1024"
	}
	if height == "" {
		height = "768"
	}
	if dpi == "" {
		dpi = "96"
	}
	WriteInstruction(c.conn, "size", width, height, dpi)
	WriteInstruction(c.conn, "audio") // no audio support
	WriteInstruction(c.conn, "video") // no video support
	WriteInstruction(c.conn, "image", "image/png") // declare PNG support for createImageBitmap

	// Step 4: Send connect with ALL args in the order specified by server
	// Each arg must match the args list; use empty string for unspecified params
	connectArgs := make([]string, 0, len(argsInstr.Args))
	connectArgs = append(connectArgs, protoVersion)
	for _, name := range argNames {
		if val, ok := params[name]; ok {
			connectArgs = append(connectArgs, val)
		} else {
			connectArgs = append(connectArgs, "")
		}
	}
	if err := WriteInstruction(c.conn, "connect", connectArgs...); err != nil {
		return err
	}

	// Do NOT read ready here — let the caller's Read() get it naturally.
	// guacd's ready response is the first data after connect and should be
	// forwarded directly to the browser so its Guacamole.Client processes
	// the handshake correctly.

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

// Read reads one instruction from guacd with the given deadline.
func (c *Client) Read() (*Instruction, error) {
	c.conn.SetReadDeadline(time.Now().Add(24 * time.Hour))
	return ReadInstruction(c.reader)
}

// ReadDeadline reads one instruction with a custom deadline duration.
func (c *Client) ReadDeadline(deadline time.Duration) (*Instruction, error) {
	c.conn.SetReadDeadline(time.Now().Add(deadline))
	return ReadInstruction(c.reader)
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
