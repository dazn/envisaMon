package tpi

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"testing"
	"testing/iotest"
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
				client.reader = bufio.NewReader(mockConn)
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
			client.reader = bufio.NewReader(mockConn)

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

func TestClient_ReadLoop_ClosesSession(t *testing.T) {
	readErr := errors.New("injected read failure")
	closeErr := errors.New("injected close failure")
	for _, tt := range []struct {
		name        string
		data        string
		readErr     error
		wantMessage string
		wantErr     error
	}{
		{name: "EOF", wantMessage: "connection closed"},
		{name: "read error", readErr: readErr, wantMessage: "read error", wantErr: readErr},
		{name: "oversized token", data: strings.Repeat("x", bufio.MaxScanTokenSize+1), wantMessage: "read error", wantErr: bufio.ErrTooLong},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, cleanup := range []struct {
				name string
				err  error
			}{
				{name: "close succeeds"},
				{name: "close fails", err: closeErr},
			} {
				t.Run(cleanup.name, func(t *testing.T) {
					conn := newMockConn(tt.data)
					conn.readErr = tt.readErr
					conn.closeErr = cleanup.err
					client := newTestClient(-1)
					appBuf, appLogger := newTestLogger()
					client.appLogger = appLogger
					client.conn = conn
					client.reader = bufio.NewReader(conn)

					err := client.ReadLoop()
					if !conn.closed {
						t.Error("session connection was not closed before ReadLoop returned")
					}
					select {
					case <-client.stopCh:
						t.Error("ReadLoop closed the permanent shutdown channel")
					default:
					}
					connErr, ok := err.(*ConnectionError)
					if !ok {
						t.Fatalf("ReadLoop() error = %v (%T), want *ConnectionError", err, err)
					}
					if connErr.Message != tt.wantMessage || connErr.Err != tt.wantErr {
						t.Errorf("ReadLoop() error = %#v, want Message %q and Err %v", connErr, tt.wantMessage, tt.wantErr)
					}
					if cleanup.err != nil && !strings.Contains(appBuf.String(), "WARN: Failed to close session connection: "+cleanup.err.Error()) {
						t.Errorf("missing cleanup warning in app log: %q", appBuf.String())
					}
				})
			}
		})
	}
}

func TestClient_Connect_ClosesFailedSessionBeforeReconnect(t *testing.T) {
	first := newMockConn("Login:\r\nOK\r\nfirst-message\r\n")
	first.readErr = io.ErrUnexpectedEOF
	second := newMockConn("Login:\r\nOK\r\nsecond-message\r\nthird-message\r\n")
	connections := []*mockConn{first, second}
	dials := 0
	originalDial := dialTimeout
	t.Cleanup(func() { dialTimeout = originalDial })
	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		if dials >= len(connections) {
			t.Fatal("unexpected extra dial")
		}
		if dials == 1 && !first.closed {
			t.Error("first connection was still open when the second dial started")
		}
		conn := connections[dials]
		dials++
		return conn, nil
	}

	client := newTestClient(-1)
	tpiBuf, tpiLogger := newTestLogger()
	client.tpiLogger = tpiLogger
	for i, conn := range connections {
		if err := client.Connect(); err != nil {
			t.Fatalf("session %d Connect() returned error: %v", i+1, err)
		}
		if got := conn.writeBuf.String(); got != "testpass\r" {
			t.Errorf("session %d authentication wrote %q, want password", i+1, got)
		}
		err := client.ReadLoop()
		wantMessage := "connection closed"
		if conn.readErr != nil {
			wantMessage = "read error"
		}
		if connErr, ok := err.(*ConnectionError); !ok || connErr.Message != wantMessage || connErr.Err != conn.readErr {
			t.Errorf("session %d ReadLoop() error = %v, want %q with cause %v", i+1, err, wantMessage, conn.readErr)
		}
		if !conn.closed {
			t.Errorf("session %d connection was not closed", i+1)
		}
		select {
		case <-client.stopCh:
			t.Fatal("ReadLoop closed the permanent shutdown channel")
		default:
		}
	}
	if got, want := tpiBuf.String(), "first-message\nsecond-message\nthird-message\n"; got != want {
		t.Errorf("logged messages = %q, want %q", got, want)
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
	client.reader = bufio.NewReader(mockConn)

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
			client.reader = bufio.NewReader(mock)

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

func TestClient_Connect_ReadLoopPreservesBufferedMessages(t *testing.T) {
	for _, tt := range []struct {
		name       string
		serverData string
		laterData  string
	}{
		{
			name:       "complete messages buffered during authentication",
			serverData: "Login:\r\nOK\r\nmessage1\r\nmessage2\r\n",
		},
		{
			name:       "partial message buffered during authentication",
			serverData: "Login:\r\nOK\r\nmess",
			laterData:  "age1\r\nmessage2\r\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockConn(tt.serverData)
			originalDial := dialTimeout
			t.Cleanup(func() { dialTimeout = originalDial })
			dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
				return mock, nil
			}

			client := newTestClient(-1)
			tpiBuf, tpiLogger := newTestLogger()
			client.tpiLogger = tpiLogger
			if err := client.Connect(); err != nil {
				t.Fatalf("Connect() returned error: %v", err)
			}
			if mock.readBuf.Len() != 0 {
				t.Fatal("authentication did not read ahead through the supplied data")
			}
			// Make the remainder available only after authentication has read ahead.
			mock.readBuf.WriteString(tt.laterData)

			err := client.ReadLoop()
			if connErr, ok := err.(*ConnectionError); !ok || connErr.Message != "connection closed" || connErr.Err != nil {
				t.Errorf("ReadLoop() error = %v, want connection closed at EOF", err)
			}
			if got, want := tpiBuf.String(), "message1\nmessage2\n"; got != want {
				t.Errorf("logged messages = %q, want %q", got, want)
			}
		})
	}
}

func TestClient_Connect_ReplacesBufferedReader(t *testing.T) {
	first := newMockConn("Login:\r\nOK\r\nstale-message\r\n")
	second := newMockConn("Login:\r\nOK\r\nnew-message\r\n")
	originalDial := dialTimeout
	t.Cleanup(func() { dialTimeout = originalDial })
	connections := []*mockConn{first, second}
	dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
		if len(connections) == 0 {
			t.Fatal("unexpected extra dial")
		}
		conn := connections[0]
		connections = connections[1:]
		return conn, nil
	}

	client := newTestClient(-1)
	tpiBuf, tpiLogger := newTestLogger()
	client.tpiLogger = tpiLogger
	if err := client.Connect(); err != nil {
		t.Fatalf("first Connect() returned error: %v", err)
	}
	previousReader := client.reader
	if got, want := previousReader.Buffered(), len("stale-message\r\n"); got != want {
		t.Fatalf("buffered bytes = %d, want %d", got, want)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("closing first connection: %v", err)
	}

	if err := client.Connect(); err != nil {
		t.Fatalf("second Connect() returned error: %v", err)
	}
	if client.reader == previousReader {
		t.Fatal("second Connect() reused the previous reader")
	}
	if client.conn != second {
		t.Fatal("second Connect() did not replace the connection")
	}
	err := client.ReadLoop()
	if connErr, ok := err.(*ConnectionError); !ok || connErr.Message != "connection closed" || connErr.Err != nil {
		t.Errorf("ReadLoop() error = %v, want connection closed at EOF", err)
	}
	if got, want := tpiBuf.String(), "new-message\n"; got != want {
		t.Errorf("logged messages = %q, want %q", got, want)
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
			serverData:  "Login:\r\nFAILED\r\nstale-message\r\n",
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
			mock := newMockConn(tt.serverData)
			dialTimeout = func(network, address string, timeout time.Duration) (net.Conn, error) {
				if tt.dialErr != nil {
					return nil, tt.dialErr
				}
				return mock, nil
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
				if client.reader == nil {
					t.Error("client.reader is nil after successful Connect")
				}
			} else {
				if client.conn != nil {
					t.Error("client.conn should be nil after failed Connect")
				}
				if client.reader != nil {
					t.Error("client.reader should be nil after failed Connect")
				}
				if tt.dialErr == nil && !mock.closed {
					t.Error("connection should be closed after failed authentication")
				}
			}
		})
	}
}

func TestScanFrames(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []string
	}{
		{
			name: "packets with line endings",
			data: "%01,a$\r\n%02,b$\r\n",
			want: []string{"%01,a$", "%02,b$"},
		},
		{
			name: "packets without line endings",
			data: "%01,a$%02,b$",
			want: []string{"%01,a$", "%02,b$"},
		},
		{
			name: "mixed line endings",
			data: "^00,00$\r\n%05,x$%01,a$\n",
			want: []string{"^00,00$", "%05,x$", "%01,a$"},
		},
		{
			name: "plain lines",
			data: "OK\r\nTimed Out\n",
			want: []string{"OK", "Timed Out"},
		},
		{
			name: "blank lines are skipped",
			data: "first\n\r\n\nsecond\n",
			want: []string{"first", "second"},
		},
		{
			name: "partial packet at EOF",
			data: "%01,a$%02,b",
			want: []string{"%01,a$", "%02,b"},
		},
		{
			name: "line endings only",
			data: "\r\n\r\n",
			want: nil,
		},
	}

	for _, tt := range tests {
		for _, source := range []struct {
			name   string
			reader io.Reader
		}{
			{name: "single read", reader: strings.NewReader(tt.data)},
			{name: "one byte per read", reader: iotest.OneByteReader(strings.NewReader(tt.data))},
		} {
			t.Run(tt.name+"/"+source.name, func(t *testing.T) {
				scanner := bufio.NewScanner(source.reader)
				scanner.Split(scanFrames)

				var got []string
				for scanner.Scan() {
					got = append(got, scanner.Text())
				}
				if err := scanner.Err(); err != nil {
					t.Fatalf("scanner error: %v", err)
				}
				if strings.Join(got, "|") != strings.Join(tt.want, "|") {
					t.Errorf("frames = %q, want %q", got, tt.want)
				}
			})
		}
	}
}

const (
	testHostReport      = "%05,001C2A45F894,UNO,01.00.117,1,20250923183203$"
	testLaterHostReport = "%05,001C2A45F894,UNO,01.00.117,1,20250923183223$"
)

func TestClient_ReadLoop_UNOSession(t *testing.T) {
	tpiBuf, tpiLogger := newTestLogger()
	appBuf, appLogger := newTestLogger()
	client := NewClient("192.168.1.50:4025", "testpass", tpiLogger, appLogger, -1)

	conn := newMockConn(testHostReport + "\r\n%01,00$" +
		"^00,00$\r\n" + testLaterHostReport + "\r\n^00,06$\r\n%02,01$")
	client.conn = conn
	client.reader = bufio.NewReader(conn)

	if _, ok := client.ReadLoop().(*ConnectionError); !ok {
		t.Error("ReadLoop() should end with *ConnectionError at EOF")
	}

	if got, want := tpiBuf.String(), testHostReport+"\n%01,00$\n%02,01$\n"; got != want {
		t.Errorf("logged messages = %q, want %q", got, want)
	}
	if got := conn.writeBuf.String(); got != "" {
		t.Errorf("wrote %q, want nothing", got)
	}

	appOutput := appBuf.String()
	if !strings.Contains(appOutput, "INFO: UNO-family panel detected: "+testHostReport) {
		t.Errorf("app log should report the detected panel, got %q", appOutput)
	}
	if !strings.Contains(appOutput, "WARN: TPI command rejected: ^00,06$") {
		t.Errorf("app log should report the rejected command, got %q", appOutput)
	}
	if strings.Contains(appOutput, "^00,00$") {
		t.Errorf("app log should not report accepted commands, got %q", appOutput)
	}
}

// Bytes captured from a UNO on firmware 1.01.197, where frames end in "\n\r"
func TestClient_ReadLoop_UNOCapture(t *testing.T) {
	tpiBuf, tpiLogger := newTestLogger()
	_, appLogger := newTestLogger()
	client := NewClient("192.168.1.50:4025", "testpass", tpiLogger, appLogger, -1)

	const (
		zones      = "%01,00000000000000000000000000000000$"
		partitions = "%02,0500000000000000$"
		bypasses   = "%04,00000000000000000000000000000000$"
		troubles   = "%06,0000000000000000$"
		report     = "%05,001C2A401484,UNO,01.01.197 -  ,1,20261031223326$"
	)
	frames := []string{zones, partitions, bypasses, report, troubles, "^00,00$"}

	conn := newMockConn("Login:\r\nOK\r\n" + strings.Join(frames, "\n\r") + "\n\r")
	client.conn = conn
	client.reader = bufio.NewReader(conn)
	if err := client.authenticate(); err != nil {
		t.Fatalf("authenticate() returned error: %v", err)
	}
	conn.writeBuf.Reset()
	_ = client.ReadLoop()

	want := strings.Join([]string{zones, partitions, bypasses, report, troubles}, "\n") + "\n"
	if got := tpiBuf.String(); got != want {
		t.Errorf("logged messages = %q, want %q", got, want)
	}
	if got := conn.writeBuf.String(); got != "" {
		t.Errorf("wrote %q, want nothing", got)
	}
}

func TestClient_ReadLoop_UNOReconnectRepeatsHostReport(t *testing.T) {
	tpiBuf, tpiLogger := newTestLogger()
	_, appLogger := newTestLogger()
	client := NewClient("192.168.1.50:4025", "testpass", tpiLogger, appLogger, -1)

	sessions := []*mockConn{newMockConn(testHostReport), newMockConn(testLaterHostReport)}
	for _, conn := range sessions {
		client.conn = conn
		client.reader = bufio.NewReader(conn)
		_ = client.ReadLoop()
	}

	if got, want := tpiBuf.String(), testHostReport+"\n"+testLaterHostReport+"\n"; got != want {
		t.Errorf("logged messages = %q, want %q", got, want)
	}
}

func TestClient_ReadLoop_DeduplicatesAcrossPollReply(t *testing.T) {
	tpiBuf, tpiLogger := newTestLogger()
	_, appLogger := newTestLogger()
	client := NewClient("192.168.1.50:4025", "testpass", tpiLogger, appLogger, 0)

	conn := newMockConn(testHostReport + "%07,00$^00,00$" + testLaterHostReport + "%07,00$")
	client.conn = conn
	client.reader = bufio.NewReader(conn)
	_ = client.ReadLoop()

	if got, want := tpiBuf.String(), testHostReport+"\n%07,00$\n"; got != want {
		t.Errorf("logged messages = %q, want %q", got, want)
	}
}

func TestClient_ReadLoop_AdemcoSessionSendsNothing(t *testing.T) {
	tpiBuf, tpiLogger := newTestLogger()
	_, appLogger := newTestLogger()
	client := NewClient("192.168.1.50:4025", "testpass", tpiLogger, appLogger, -1)

	keypadUpdate := "%00,01,1C08,08,00,****DISARMED****  Ready to Arm  $"
	conn := newMockConn(keypadUpdate + "\r\n%02,0100000000000000$\r\n")
	client.conn = conn
	client.reader = bufio.NewReader(conn)
	_ = client.ReadLoop()

	if got, want := tpiBuf.String(), keypadUpdate+"\n%02,0100000000000000$\n"; got != want {
		t.Errorf("logged messages = %q, want %q", got, want)
	}
	if got := conn.writeBuf.String(); got != "" {
		t.Errorf("wrote %q, want nothing", got)
	}
}

// newPipeClient returns a client connected to the panel end of a net.Pipe, which
// supports deadlines unlike mockConn
func newPipeClient(t *testing.T, idle, timeout time.Duration) (*Client, net.Conn) {
	t.Helper()

	originalIdle, originalTimeout := pollIdle, pollTimeout
	t.Cleanup(func() { pollIdle, pollTimeout = originalIdle, originalTimeout })
	pollIdle, pollTimeout = idle, timeout

	clientConn, panel := net.Pipe()
	t.Cleanup(func() { panel.Close() })

	client := newTestClient(-1)
	client.conn = clientConn
	client.reader = bufio.NewReader(clientConn)
	return client, panel
}

// startReadLoop runs ReadLoop in the background and returns its result channel
func startReadLoop(client *Client) <-chan error {
	result := make(chan error, 1)
	go func() { result <- client.ReadLoop() }()
	return result
}

func waitForReadLoop(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ReadLoop to return")
		return nil
	}
}

func readCommand(t *testing.T, panel net.Conn) string {
	t.Helper()
	if err := panel.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("failed to set read deadline: %v", err)
	}
	buf := make([]byte, 64)
	n, err := panel.Read(buf)
	if err != nil {
		t.Fatalf("failed to read command: %v", err)
	}
	return string(buf[:n])
}

func TestClient_ReadLoop_PollsQuietPanel(t *testing.T) {
	client, panel := newPipeClient(t, 20*time.Millisecond, 2*time.Second)
	result := startReadLoop(client)

	// Each reply must re-arm the poll for the next quiet period
	for i := 0; i < 3; i++ {
		if got := readCommand(t, panel); got != pollCommand {
			t.Fatalf("poll %d = %q, want %q", i+1, got, pollCommand)
		}
		if _, err := panel.Write([]byte("^00,00$\r\n")); err != nil {
			t.Fatalf("failed to send reply: %v", err)
		}
	}

	panel.Close()
	if err := waitForReadLoop(t, result); err == nil {
		t.Fatal("ReadLoop() expected error, got nil")
	} else if _, ok := err.(*ConnectionError); !ok {
		t.Errorf("ReadLoop() error = %v (%T), want *ConnectionError", err, err)
	}
}

func TestClient_ReadLoop_ClosedConnectionIsNotATimeout(t *testing.T) {
	client, panel := newPipeClient(t, time.Second, time.Second)
	panel.Close()

	if err := client.ReadLoop(); err == nil {
		t.Fatal("ReadLoop() expected error, got nil")
	} else if _, ok := err.(*ConnectionError); !ok {
		t.Errorf("ReadLoop() error = %v (%T), want *ConnectionError", err, err)
	}
}

func TestClient_ReadLoop_TimesOutWithoutPollReply(t *testing.T) {
	client, panel := newPipeClient(t, 10*time.Millisecond, 50*time.Millisecond)
	result := startReadLoop(client)

	if got := readCommand(t, panel); got != pollCommand {
		t.Fatalf("command = %q, want %q", got, pollCommand)
	}

	err := waitForReadLoop(t, result)
	timeoutErr, ok := err.(*TimeoutError)
	if !ok {
		t.Fatalf("ReadLoop() error = %v (%T), want *TimeoutError", err, err)
	}
	if timeoutErr.Operation != "read" {
		t.Errorf("Operation = %q, want %q", timeoutErr.Operation, "read")
	}
}

func TestClient_ReadLoop_DoesNotPollActivePanel(t *testing.T) {
	client, panel := newPipeClient(t, 500*time.Millisecond, 2*time.Second)
	client.deduplicateLimit = 0 // Frames dropped as duplicates still count as traffic
	result := startReadLoop(client)

	commands := make(chan string, 1)
	go func() {
		buf := make([]byte, 64)
		if n, err := panel.Read(buf); err == nil {
			commands <- string(buf[:n])
		}
	}()

	for i := 0; i < 10; i++ {
		if _, err := panel.Write([]byte("%01,00$\r\n")); err != nil {
			t.Fatalf("failed to send frame: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	panel.Close()
	waitForReadLoop(t, result)

	select {
	case got := <-commands:
		t.Errorf("sent %q to an active panel, want nothing", got)
	default:
	}
}
