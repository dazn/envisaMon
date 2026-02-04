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
./envisaMon [options] <ip-address>[:port] [<url>]
```

### Arguments

*   `<ip-address>[:port]`: **(Required)** The IP address of the EnvisaLink module. Optionally include the port (e.g., `192.168.1.50:4026`). Defaults to port `4025` if omitted.
*   `<url>`: **(Optional)** The destination HTTPS URL for reporting events (e.g., `https://events.example.com/api/ingest`). Remote reporting is only active if this URL is provided and the `ALARM_MON_API_KEY` environment variable is set.

### Options

*   `-m`: Print raw TPI messages to standard output (stdout) in addition to the log file.
*   `-l`: Print application operational logs to standard output (stdout) in addition to the log file.
*   `-u`: Deduplicate consecutive identical TPI messages.

### Examples

Connect and log locally only:
```bash
./envisaMon 192.168.1.50
```

Connect to default port 4025 and report to an API:
```bash
./envisaMon 192.168.1.50 https://api.myserver.com/events
```

Connect to a custom port and report:
```bash
./envisaMon 192.168.1.50:4026 https://api.myserver.com/webhook
```

Connect, see logs in console, and report:
```bash
./envisaMon -m -l 192.168.1.50 https://api.myserver.com/events
```

## REST API Reporting (Optional)

EnvisaMon can report all TPI messages and application events to a REST API endpoint via HTTPS POST requests. This feature is enabled only if both a destination URL is provided on the command line and the `ALARM_MON_API_KEY` is set.

### Authentication

To authenticate with your REST API, set the `ALARM_MON_API_KEY` environment variable.

```bash
export ALARM_MON_API_KEY="your_api_key_here"
```

### JSON Payload

The reporting API expects a JSON object with the following structure:

```json
{
  "event_id": "uuid-string",
  "event_unixtime": "1678886400",
  "event_message": "The log message content",
  "message_type": "TPI" | "Application",
  "system_id": "192.168.1.50:4025"
}
```

*   `event_id`: A unique UUID v4 for the event.
*   `event_unixtime`: The Unix timestamp when the event occurred.
*   `event_message`: The log content (timestamps are stripped from Application logs).
*   `message_type`: Indicates the source of the log ("TPI" for raw device messages, "Application" for internal app logs).
*   `system_id`: The configured EnvisaLink host and port.

## Logging

The application maintains two distinct log files in the `./logs` directory. These files are automatically rotated and compressed.

### 1. TPI Messages Log (`logs/tpi-messages.log`)
Contains the raw, unprocessed ASCII data received from the EnvisaLink module.
*   **Format:** Raw message text only (no added timestamps or prefixes).
*   **Rotation:** Rotates when reaching 5MB. Retains the 3 most recent log files.

### 2. Application Log (`logs/application.log`)
Contains operational events such as connection attempts, authentication status, and errors.
*   **Format:** Standard log format with date and time (e.g., `2026/01/27 10:00:00 INFO: Connected...`).
*   **Rotation:** Rotates when reaching 5MB. Retains the 3 most recent log files.

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
