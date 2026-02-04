package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAsyncReporter_TimestampResolution(t *testing.T) {
	// Set API key for NewAsyncReporter
	os.Setenv("ALARM_MON_API_KEY", "test-key")
	defer os.Unsetenv("ALARM_MON_API_KEY")

	// Create a test server to capture the request
	capturedPayload := make(chan []byte, 1)
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedPayload <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	errorWriter := &strings.Builder{}
	reporter := NewAsyncReporter(ts.URL, "test-system", "TPI", false, errorWriter)
	if reporter == nil {
		t.Fatal("Failed to create AsyncReporter")
	}

	// Write a message
	testMsg := "test message"
	n, err := reporter.Write([]byte(testMsg))
	if err != nil {
		t.Errorf("Write error: %v", err)
	}
	if n != len(testMsg) {
		t.Errorf("Expected %d bytes written, got %d", len(testMsg), n)
	}

	// Wait for the message to be reported
	select {
	case payload := <-capturedPayload:
		var event Event
		if err := json.Unmarshal(payload, &event); err != nil {
			t.Fatalf("Failed to unmarshal payload: %v", err)
		}

		// Verify timestamp format (seconds.microseconds)
		if !strings.Contains(event.EventUnixTime, ".") {
			t.Errorf("Expected fractional timestamp, got: %s", event.EventUnixTime)
		}

		parts := strings.Split(event.EventUnixTime, ".")
		if len(parts) != 2 {
			t.Errorf("Invalid timestamp format: %s", event.EventUnixTime)
		} else if len(parts[1]) != 6 {
			t.Errorf("Expected 6 microsecond digits, got %d in: %s", len(parts[1]), event.EventUnixTime)
		}

		fmt.Printf("Captured event_unixtime: %s\n", event.EventUnixTime)

	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for report")
	}

	if errorWriter.Len() > 0 {
		t.Errorf("Internal reporter errors: %s", errorWriter.String())
	}
}
