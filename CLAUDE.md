# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

EnvisaMon is a Go application that connects to an EnvisaLink module via the Third Party Interface (TPI). It's a monitoring and logging system that maintains persistent TCP connections to alarm panels, handles authentication, and logs all TPI protocol messages to disk for analysis.

## Commands

### Building
```bash
# Local build
go build -o envisaMon

# Cross-compile for MIPS Big Endian, Soft Float
GOOS=linux GOARCH=mips GOMIPS=softfloat go build -ldflags="-s -w" -o build/envisaMon-linux-mips-softfloat .
```

### Running
```bash
# Set required environment variable first
export ENVISALINK_TPI_KEY="your_password"

# Run with default port (4025)
./envisaMon 192.168.1.50 https://your-endpoint.com/restPath

# Run with custom port
./envisaMon 192.168.1.50:4026 https://your-endpoint.com/restPath

# Debug mode (logs to stdout)
./envisaMon -m -l 192.168.1.50 https://your-endpoint.com/restPath
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./tpi

# Run specific test function
go test ./tpi -run TestConnect

# Run with verbose output
go test -v ./...
```

### Dependencies
```bash
# Download dependencies
go mod download

# Update dependencies
go mod tidy
```

### Core Components

**Main Package (`envisaMon.go`):**
- Entry point and orchestration layer
- Handles dual logging setup with lumberjack for log rotation (logs/tpi-messages.log and logs/application.log)
- Manages signal handling for graceful shutdown (SIGINT, SIGTERM)
- Implements auto-reconnect loop with infinite retries
- Command-line flag parsing for `-m` (print messages), `-l` (print app logs), `-u [n]` (deduplicate with optional limit)

**TPI Package (`tpi/`):**
- `client.go`: Core TPI client with connection management
  - Implements exponential backoff (1s to 60s) for reconnection
  - Handles TCP connection lifecycle (Connect/ReadLoop/Close)
  - Authentication flow: reads "Login:" prompt, sends password with `\r`, validates "OK"/"FAILED" response
  - Message deduplication option (enabled with `-u` flag) filters consecutive identical messages. Supports an optional limit to periodically log duplicates.
  - Uses separate loggers for TPI messages (raw, no timestamp) vs application events
  - Exposes `dialTimeout` variable for mocking in tests
- `errors.go`: Custom error types for error handling semantics
  - `AuthError`: Fatal authentication failures (no retry)
  - `ConnectionError`: Network issues (retry with backoff)
  - `TimeoutError`: Read/write timeouts (retry with backoff)
- `testutil_test.go`: Mock TCP server utilities for testing TPI protocol interactions

### Key Design Patterns

**Dual Logging System:**
The application maintains strict separation between TPI protocol messages (raw data) and operational logs (application events). The TPI logger has NO timestamp/prefix (flags=0) to preserve raw protocol data, while the application logger includes standard log timestamps.

**Resilient Connection Handling:**
The main loop in `envisaMon.go:54-68` continuously calls `Connect()` and `ReadLoop()` in sequence. Connection failures trigger exponential backoff before retry. Authentication errors also trigger retry (though typically these indicate incorrect credentials).

**Authentication Flow:**
The TPI protocol requires reading a "Login:" prompt, responding with password followed by `\r` (carriage return), then validating the response is "OK". Auth timeout is set to 10 seconds during this phase, then cleared for ongoing message reading.

**Testing Pattern:**
Tests use table-driven approach and mock TCP connections via `testutil_test.go`. The `dialTimeout` variable in `client.go` is reassigned during tests to inject mock connections. Tests verify the exact protocol sequence including login prompts, authentication responses, and message handling.

## Workflow
- Do not use git commands (commits, etc.) unless explicitly requested