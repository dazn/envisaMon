# EnvisaMon

EnvisaMon is a lightweight Go application designed to connect to an EnvisaLink™ module via the Third Party Interface (TPI). It monitors the connection, handles authentication, and logs all TPI messages to disk for analysis or audit. It also supports remote reporting, allowing you to forward events to a REST API for centralized monitoring or integration.

## Supported Devices

- **EnvisaLink 4 with Ademco/Honeywell firmware.**
- **UNO-family panels:** EyezOn UNO, SOLO, and the EnvisaLink 4MAX and DUO in standalone mode.

No configuration is needed to select the device. UNO-family firmware sends a `%05` Host Information Report after login, and EnvisaMon uses that to detect the panel.

EnvisaMon never sends arm, disarm, panic or PGM commands. The only command it sends after login is the poll described below.

## Features

- **TPI Connection:** Connects to the EnvisaLink TPI over TCP/IP (default port 4025).
- **Auto-Reconnect:** Automatically attempts to reconnect with exponential backoff if the connection is lost.
- **Connection Checks:** Polls the panel (`^00`) after 20 seconds of silence and reconnects if nothing arrives within a further 15 seconds. Panels with regular traffic, such as an Ademco EnvisaLink sending keypad updates, are not polled.
- **Dual Logging:** Separates raw TPI protocol messages from application operational logs.
- **Log Rotation:** Automatically manages log file sizes and retention.
- **Remote Reporting:** Securely forwards events to a REST API via HTTPS with API key authentication.

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

*   `-v`: Verbose output. Print both raw TPI messages and application operational logs to standard output (stdout) in addition to the log files.
*   `-u [n]`: Deduplicate consecutive identical TPI messages. Optionally specify a number `n` to ignore only `n` duplicates before logging the next one (e.g., `-u 10` logs the first instance, ignores 10 duplicates, then logs the 11th). If `n` is omitted, all subsequent duplicates are ignored.

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
./envisaMon -v 192.168.1.50 https://api.myserver.com/events
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
  "event_unixtime": "1678886400.123456",
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
Contains the raw, unprocessed ASCII data received from the EnvisaLink module. Replies to EnvisaMon's polls are left out: `^` acknowledgements, and any `%05` report after the first in a session. Rejected commands are recorded in the application log.
*   **Format:** Raw message text only (no added timestamps or prefixes).
*   **Rotation:** Rotates when reaching 5MB. Retains the 3 most recent log files.

### 2. Application Log (`logs/application.log`)
Contains operational events such as connection attempts, authentication status, and errors.
*   **Format:** Standard log format with date and time (e.g., `2026/01/27 10:00:00 INFO: Connected...`).
*   **Rotation:** Rotates when reaching 5MB. Retains the 3 most recent log files.

## Message Structure

For detailed information on the structure and meaning of the raw TPI messages logged in `logs/tpi-messages.log`, please refer to the TPI Programmer's Document for your device:

*   [Ademco/Honeywell](docs/EnvisaLinkTPI-ADEMCO-1-03.md)
*   [UNO family](docs/EnvisaLinkTPI-UNO-1-08.md)

TPI messages have the format `%CC,DATA$`, where `CC` is the command code. For example:
*   `00`: Virtual Keypad Update (Ademco only)
*   `01`: Zone State Change
*   `02`: Partition State Change
*   `03`: Realtime CID Event
*   `05`: Host Information Report (UNO family only)

### Decoding Messages From Both Panel Types

The JSON payload does not say which panel type sent a message, and some codes mean different things on each. A receiver must select its decoding by panel. A UNO-family panel's `%05` message names the device (`UNO`, `DUO`, `EVL` or `SOLO`) and is forwarded once per session, but it is not the first message: a UNO sends `%01`, `%02` and `%04` before it at login. A receiver must not decode a session's first `%02` until it knows the panel type. A session with no `%05` is Ademco.

Partition states in `%02` differ as follows:

| Code | Ademco | UNO family |
| :--- | :--- | :--- |
| `06` | Armed Instant (Zero Entry Delay - Stay) | Not used |
| `07` | Exit Delay | Not used |
| `08` | **Partition in Alarm** | Exit Delay |
| `09` | Alarm Has Occurred (Alarm in Memory) | Armed Zero Entry Delay - Away |
| `0C` | Not defined | Entry Delay |
| `10` | Armed Maximum (Zero Entry Delay - Away) | Not defined |
| `11` | Not defined | **Partition in Alarm** |
