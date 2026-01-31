# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

EnvisaMon is a Go application that connects to an EnvisaLink module via the Third Party Interface (TPI). It's a monitoring and logging system that maintains persistent TCP connections to alarm panels, handles authentication, and logs all TPI protocol messages to disk for analysis.

## Commands

### Building
```bash
go build -o envisaMon
```

### Running
```bash
# Set required environment variable first
export ENVISALINK_TPI_KEY="your_password"

# Run with default port (4025)
./envisaMon 192.168.1.50

# Run with custom port
./envisaMon 192.168.1.50 4026

# Debug mode (logs to stdout)
./envisaMon -m -l 192.168.1.50
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./tpi

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

## Architecture

### Core Components

**Main Package (`envisaMon.go`):**
- Entry point and orchestration layer
- Handles dual logging setup with lumberjack for log rotation (logs/tpi-messages.log and logs/application.log)
- Manages signal handling for graceful shutdown (SIGINT, SIGTERM)
- Implements auto-reconnect loop with infinite retries
- Command-line flag parsing for `-m` (print messages), `-l` (print app logs), `-u` (deduplicate)

**TPI Package (`tpi/`):**
- `client.go`: Core TPI client with connection management
  - Implements exponential backoff (1s to 60s) for reconnection
  - Handles TCP connection lifecycle (Connect/ReadLoop/Close)
  - Authentication flow: reads "Login:" prompt, sends password with `\r`, validates "OK"/"FAILED" response
  - Message deduplication option to filter consecutive identical messages
  - Uses separate loggers for TPI messages (raw, no timestamp) vs application events
- `errors.go`: Custom error types for error handling semantics
  - `AuthError`: Fatal authentication failures (no retry)
  - `ConnectionError`: Network issues (retry with backoff)
  - `TimeoutError`: Read/write timeouts (retry with backoff)

### Key Design Patterns

**Dual Logging System:**
The application maintains strict separation between TPI protocol messages (raw data) and operational logs (application events). The TPI logger has NO timestamp/prefix (flags=0) to preserve raw protocol data, while the application logger includes standard log timestamps.

**Resilient Connection Handling:**
The main loop in `envisaMon.go:54-68` continuously calls `Connect()` and `ReadLoop()` in sequence. Connection failures trigger exponential backoff before retry. Authentication errors also trigger retry (though typically these indicate incorrect credentials).

**Authentication Flow:**
The TPI protocol requires reading a "Login:" prompt, responding with password followed by `\r` (carriage return), then validating the response is "OK". Auth timeout is set to 10 seconds during this phase, then cleared for ongoing message reading.

## Workflow
- Do not use git commands (commits, etc.) unless explicitly requested