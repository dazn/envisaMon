package main

import (
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantConfig  *Config
		wantErr     bool
		errContains string
		wantUsage   bool
	}{
		{
			name: "valid IP only",
			args: []string{"192.168.1.100"},
			wantConfig: &Config{
				EnvisaLinkIP:   "192.168.1.100",
				EnvisaLinkPort: 4025,
			},
			wantErr: false,
		},
		{
			name: "valid IP and port",
			args: []string{"192.168.1.100:4026"},
			wantConfig: &Config{
				EnvisaLinkIP:   "192.168.1.100",
				EnvisaLinkPort: 4026,
			},
			wantErr: false,
		},
		{
			name: "valid IP and URL",
			args: []string{"192.168.1.100", "https://events.example.com/api"},
			wantConfig: &Config{
				EnvisaLinkIP:    "192.168.1.100",
				EnvisaLinkPort:  4025,
				DestinationURL:  "https://events.example.com/api",
				DestinationPath: "/api",
			},
			wantErr: false,
		},
		{
			name: "valid IP, port, and URL",
			args: []string{"192.168.1.100:4026", "https://events.example.com:8080/webhook"},
			wantConfig: &Config{
				EnvisaLinkIP:    "192.168.1.100",
				EnvisaLinkPort:  4026,
				DestinationURL:  "https://events.example.com:8080/webhook",
				DestinationPath: "/webhook",
			},
			wantErr: false,
		},
		{
			name: "flags and valid arguments",
			args: []string{"-m", "-l", "-u", "192.168.1.100"},
			wantConfig: &Config{
				EnvisaLinkIP:     "192.168.1.100",
				EnvisaLinkPort:   4025,
				PrintTPIMessages: true,
				PrintAppLog:      true,
				Deduplicate:      true,
			},
			wantErr: false,
		},
		{
			name:        "no arguments",
			args:        []string{},
			wantErr:     true,
			errContains: "expected 1 or 2 positional arguments",
			wantUsage:   true,
		},
		{
			name:        "too many arguments",
			args:        []string{"192.168.1.100", "https://api.com", "extra"},
			wantErr:     true,
			errContains: "expected 1 or 2 positional arguments",
			wantUsage:   true,
		},
		{
			name:        "invalid port",
			args:        []string{"192.168.1.100:abc"},
			wantErr:     true,
			errContains: "invalid port number",
			wantUsage:   true,
		},
		{
			name:        "port out of range",
			args:        []string{"192.168.1.100:70000"},
			wantErr:     true,
			errContains: "port must be between 1 and 65535",
			wantUsage:   true,
		},
		{
			name:        "invalid URL scheme",
			args:        []string{"192.168.1.100", "http://api.com"},
			wantErr:     true,
			errContains: "URL scheme must be 'https'",
			wantUsage:   true,
		},
		{
			name:    "help flag",
			args:    []string{"-h"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			usageCalled := false
			fs.Usage = func() { usageCalled = true }

			got, err := parseConfig(fs, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.name == "help flag" {
					if err != flag.ErrHelp {
						t.Errorf("parseConfig() error = %v, want flag.ErrHelp", err)
					}
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseConfig() error = %v, want error containing %v", err, tt.errContains)
				}
				if tt.wantUsage && !usageCalled {
					t.Errorf("parseConfig() usage was not called, want it to be called")
				}
				if !tt.wantUsage && usageCalled && tt.name != "help flag" {
					t.Errorf("parseConfig() usage was called unexpectedly")
				}
				return
			}
			if !reflect.DeepEqual(got, tt.wantConfig) {
				t.Errorf("parseConfig() = %+v, want %+v", got, tt.wantConfig)
			}
		})
	}
}

func TestSetupLogging(t *testing.T) {
	config := &Config{
		EnvisaLinkIP:   "127.0.0.1",
		EnvisaLinkPort: 4025,
	}

	// Create a temp directory for logs
	tmpDir, err := os.MkdirTemp("", "envisaMon_test_logs")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Since setupLogging has hardcoded "./logs", we might need to change directory or mock it.
	// For now, let's just test that it runs without error if the directory exists.
	// In a real scenario, we'd refactor setupLogging to accept a base path.
	
	// Temporarily change working directory to temp dir
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	tpiLogger, appLogger, err := setupLogging(config)
	if err != nil {
		t.Errorf("setupLogging() error = %v", err)
	}
	if tpiLogger == nil || appLogger == nil {
		t.Error("setupLogging() returned nil loggers")
	}

	// Write something to trigger file creation
	tpiLogger.Println("test tpi message")
	appLogger.Println("test app message")

	// Verify log files were created
	if _, err := os.Stat("logs/tpi-messages.log"); os.IsNotExist(err) {
		t.Error("tpi-messages.log was not created")
	}
	if _, err := os.Stat("logs/application.log"); os.IsNotExist(err) {
		t.Error("application.log was not created")
	}
}
