package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Event represents the JSON payload for the REST API
type Event struct {
	EventID       string `json:"event_id"`
	EventUnixTime string `json:"event_unixtime"`
	EventMessage  string `json:"event_message"`
	MessageType   string `json:"message_type"`
	SystemID      string `json:"system_id"`
}

// AsyncReporter implements io.Writer to intercept logs and send them to a remote API
type AsyncReporter struct {
	url            string
	apiKey         string
	systemID       string
	messageType    string
	stripTimestamp bool
	client         *http.Client
	msgChan        chan string
	errorWriter    io.Writer // Writer to log internal errors (e.g., file writer)
}

// NewAsyncReporter creates a new reporter
func NewAsyncReporter(url, systemID, messageType string, stripTimestamp bool, errorWriter io.Writer) *AsyncReporter {
	// Configure transport to skip SSL verification for self-signed certificates
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	ar := &AsyncReporter{
		url:            url,
		apiKey:         os.Getenv("ALARM_MON_API_KEY"),
		systemID:       systemID,
		messageType:    messageType,
		stripTimestamp: stripTimestamp,
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: tr,
		},
		msgChan:     make(chan string, 100), // Buffer to avoid blocking main thread
		errorWriter: errorWriter,
	}

	if ar.apiKey == "" {
		fmt.Fprintln(errorWriter, "WARN: ALARM_MON_API_KEY environment variable not set, requests may fail auth")
	}

	go ar.worker()
	return ar
}

// Write implements io.Writer. It parses the log line and queues it for sending.
func (ar *AsyncReporter) Write(p []byte) (n int, err error) {
	msg := string(p)
	
	// Queue the message non-blocking (drop if full to avoid halting application)
	select {
	case ar.msgChan <- msg:
	default:
		// Channel full, drop message or log error to errorWriter
		fmt.Fprintf(ar.errorWriter, "AsyncReporter channel full, dropping message: %s", msg)
	}
	
	return len(p), nil
}

func (ar *AsyncReporter) worker() {
	for msg := range ar.msgChan {
		ar.report(msg)
	}
}

func (ar *AsyncReporter) report(rawMsg string) {
	// Prepare message
	cleanMsg := rawMsg
	if ar.stripTimestamp {
		// Expecting "2009/01/23 01:23:23 msg..."
		// 19 chars for date/time + 1 space = 20 chars
		if len(cleanMsg) > 20 {
			cleanMsg = cleanMsg[20:]
		}
	}
	cleanMsg = strings.TrimSpace(cleanMsg)

	// Create payload
	event := Event{
		EventID:       newUUID(),
		EventUnixTime: fmt.Sprintf("%d", time.Now().Unix()),
		EventMessage:  cleanMsg,
		MessageType:   ar.messageType,
		SystemID:      ar.systemID,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(ar.errorWriter, "AsyncReporter marshal error: %v\n", err)
		return
	}

	// Send request
	req, err := http.NewRequest("POST", ar.url, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Fprintf(ar.errorWriter, "AsyncReporter request creation error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if ar.apiKey != "" {
		req.Header.Set("X-API-Key", ar.apiKey)
	}

	resp, err := ar.client.Do(req)
	if err != nil {
		fmt.Fprintf(ar.errorWriter, "AsyncReporter request error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(ar.errorWriter, "AsyncReporter API error: %d - %s\n", resp.StatusCode, string(body))
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback if crypto/rand fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	// Version 4 UUID
	b[6] = (b[6] & 0x0f) | 0x40
	// Variant 10
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
