package tpi

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

const (
	initialDelay = 1 * time.Second
	maxDelay     = 60 * time.Second
	multiplier   = 2.0
	authTimeout  = 10 * time.Second
)

// dialTimeout is a variable to allow mocking in tests
var dialTimeout = net.DialTimeout

// Client manages the TPI connection and message handling
type Client struct {
	address        string
	password       string
	conn           net.Conn
	tpiLogger      *log.Logger
	appLogger      *log.Logger
	stopCh         chan struct{}
	reconnectDelay time.Duration
	deduplicate    bool
	lastMessage    string
}

// NewClient creates a new TPI client
func NewClient(address, password string, tpiLogger, appLogger *log.Logger, deduplicate bool) *Client {
	return &Client{
		address:        address,
		password:       password,
		tpiLogger:      tpiLogger,
		appLogger:      appLogger,
		stopCh:         make(chan struct{}),
		reconnectDelay: initialDelay,
		deduplicate:    deduplicate,
		lastMessage:    "",
	}
}

// Connect establishes a TCP connection to the TPI server and authenticates
func (c *Client) Connect() error {
	c.reconnectWithBackoff()

	// Establish TCP connection
	c.appLogger.Printf("INFO: Connecting to %s", c.address)
	conn, err := dialTimeout("tcp", c.address, 10*time.Second)
	if err != nil {
		c.appLogger.Printf("ERROR: Failed to connect: %v", err)
		return &ConnectionError{Message: "failed to dial", Err: err}
	}

	c.conn = conn
	c.appLogger.Printf("INFO: Connected to %s", c.address)

	// Authenticate
	if err := c.authenticate(); err != nil {
		c.conn.Close()
		c.conn = nil
		return err
	}

	c.appLogger.Println("INFO: Authentication successful")
	c.resetBackoff()

	return nil
}

// authenticate handles the login flow
func (c *Client) authenticate() error {
	// Set timeout for authentication phase
	if err := c.conn.SetReadDeadline(time.Now().Add(authTimeout)); err != nil {
		return &TimeoutError{Operation: "set read deadline", Err: err}
	}

	reader := bufio.NewReader(c.conn)

	// Read "Login:" prompt
	loginPrompt, err := reader.ReadString('\n')
	if err != nil {
		return &TimeoutError{Operation: "read login prompt", Err: err}
	}

	if !strings.Contains(loginPrompt, "Login") {
		return &AuthError{Message: fmt.Sprintf("unexpected prompt: %s", strings.TrimSpace(loginPrompt))}
	}

	// Send password with carriage return
	_, err = fmt.Fprintf(c.conn, "%s\r", c.password)
	if err != nil {
		return &ConnectionError{Message: "failed to send password", Err: err}
	}

	// Read authentication response
	response, err := reader.ReadString('\n')
	if err != nil {
		return &TimeoutError{Operation: "read auth response", Err: err}
	}

	// Clear read deadline after authentication
	if err := c.conn.SetReadDeadline(time.Time{}); err != nil {
		return &TimeoutError{Operation: "clear read deadline", Err: err}
	}

	// Check authentication result
	responseTrimmed := strings.TrimSpace(response)
	if responseTrimmed == "OK" {
		return nil
	} else if responseTrimmed == "FAILED" {
		return &AuthError{Message: "incorrect password"}
	}

	return &AuthError{Message: fmt.Sprintf("unexpected response: %s", responseTrimmed)}
}

// ReadLoop reads messages from the TPI server and logs them
func (c *Client) ReadLoop() error {
	scanner := bufio.NewScanner(c.conn)

	for scanner.Scan() {
		line := scanner.Text()

		// Apply deduplication if enabled
		if c.deduplicate && line == c.lastMessage {
			continue // Skip logging this duplicate message
		}

		// Log raw line with NO timestamp, NO prefix
		c.tpiLogger.Println(line)
		c.lastMessage = line
	}

	// Check for errors
	if err := scanner.Err(); err != nil {
		c.appLogger.Printf("WARN: Read error: %v", err)
		return &ConnectionError{Message: "read error", Err: err}
	}

	// EOF reached (connection closed)
	c.appLogger.Println("WARN: Connection closed by remote")
	return &ConnectionError{Message: "connection closed", Err: nil}
}

// Close gracefully closes the connection
func (c *Client) Close() error {
	close(c.stopCh)

	if c.conn != nil {
		c.appLogger.Println("INFO: Closing connection")
		return c.conn.Close()
	}

	return nil
}

// reconnectWithBackoff implements exponential backoff for reconnection
func (c *Client) reconnectWithBackoff() {
	if c.reconnectDelay > initialDelay {
		c.appLogger.Printf("INFO: Waiting %v before reconnecting...", c.reconnectDelay)
		time.Sleep(c.reconnectDelay)
	}

	// Increase delay for next time
	c.reconnectDelay = time.Duration(float64(c.reconnectDelay) * multiplier)
	if c.reconnectDelay > maxDelay {
		c.reconnectDelay = maxDelay
	}
}

// resetBackoff resets the reconnection delay after successful connection
func (c *Client) resetBackoff() {
	c.reconnectDelay = initialDelay
}
