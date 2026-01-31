package tpi

import (
	"bytes"
	"io"
	"log"
	"net"
	"time"
)

// mockConn implements net.Conn for testing
type mockConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
}

// newMockConn creates a new mock connection with the given read data
func newMockConn(readData string) *mockConn {
	return &mockConn{
		readBuf:  bytes.NewBufferString(readData),
		writeBuf: &bytes.Buffer{},
		closed:   false,
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.readBuf.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.writeBuf.Write(b)
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}

func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(192, 168, 1, 50), Port: 4025}
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// newTestLogger creates a logger that writes to a buffer for testing
func newTestLogger() (*bytes.Buffer, *log.Logger) {
	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)
	return buf, logger
}

// newTestClient creates a client with test defaults
func newTestClient(deduplicate bool) *Client {
	return NewClient(
		"192.168.1.50:4025",
		"testpass",
		log.New(io.Discard, "", 0),
		log.New(io.Discard, "", 0),
		deduplicate,
	)
}
