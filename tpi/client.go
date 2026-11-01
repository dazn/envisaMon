package tpi

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const (
	initialDelay = 1 * time.Second
	maxDelay     = 60 * time.Second
	multiplier   = 2.0
	authTimeout  = 10 * time.Second
	writeTimeout = 5 * time.Second

	// The only command the client sends after login
	pollCommand = "^00,$\r\n"

	// Host Information Report, only sent by UNO-family firmware
	hostReportPrefix = "%05,"
)

// dialTimeout is a variable to allow mocking in tests
var dialTimeout = net.DialTimeout

// pollIdle and pollTimeout are variables to allow shorter timings in tests
var (
	pollIdle    = 20 * time.Second // Silence before polling the panel
	pollTimeout = 15 * time.Second // Time allowed for the panel to reply
)

// Client manages the TPI connection and message handling
type Client struct {
	address          string
	password         string
	conn             net.Conn
	reader           *bufio.Reader
	tpiLogger        *log.Logger
	appLogger        *log.Logger
	stopCh           chan struct{}
	reconnectDelay   time.Duration
	deduplicateLimit int // -1: disabled, 0: infinite, >0: ignore n duplicates
	deduplicateCount int
	lastMessage      string
	hostReported     bool // true once the session's first host report is seen
}

// NewClient creates a new TPI client
func NewClient(address, password string, tpiLogger, appLogger *log.Logger, deduplicateLimit int) *Client {
	return &Client{
		address:          address,
		password:         password,
		tpiLogger:        tpiLogger,
		appLogger:        appLogger,
		stopCh:           make(chan struct{}),
		reconnectDelay:   initialDelay,
		deduplicateLimit: deduplicateLimit,
		deduplicateCount: 0,
		lastMessage:      "",
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
	c.reader = bufio.NewReader(conn)
	c.appLogger.Printf("INFO: Connected to %s", c.address)

	// Authenticate
	if err := c.authenticate(); err != nil {
		c.conn.Close()
		c.conn = nil
		c.reader = nil
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

	// Read "Login:" prompt
	loginPrompt, err := c.reader.ReadString('\n')
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
	response, err := c.reader.ReadString('\n')
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
	// Close this session without signaling permanent client shutdown.
	defer func(conn net.Conn) {
		if err := conn.Close(); err != nil {
			c.appLogger.Printf("WARN: Failed to close session connection: %v", err)
		}
	}(c.conn)

	conn := c.conn
	c.hostReported = false

	// Poll when the panel goes quiet. Panels with regular traffic are never polled.
	pollTimer := time.AfterFunc(pollIdle, func() { c.sendCommand(conn, pollCommand) })
	defer pollTimer.Stop()

	scanner := bufio.NewScanner(c.reader)
	scanner.Split(scanFrames)

	for {
		if err := conn.SetReadDeadline(time.Now().Add(pollIdle + pollTimeout)); err != nil {
			return &ConnectionError{Message: "failed to set read deadline", Err: err}
		}
		if !scanner.Scan() {
			break
		}
		pollTimer.Reset(pollIdle)

		line := scanner.Text()
		if !c.filterFrame(line) {
			continue
		}

		if line == c.lastMessage {
			if c.deduplicateLimit == 0 {
				// Infinite deduplication: skip all subsequent identical messages
				continue
			} else if c.deduplicateLimit > 0 {
				if c.deduplicateCount < c.deduplicateLimit {
					// Skip this duplicate but increment count
					c.deduplicateCount++
					continue
				}
				// Limit reached, we will log this one and reset count
			}
			// If limit is -1, we fall through and log everything
		} else {
			// New message, reset tracking
			c.lastMessage = line
		}
		c.deduplicateCount = 0

		// Log raw line with NO timestamp, NO prefix
		c.tpiLogger.Println(line)
	}

	// Check for errors
	if err := scanner.Err(); err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			c.appLogger.Printf("WARN: No data from panel for %v", pollIdle+pollTimeout)
			return &TimeoutError{Operation: "read", Err: err}
		}
		c.appLogger.Printf("WARN: Read error: %v", err)
		return &ConnectionError{Message: "read error", Err: err}
	}

	// EOF reached (connection closed)
	c.appLogger.Println("WARN: Connection closed by remote")
	return &ConnectionError{Message: "connection closed", Err: nil}
}

// scanFrames is a bufio.SplitFunc for the TPI stream. A frame ends at the '$'
// sentinel, which is kept, or at a newline, which is stripped. The UNO spec does
// not guarantee a line ending after '%' packets, and UNO firmware 1.01.197 ends
// them with "\n\r", so the leading CR is skipped.
func scanFrames(data []byte, atEOF bool) (advance int, token []byte, err error) {
	start := 0
	for start < len(data) && (data[start] == '\r' || data[start] == '\n') {
		start++
	}

	if i := bytes.IndexAny(data[start:], "$\n"); i >= 0 {
		end := start + i
		if data[end] == '$' {
			return end + 1, data[start : end+1], nil
		}
		return end + 1, bytes.TrimRight(data[start:end], "\r"), nil
	}

	if atEOF && start < len(data) {
		return len(data), bytes.TrimRight(data[start:], "\r"), nil
	}

	return start, nil, nil
}

// filterFrame handles frames caused by the client's own polls and reports
// whether the frame should be forwarded
func (c *Client) filterFrame(line string) bool {
	if strings.HasPrefix(line, "^") {
		_, code, _ := strings.Cut(strings.TrimSuffix(line, "$"), ",")
		if strings.Trim(code, "0") != "" {
			c.appLogger.Printf("WARN: TPI command rejected: %s", line)
		}
		return false
	}

	if strings.HasPrefix(line, hostReportPrefix) {
		// The spec has polls repeat the report, so only the first in a session is forwarded
		if c.hostReported {
			return false
		}
		c.hostReported = true
		c.appLogger.Printf("INFO: UNO-family panel detected: %s", line)
	}

	return true
}

// sendCommand writes a command in a single call so it stays in one IP packet.
// A failed write is only logged, the read deadline ends the session if it is dead.
func (c *Client) sendCommand(conn net.Conn, command string) {
	if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		c.appLogger.Printf("WARN: Failed to set write deadline: %v", err)
		return
	}
	if _, err := conn.Write([]byte(command)); err != nil {
		c.appLogger.Printf("WARN: Failed to send command %q: %v", command, err)
	}
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
