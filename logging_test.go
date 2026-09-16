package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type logWriterFunc func([]byte) (int, error)

func (f logWriterFunc) Write(p []byte) (int, error) { return f(p) }

func TestLogFanout(t *testing.T) {
	fileErr := errors.New("file failed")
	remoteErr := errors.New("remote failed")
	stdoutErr := errors.New("stdout failed")
	tests := []struct {
		name  string
		errs  [3]error
		short bool
	}{
		{name: "success"},
		{name: "first failure", errs: [3]error{fileErr, nil, nil}},
		{name: "middle failure", errs: [3]error{nil, remoteErr, nil}},
		{name: "last failure", errs: [3]error{nil, nil, stdoutErr}},
		{name: "multiple failures", errs: [3]error{fileErr, nil, stdoutErr}},
		{name: "all failures", errs: [3]error{fileErr, remoteErr, stdoutErr}},
		{name: "short write", short: true},
		{name: "short write and error", errs: [3]error{fileErr, nil, nil}, short: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := []byte("complete private event\n")
			names := []string{"file", "remote", "stdout"}
			var order []string
			var diagnostics bytes.Buffer
			var destinations []logDestination
			for i, name := range names {
				destinations = append(destinations, logDestination{name, logWriterFunc(func(p []byte) (int, error) {
					order = append(order, name)
					if !bytes.Equal(p, entry) {
						t.Errorf("%s received %q, want %q", name, p, entry)
					}
					if tt.short && i == 0 {
						return len(p) - 1, tt.errs[i]
					}
					if tt.errs[i] != nil {
						return 0, tt.errs[i]
					}
					return len(p), nil
				})})
			}
			diagnosticWriter := logWriterFunc(func(p []byte) (int, error) {
				if !reflect.DeepEqual(order, names) {
					t.Errorf("diagnostics written before all destinations: %v", order)
				}
				return diagnostics.Write(p)
			})
			writer := newLogFanout("TPI", diagnosticWriter, destinations...)
			n, err := writer.Write(entry)
			if !reflect.DeepEqual(order, names) {
				t.Errorf("write order = %v, want exactly %v", order, names)
			}
			failureCount := 0
			for i, wantErr := range tt.errs {
				if i == 0 && tt.short && wantErr == nil {
					wantErr = io.ErrShortWrite
				}
				if wantErr == nil {
					continue
				}
				failureCount++
				if !errors.Is(err, wantErr) {
					t.Errorf("error = %v, want wrapped %v", err, wantErr)
				}
				label := names[i] + ": " + wantErr.Error()
				if err == nil || !strings.Contains(err.Error(), label) {
					t.Errorf("error = %v, want label %q", err, label)
				}
				if !strings.Contains(diagnostics.String(), "TPI logging: "+label) {
					t.Errorf("missing labelled diagnostic: %q", diagnostics.String())
				}
			}
			if failureCount == 0 {
				if n != len(entry) || err != nil {
					t.Errorf("Write = (%d, %v), want (%d, nil)", n, err, len(entry))
				}
			} else if n != 0 {
				t.Errorf("Write count = %d, want 0 on failure", n)
			}
			if strings.Count(diagnostics.String(), "\n") != failureCount || strings.Contains(diagnostics.String(), "private event") {
				t.Errorf("unexpected diagnostics: %q", diagnostics.String())
			}
		})
	}
}

func TestLogFanoutPrintlnDiagnostics(t *testing.T) {
	for _, stream := range []string{"TPI", "Application"} {
		for _, diagnosticFails := range []bool{false, true} {
			t.Run(stream+"/diagnosticFails="+strconv.FormatBool(diagnosticFails), func(t *testing.T) {
				var diagnostics bytes.Buffer
				eventCalls, diagnosticCalls := 0, 0
				writer := newLogFanout(stream, logWriterFunc(func(p []byte) (int, error) {
					diagnosticCalls++
					diagnostics.Write(p)
					if diagnosticFails {
						return 0, errors.New("stderr failed")
					}
					return len(p), nil
				}), logDestination{"file", logWriterFunc(func(p []byte) (int, error) {
					eventCalls++
					return 0, errors.New("disk full")
				})})
				log.New(writer, "", 0).Println("private event")
				if eventCalls != 1 || diagnosticCalls != 1 {
					t.Errorf("event calls = %d, diagnostic calls = %d; want 1 each", eventCalls, diagnosticCalls)
				}
				if !strings.Contains(diagnostics.String(), stream+" logging: file: disk full") || strings.Contains(diagnostics.String(), "private event") {
					t.Errorf("unexpected diagnostics: %q", diagnostics.String())
				}
			})
		}
	}
}

func loggingTestDirectory(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Error(err)
		}
	})
}

func captureLoggingOutput(t *testing.T, target **os.File) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "output")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	previous := *target
	*target = file
	t.Cleanup(func() {
		*target = previous
		file.Close()
	})
	return path
}

func readLoggingOutput(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertApplicationLogFormat(t *testing.T, line string) {
	t.Helper()
	if len(line) < 20 || line[19:] != " application event\n" {
		t.Fatalf("unexpected application log format: %q", line)
	}
	if _, err := time.Parse("2006/01/02 15:04:05", line[:19]); err != nil {
		t.Errorf("invalid application timestamp: %v", err)
	}
}

func TestSetupLoggingDestinations(t *testing.T) {
	tests := []struct {
		name       string
		url        bool
		key        string
		verbose    bool
		brokenFile bool
	}{
		{name: "file failure preserves remote and stdout", url: true, key: "test-key", verbose: true, brokenFile: true},
		{name: "file failure preserves remote without verbose", url: true, key: "test-key", brokenFile: true},
		{name: "missing key disables remote", url: true, verbose: true, brokenFile: true},
		{name: "missing URL disables remote", key: "test-key", brokenFile: true},
		{name: "file only"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loggingTestDirectory(t)
			t.Setenv("ALARM_MON_API_KEY", tt.key)
			stdout := captureLoggingOutput(t, &os.Stdout)
			stderr := captureLoggingOutput(t, &os.Stderr)
			captured := make(chan Event, 4)
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var event Event
				if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
					t.Errorf("decode event: %v", err)
				}
				if r.Header.Get("X-API-Key") != tt.key {
					t.Error("incorrect API key")
				}
				select {
				case captured <- event:
				default:
					t.Error("unexpected extra event")
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(server.Close)
			config := &Config{EnvisaLinkIP: "127.0.0.1", EnvisaLinkPort: 4025, Verbose: tt.verbose}
			if tt.url {
				config.DestinationURL = server.URL
			}
			tpiLogger, appLogger, err := setupLogging(config)
			if err != nil {
				t.Fatal(err)
			}
			if tt.brokenFile {
				// Make both file paths unusable after setup succeeds. A regular
				// file in place of their parent fails even when run as root and
				// cannot be repaired by lumberjack rotating the log files.
				if err := os.Rename("logs", "original-logs"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile("logs", []byte("not a directory"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			remoteEnabled := tt.url && tt.key != ""
			for _, logger := range []*log.Logger{tpiLogger, appLogger} {
				fanout := logger.Writer().(*logFanout)
				var names []string
				for _, destination := range fanout.destinations {
					names = append(names, destination.name)
					if closer, ok := destination.writer.(io.Closer); ok && destination.name == "file" {
						t.Cleanup(func() { closer.Close() })
					}
					if reporter, ok := destination.writer.(*AsyncReporter); ok {
						t.Cleanup(func() {
							close(reporter.msgChan)
							reporter.client.CloseIdleConnections()
						})
					}
				}
				wantNames := []string{"file"}
				if remoteEnabled {
					wantNames = append(wantNames, "remote")
				}
				if tt.verbose {
					wantNames = append(wantNames, "stdout")
				}
				if !reflect.DeepEqual(names, wantNames) {
					t.Fatalf("destinations = %v, want %v", names, wantNames)
				}
			}
			tpiLogger.Println("raw TPI event")
			appLogger.Println("application event")
			if remoteEnabled {
				want := map[string]string{"TPI": "raw TPI event", "Application": "application event"}
				timer := time.NewTimer(5 * time.Second)
				defer timer.Stop()
				for range 2 {
					select {
					case event := <-captured:
						message, ok := want[event.MessageType]
						if !ok || event.EventMessage != message || event.SystemID != "127.0.0.1:4025" {
							t.Errorf("unexpected remote event: %+v", event)
						}
						delete(want, event.MessageType)
					case <-timer.C:
						t.Fatalf("timed out waiting for remote events: %v", want)
					}
				}
			}
			output := readLoggingOutput(t, stdout)
			if tt.verbose {
				if !strings.HasPrefix(output, "raw TPI event\n") {
					t.Fatalf("missing raw TPI stdout: %q", output)
				}
				assertApplicationLogFormat(t, strings.TrimPrefix(output, "raw TPI event\n"))
			} else if output != "" {
				t.Errorf("unexpected stdout with verbose disabled: %q", output)
			}
			diagnostics := readLoggingOutput(t, stderr)
			if tt.brokenFile {
				if strings.Count(diagnostics, "\n") != 2 || !strings.Contains(diagnostics, "TPI logging: file:") || !strings.Contains(diagnostics, "Application logging: file:") || strings.Contains(diagnostics, "event") {
					t.Errorf("unexpected file failure diagnostics: %q", diagnostics)
				}
			} else {
				if diagnostics != "" {
					t.Errorf("unexpected diagnostics: %q", diagnostics)
				}
				if output := readLoggingOutput(t, "logs/tpi-messages.log"); output != "raw TPI event\n" {
					t.Errorf("unexpected TPI file format: %q", output)
				}
				assertApplicationLogFormat(t, readLoggingOutput(t, "logs/application.log"))
			}
		})
	}
}

func TestSetupLoggingDirectoryFailure(t *testing.T) {
	loggingTestDirectory(t)
	if err := os.WriteFile("logs", []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	tpiLogger, appLogger, err := setupLogging(&Config{})
	if err == nil || !strings.Contains(err.Error(), "failed to create logs directory") || tpiLogger != nil || appLogger != nil {
		t.Fatalf("setupLogging = (%v, %v, %v), want nil loggers and directory error", tpiLogger, appLogger, err)
	}
}
