# EnvisaMon

EnvisaMon is a lightweight Go application designed to connect to an EnvisaLink™ module via the Third Party Interface (TPI). It monitors the connection, handles authentication, and logs all TPI messages to disk for analysis or audit.

## Features

- **TPI Connection:** Connects to the EnvisaLink TPI over TCP/IP (default port 4025).
- **Auto-Reconnect:** Automatically attempts to reconnect with exponential backoff if the connection is lost.
- **Dual Logging:** Separates raw TPI protocol messages from application operational logs.
- **Log Rotation:** Automatically manages log file sizes and retention.

## Prerequisites

Before running the application, you must set the EnvisaLink TPI password as an environment variable:

```bash
export ENVISALINK_TPI_KEY="your_password"
```

*Note: The password is the same one used to access the EnvisaLink's local web interface.*

## Usage

```bash
./envisaMon [options] <ip-address> [port]
```

### Arguments

*   `<ip-address>`: **(Required)** The IP address or hostname of the EnvisaLink module.
*   `[port]`: **(Optional)** The TPI port. Defaults to `4025` if not specified.

### Options

*   `-m`: Print raw TPI messages to standard output (stdout) in addition to the log file.
*   `-l`: Print application operational logs to standard output (stdout) in addition to the log file.

### Examples

Connect to default port 4025:
```bash
./envisaMon 192.168.1.50
```

Connect to a custom port:
```bash
./envisaMon 192.168.1.50 4026
```

Connect and see all logs in the console (useful for debugging):
```bash
./envisaMon -m -l 192.168.1.50
```

## Logging

The application maintains two distinct log files in the `./logs` directory. These files are automatically rotated and compressed.

### 1. TPI Messages Log (`logs/tpi-messages.log`)
Contains the raw, unprocessed ASCII data received from the EnvisaLink module.
*   **Format:** Raw message text only (no added timestamps or prefixes).
*   **Rotation:** Rotates daily (at midnight) or when reaching 100MB. Retains logs for 90 days.

### 2. Application Log (`logs/application.log`)
Contains operational events such as connection attempts, authentication status, and errors.
*   **Format:** Standard log format with date and time (e.g., `2026/01/27 10:00:00 INFO: Connected...`).
*   **Rotation:** Rotates daily (at midnight) or when reaching 100MB. Retains logs for 90 days.

## Message Structure

For detailed information on the structure and meaning of the raw TPI messages logged in `logs/tpi-messages.log`, please refer to the [EnvisaLink TPI Programmer's Document](EnvisaLinkTPI-ADEMCO-1-03.md).

Common TPI messages include:
*   `%CC,DATA$`: Commands/Data from EnvisaLink.
*   `^CC,DATA$`: Commands sent to EnvisaLink (or responses to commands).

Where `CC` is the command code. For example:
*   `00`: Virtual Keypad Update
*   `01`: Zone State Change
*   `02`: Partition State Change
*   `03`: Realtime CID Event
