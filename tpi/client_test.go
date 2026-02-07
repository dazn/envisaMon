package tpi

import (
	"io"
	"log"
	"net"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name             string
		address          string
		password         string
		deduplicateLimit int
	}{
		{
			name:             "basic client creation",
			address:          "192.168.1.50:4025",
			password:         "testpass",
			deduplicateLimit: -1,
		},
		{
			name:             "client with deduplication enabled",
			address:          "10.0.0.100:4026",
			password:         "secret123",
			deduplicateLimit: 0,
		},
		{
			name:             "client with limited deduplication",
			address:          "10.0.0.100:4026",
			password:         "secret123",
			deduplicateLimit: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpiLogger := log.New(io.Discard, "", 0)
			appLogger := log.New(io.Discard, "", 0)

			client := NewClient(tt.address, tt.password, tpiLogger, appLogger, tt.deduplicateLimit)

			if client.address != tt.address {
				t.Errorf("address = %q, want %q", client.address, tt.address)
			}
			if client.password != tt.password {
				t.Errorf("password = %q, want %q", client.password, tt.password)
			}
			if client.tpiLogger != tpiLogger {
				t.Error("tpiLogger not set correctly")
			}
			if client.appLogger != appLogger {
				t.Error("appLogger not set correctly")
			}
			if client.deduplicateLimit != tt.deduplicateLimit {
				t.Errorf("deduplicateLimit = %v, want %v", client.deduplicateLimit, tt.deduplicateLimit)
			}
			if client.reconnectDelay != initialDelay {
				t.Errorf("reconnectDelay = %v, want %v", client.reconnectDelay, initialDelay)
			}
			if client.lastMessage != "" {
				t.Errorf("lastMessage = %q, want empty string", client.lastMessage)
			}
			if client.stopCh == nil {
				t.Error("stopCh not initialized")
			}
			if client.conn != nil {
				t.Error("conn should be nil on initialization")
			}
		})
	}
}

func TestClient_Close(t *testing.T) {
	tests := []struct {
		name       string
		setupConn  bool
		wantClosed bool
	}{
		{
			name:       "close with active connection",
			setupConn:  true,
			wantClosed: true,
		},
		{
			name:       "close without connection",
			setupConn:  false,
			wantClosed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(-1)
			var mockConn *mockConn

			if tt.setupConn {
				mockConn = newMockConn("")
				client.conn = mockConn
			}

			err := client.Close()
			if err != nil {
				t.Errorf("Close() returned error: %v", err)
			}

			// Verify stopCh is closed
			select {
			case <-client.stopCh:
				// Channel closed as expected
			default:
				t.Error("stopCh was not closed")
			}

			// Verify connection closed if it existed
			if tt.setupConn && !mockConn.closed {
				t.Error("connection was not closed")
			}
		})
	}
}

func TestClient_resetBackoff(t *testing.T) {
	tests := []struct {
		name          string
		initialDelay  time.Duration
		expectedDelay time.Duration
	}{
		{
			name:          "reset from max delay",
			initialDelay:  maxDelay,
			expectedDelay: initialDelay,
		},
		{
			name:          "reset from intermediate delay",
			initialDelay:  30 * time.Second,
			expectedDelay: initialDelay,
		},
		{
			name:          "reset from initial delay",
			initialDelay:  initialDelay,
			expectedDelay: initialDelay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(-1)
			client.reconnectDelay = tt.initialDelay

			client.resetBackoff()

			if client.reconnectDelay != tt.expectedDelay {
				t.Errorf("reconnectDelay = %v, want %v", client.reconnectDelay, tt.expectedDelay)
			}
		})
	}
}

func TestClient_ReadLoop(t *testing.T) {
	tests := []struct {
		name             string
		readData         string
		deduplicateLimit int
		wantMessages     []string
		wantErrType      string
		wantLogContains  string
	}{
		{
			name:             "single message",
			readData:         "message1\n",
			deduplicateLimit: -1,
			wantMessages:     []string{"message1"},
			wantErrType:      "ConnectionError",
		},
		{
			name:             "multiple messages",
			readData:         "message1\nmessage2\nmessage3\n",
			deduplicateLimit: -1,
			wantMessages:     []string{"message1", "message2", "message3"},
			wantErrType:      "ConnectionError",
		},
		{
			name:             "infinite deduplication enabled - skip duplicates",
			readData:         "msg1\nmsg1\nmsg2\nmsg2\nmsg3\n",
			deduplicateLimit: 0,
			wantMessages:     []string{"msg1", "msg2", "msg3"},
			wantErrType:      "ConnectionError",
		},
		{
			name:             "deduplication disabled - keep all",
			readData:         "msg1\nmsg1\nmsg2\n",
			deduplicateLimit: -1,
			wantMessages:     []string{"msg1", "msg1", "msg2"},
			wantErrType:      "ConnectionError",
		},
		{
			name:             "limited deduplication - ignore 1",
			readData:         "msg1\nmsg1\nmsg1\nmsg1\n",
			deduplicateLimit: 1,
			wantMessages:     []string{"msg1", "msg1"},
			wantErrType:      "ConnectionError",
		},
		{
			name:             "limited deduplication - ignore 2",
			readData:         "msg1\nmsg1\nmsg1\nmsg1\n",
			deduplicateLimit: 2,
			wantMessages:     []string{"msg1", "msg1"}, // 1st is logged, 2nd & 3rd ignored, 4th is logged
			wantErrType:      "ConnectionError",
		},
		{
			name:             "limited deduplication - complex sequence",
			readData:         "msg1\nmsg1\nmsg1\nmsg2\nmsg2\nmsg2\nmsg1\n",
			deduplicateLimit: 1,
			wantMessages:     []string{"msg1", "msg1", "msg2", "msg2", "msg1"},
			wantErrType:      "ConnectionError",
		},
		{
			name:             "connection closed (EOF)",
			readData:         "",
			deduplicateLimit: -1,
			wantMessages:     []string{},
			wantErrType:      "ConnectionError",
			wantLogContains:  "Connection closed by remote",
		},
		{
			name:             "deduplication with alternating messages",
			readData:         "msg1\nmsg2\nmsg1\nmsg2\n",
			deduplicateLimit: 0,
			wantMessages:     []string{"msg1", "msg2", "msg1", "msg2"},
			wantErrType:      "ConnectionError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client with test loggers
			tpiBuf, tpiLogger := newTestLogger()
			appBuf, appLogger := newTestLogger()

			client := NewClient(
				"192.168.1.50:4025",
				"testpass",
				tpiLogger,
				appLogger,
				tt.deduplicateLimit,
			)

			// Set up mock connection
			mockConn := newMockConn(tt.readData)
			client.conn = mockConn

			// Run ReadLoop
			err := client.ReadLoop()

			// Verify error type
			if err == nil {
				t.Fatal("ReadLoop() expected error, got nil")
			}
			if tt.wantErrType == "ConnectionError" {
				if _, ok := err.(*ConnectionError); !ok {
					t.Errorf("ReadLoop() error type = %T, want *ConnectionError", err)
				}
			}

			// Verify logged messages
			tpiOutput := tpiBuf.String()
			loggedMessages := []string{}
			if tpiOutput != "" {
				lines := strings.Split(strings.TrimRight(tpiOutput, "\n"), "\n")
				loggedMessages = lines
			}

			if len(loggedMessages) != len(tt.wantMessages) {
				t.Errorf("logged %d messages, want %d\nGot: %v\nWant: %v",
					len(loggedMessages), len(tt.wantMessages), loggedMessages, tt.wantMessages)
			} else {
				for i, want := range tt.wantMessages {
					if loggedMessages[i] != want {
						t.Errorf("message[%d] = %q, want %q", i, loggedMessages[i], want)
					}
				}
			}

			// Verify app log contains expected string
			if tt.wantLogContains != "" {
				appOutput := appBuf.String()
				if !strings.Contains(appOutput, tt.wantLogContains) {
					t.Errorf("app log should contain %q, got %q", tt.wantLogContains, appOutput)
				}
			}
		})
	}
}

func TestClient_ReadLoop_LastMessageTracking(t *testing.T) {
	// Test that lastMessage is updated correctly
	tpiBuf, tpiLogger := newTestLogger()
	_, appLogger := newTestLogger()

	client := NewClient(
		"192.168.1.50:4025",
		"testpass",
		tpiLogger,
		appLogger,
		0, // infinite deduplication enabled
	)

	mockConn := newMockConn("first\nsecond\nsecond\nthird\n")
	client.conn = mockConn

	// Run ReadLoop (will return error when EOF reached)
	_ = client.ReadLoop()

	// Verify lastMessage was updated to last unique message
	if client.lastMessage != "third" {
		t.Errorf("lastMessage = %q, want %q", client.lastMessage, "third")
	}

	// Verify only unique messages were logged
	tpiOutput := tpiBuf.String()
	loggedMessages := strings.Split(strings.TrimRight(tpiOutput, "\n"), "\n")
	expectedMessages := []string{"first", "second", "third"}

	if len(loggedMessages) != len(expectedMessages) {
		t.Errorf("logged %d messages, want %d", len(loggedMessages), len(expectedMessages))
	}
}

func TestClient_authenticate(t *testing.T) {
	tests := []struct {
		name         string
		password     string
		serverRead   string // Data the server "sends" to the client
		wantWrite    string // Data the client should send to the server
		wantErrType  string
		wantAuthFail bool
	}{
		{
			name:       "successful authentication",
			password:   "testpass",
			serverRead: "Login:\r\nOK\r\n",
			wantWrite:  "testpass\r",
		},
		{
			name:         "incorrect password",
			password:     "wrong",
			serverRead:   "Login:\r\nFAILED\r\n",
			wantWrite:    "wrong\r",
			wantErrType:  "AuthError",
			wantAuthFail: true,
		},
		{
			name:        "unexpected prompt",
			password:    "testpass",
			serverRead:  "Welcome to EnvisaLink\r\n",
			wantErrType: "AuthError",
		},
		{
			name:        "read error on login prompt",
			password:    "testpass",
			serverRead:  "", // EOF
			wantErrType: "TimeoutError",
		},
		{
			name:        "unexpected response format",
			password:    "testpass",
			serverRead:  "Login:\r\nUNKNOWN_STATUS\r\n",
			wantWrite:   "testpass\r",
			wantErrType: "AuthError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, tpiLogger := newTestLogger()
			_, appLogger := newTestLogger()

			client := NewClient(
				"127.0.0.1:4025",
				tt.password,
				tpiLogger,
				appLogger,
				-1,
			)

			mock := newMockConn(tt.serverRead)
			client.conn = mock

			err := client.authenticate()

			// Verify error behavior
			if tt.wantErrType != "" {
				if err == nil {
					t.Fatalf("authenticate() expected error type %s, got nil", tt.wantErrType)
				}
				switch tt.wantErrType {
				case "AuthError":
					if _, ok := err.(*AuthError); !ok {
						t.Errorf("error type = %T, want *AuthError", err)
					}
				case "TimeoutError":
					if _, ok := err.(*TimeoutError); !ok {
						t.Errorf("error type = %T, want *TimeoutError", err)
					}
				}
			} else if err != nil {
				t.Fatalf("authenticate() returned unexpected error: %v", err)
			}

			// Verify data written to server
			if tt.wantWrite != "" {
				gotWrite := mock.writeBuf.String()
				if gotWrite != tt.wantWrite {
					t.Errorf("wrote %q, want %q", gotWrite, tt.wantWrite)
				}
			}
		})
	}
}

func TestClient_Connect(t *testing.T) {
	// Save original dialer and restore after test
	originalDial := dialTimeout
	defer func() { dialTimeout = originalDial }()

	tests := []struct {
		name        string
		serverData  string // Data simulating server response
		dialErr     error  // Error simulating dial failure
		wantErrType string
	}{
		{
			name:       "successful connection",
			serverData: "Login:\r\nOK\r\n",
		},
		{
			name:        "dial failure",
			dialErr:     io.ErrUnexpectedEOF,
			wantErrType: "ConnectionError",
		},
		{
			name:        "auth failure",
			serverData:  "Login:\r\nFAILED\r\n",
			wantErrType: "AuthError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, tpiLogger := newTestLogger()
			_, appLogger := newTestLogger()

			client := NewClient(
				"127.0.0.1:4025",
				"testpass",
				tpiLogger,
				appLogger,
				-1,
			)

			// Mock dialTimeout
			dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
				if tt.dialErr != nil {
					return nil, tt.dialErr
				}
				return newMockConn(tt.serverData), nil
			}

			err := client.Connect()

			if tt.wantErrType != "" {
				if err == nil {
					t.Fatalf("Connect() expected error, got nil")
				}
				switch tt.wantErrType {
				case "ConnectionError":
					if _, ok := err.(*ConnectionError); !ok {
						t.Errorf("error type = %T, want *ConnectionError", err)
					}
				case "AuthError":
					if _, ok := err.(*AuthError); !ok {
						t.Errorf("error type = %T, want *AuthError", err)
					}
				}
			} else if err != nil {
				t.Errorf("Connect() returned unexpected error: %v", err)
			}

			// Verify conn state
			if err == nil {
				if client.conn == nil {
					t.Error("client.conn is nil after successful Connect")
				}
			} else {
				if client.conn != nil {
					t.Error("client.conn should be nil after failed Connect")
				}
			}
		})
	}
}