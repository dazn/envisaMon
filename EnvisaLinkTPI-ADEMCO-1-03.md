# EnvisaLink™ Vista TPI Programmer's Document

**DEVELOPER DOCUMENTATION**

**VERSION 1.03**
February 10, 2017

---

## 1.0 Overview

The EnvisaLink™ Third Party Interface (TPI) consists of a set of commands and responses designed to allow third-party command and control applications to interface directly with the EnvisaLink™ module and in turn the security system, over a TCP/IP connection.

The goal in releasing this programmer's interface is not only to allow existing home-automation software greater interaction with the EnvisaLink module, but also to encourage the development of third-party applications on mobile platforms.

This version of the TPI applies only to Envisalinks running Honeywell compatible firmware. The DSC version of the TPI is completely different and described in a separate document.

Please note there are some minor differences in the TPI depending on whether you are connecting to an Envisalink 4 as opposed to an Envisalink 3. When there are differences that should be noted in this document, they will appear in **RED**.

## 2.0 Connecting to the Envisalink™

### 2.1 Hardware Connections

Please refer to Envisalink installation or "Quick Start" document.

### 2.2 TCP Connection

The Envisalink acts as a server for the TCP connection and the user application is the client. The Envisalink listens on port 4025 and will only accept one client connection on that port. Any subsequent connections will be denied.

The Envisalink will close the connection if the client closes its side.

To initiate a connection, the application must first start a session by establishing a TCP socket. Once established the TPI will send a "Login" prompt.

The client should then, within 10 seconds, issue the login password (no username is used), followed by a CR (carriage return). The password is the same password used to log into the Envisalink's local page. Upon successful login, the Envisalink's TPI will respond with “OK”. If the password is incorrect it will respond with “FAILED” and the socket will close. If a password is not received within 10 seconds, the TPI will issue a "Timed Out" and close the TCP socket.

Once the password is accepted, the session is created and will continue until the TCP connection is dropped.

Note, as with all network communications, it is possible the TCP socket could be lost due to a network disruption, or an exception at either the client or server end. Application programmers are advised to include some handling for dropped connections. The Poll command is a useful command to test of the connection is still alive. Alternately, an application could watch for the periodic Keypad update messages that are issued every 5-10 seconds.

> **Envisalink 3 vs Envisalink 4 Difference**
>
> The Envisalink 3 only supports 6 ASCII digits for a password but the Envisalink 4 supports 10.

> **IMPORTANT: Envisalink Application Firewall**
>
> As of Envisalink 4 (1.0.102) and Envisalink 3 (1.12.180) the Envisalink has an internal firewall that will block all TPI connections that originate outside of the network segment it resides upon. This is to protect users who expose their Envisalinks to the public Internet either by mistake or ignorance. This feature can be **disabled** by changing the default user's password from "user" to any other password, 4 characters or longer.

## 3.0 Detailed Description of the Feature Set

### 3.1 Communications Protocol

All data is sent as hex ASCII codes. The format of packets from the Envisalink will be as follows:

```
%CC,DATA$
```

All packets are encapsulated within the `%` `$` sentinels and it is guaranteed that these symbols will not appear within a packet.

*   `CC` => 2 digit command code in HEX.
*   `DATA` => Arbitrary data based on the individual command

Commands to the Envisalink are either interpreted as keystrokes for the active partition (default 1), or follow an escaped packet format like below.

```
^CC,DATA$
```

where

*   `CC` => 2 digit command code in HEX.
*   `DATA` => Arbitrary data based on the individual command

Upon successful reception the Envisalink will respond with:

```
^CC,EE$
```

Where `CC` is the original command, and `EE` is an error/success code.

**YOU MUST INCLUDE THE COMMA EVEN IF THE COMMAND HAS NO DATA.**

When a character is transmitted outside of the `^$` sentinels, it will be interpreted as a keystroke if it is within the set `<0..9,#,*>` and ignored otherwise.

### 3.2 Application Commands (To the Envisalink)

| Description | Command | # of Data Bytes | Data Bytes |
| :--- | :---: | :---: | :--- |
| **Poll**<br>The TPI will respond with a Command Acknowledge code.<br><br>Note: The POLL command will also reset the Envisalink's network watchdog timer. If there is no communications with the Envisalerts servers for a period of 20 minutes, the Envisalink will reboot. Sending the TPI POLL command will reset this timer. Useful if the module is not connected to the Internet or firewalled. | 0 | 0 | |
| **Change Default Partition**<br>This will change which partition keystrokes are sent to when using the virtual keypad. On power-up it defaults to 1. | 1 | 1 | 1-8 |
| **Dump Zone Timers**<br>This will dump the internal Envisalink Zone Timers. See Envisalink command FF. | 2 | 0 | |
| **Keypress to a Specific Partition**<br>This will send a keystroke to the panel from an arbitrary partition. Use this if you don't want to change the TPI default partition. | 3 | 1,1 | `<Partition>`, `<0..9,A,B,C,D,*,#>` |

### 3.3 TPI Commands (From the Envisalink)

| Description | Command | # of Data Bytes | Data Bytes |
| :--- | :---: | :---: | :--- |
| **Virtual Keypad Update**<br>The command is issued whenever the panel wants to update the state of the keypad (User Interface).<br><br>The keypad update consists of five parts, the partition, the LED/ICON bitfield, the zone/user field, the beep field, and the ASCII Keypad Message. The latter being the two-lines of text displayed on ALPHA keypads. The fields are comma delimited.<br><br>**Partition**<br>This one byte field indicating which partition the update applies to.<br><br>**LED/ICON Bitfield**<br>This is a two byte, HEX, representation of the bitfield. When a bit is set to 1, this means the keypad would display the associated ICON or LED.<br><br>Bit Pos: Description<br>15: ARMED STAY<br>14: LOW BATTERY<br>13: FIRE<br>12: READY<br>11: Not Used<br>10: Not Used<br>09: CHECK ICON – SYSTEM TROUBLE<br>08: ALARM (FIRE ZONE)<br>07: ARMED (ZERO ENTRY DELAY)<br>06: Not Used<br>05: CHIME<br>04: BYPASS (Zones are bypassed)<br>03: AC PRESENT<br>02: ARMED AWAY<br>01: ALARM IN MEMORY<br>00: ALARM (System is in Alarm)<br><br>**USER/ZONE Field**<br>This is a one byte hex field that is sent by the panel to provide non-alpha keypads with some extra information. Depending on the state of the update it will either represent a zone, or a user. During programming it will represent the numeric.<br><br>**BEEP Field**<br>This provides information to the virtual keypad on how to "beep".<br>0 = OFF<br>1,2,3 = Beep this many times<br>4 = Continuous Fast Beep (trouble/urgency)<br>5 = Continuous Slow Beep (exit delay)<br><br>**ALPHA Field**<br>This is the two-line alphanumeric string that is displayed on Alpha keypads. It is a 32 byte ASCII string which is the concatenation of the top 16 characters and the bottom 16 Characters.<br><br>Example:<br>`01,5C08,08,00,****DISARMED**** Ready to Arm`<br><br>Which means:<br>Partition 1,<br>ICONS: LOW BAT, READY, AC PRESENT<br>Numeric: 08<br>Beeps: No beeping<br>String: `****DISARMED**** Ready to Arm` | 0 | Variable | See Description |
| **Zone State Change**<br>This command is issued whenever the Envisalink determines that zone change-of-state has occurred The data payload is a packed 8 byte HEX string, representing a 64 bit bitfield. Each bit represents a zone from 1 to 64. The string is little endian and a binary 1 indicates that the zone is open/faulted.<br><br>NOTE: While the string is little-endian, the individual 8 bytes are normal big-endian, MSbit on the left.<br><br>Example: No Zones Open/Faulted<br>`0000000000000000`<br><br>Example: No Zones Open/Faulted<br>`FFFFFFFFFFFFFFFF`<br><br>Example: Zone 1 and 64 Open/Faulted<br>`0100000000000080`<br><br>Please see Section 3.5 for important information about Zone state limitations.<br><br>**Envisalink 3 vs Envisalink 4 Difference**<br>The Envisalink 3 supports 64 zones and the Envisalink 4 supports 128. Therefore on the Envisalink 4 the HEX string is 16 HEX bytes long. | 1 | 8 (16) | HEX string little endian |
| **Partition State Change**<br>This command is issued whenever the Envisalink determines that partition change-of-state has occurred The data payload is a packed 8 byte HEX string, representing the status bytes of each partition. The string is little-endian, with the MSB being in the right-most partition.<br><br>Please see section 3.4 for a description of the partition states<br><br>Example: Partition 1 READY, no other partitions<br>`0100000000000000`<br><br>Example: Partition 1 and 3 READY, no other partitions<br>`0100010000000000` | 2 | 8 | HEX string of all partition states |
| **Realtime CID Event**<br>When a system event happens that is signaled to either the Envisalerts servers or the central monitoring station, it is also presented through this command. The CID event differs from other TPI commands as it is a binary coded decimal, not HEX.<br><br>`QXXXPPZZZ0`<br><br>Where:<br>Q = Qualifier. 1 = Event, 3 = Restoral<br>XXX = 3 digit CID code<br>PP = 2 digit Partition<br>ZZZ = Zone or User (depends on CID code)<br>0 = Always 0 (padding)<br><br>NOTE: The CID event Codes are ContactID codes. Lists of these codes are widely available but will not be reproduced here.<br><br>Example:<br>`3441010020`<br><br>3 = Restoral (Closing in this case)<br>441 = Armed in STAY mode<br>01 = Partition 1<br>002 = User 2 did it<br>0 = Always 0 | 3 | 5 | ASCII Event String |
| **Envisalink Zone Timer Dump**<br>This command contains the raw zone timers used inside the Envisalink. The dump is a 256 character packed HEX string representing 64 UINT16 (little endian) zone timers. Zone timers count down from 0xFFFF (zone is open) to 0x0000 (zone is closed too long ago to remember). Each “tick" of the zone time is actually 5 seconds so a zone timer of 0xFFFE means “5 seconds ago". Remember, the zone timers are LITTLE ENDIAN so the above example would be transmitted as FEFF.<br><br>**Envisalink 3 vs Envisalink 4 Difference**<br>The Envisalink 3 supports 64 zones and the Envisalink 4 supports 128. Therefore on the Envisalink 4 the HEX string is 512 HEX bytes long. | FF | 256 (512) | HEX string of 64 (128) little endian UINT16 words |

### 3.4 Partition Status Codes

The Envisalink uses abstracted partition states to provide a uniform interface across hardware platforms. The 02 command will present a list of all partition states on change. Here is the enumerated list of possible states (on Honeywell).

*   00 - Partition is not Used/Doesn't Exist
*   01 - Ready
*   02 - Ready to Arm (Zones are Bypasses)
*   03 - Not Ready
*   04 - Armed in Stay Mode
*   05 - Armed in Away Mode
*   06 - Armed Instant (Zero Entry Delay - Stay)
*   07 - Exit Delay (not implemented on all platforms)
*   08 - Partition is in Alarm
*   09 - Alarm Has Occurred (Alarm in Memory)
*   10 – Armed Maximum (Zero Entry Delay - Away)

### 3.5 Zone Timer/State Limitations

Ademco panels are not very sophisticated compared to other security systems and provide limited zone information to their peripherals. Ademco panels only provide real-time information of when a zone is faulted (opened) but not when it is restored (closed). A further annoying limitation is that the panel does not provide any zone information while the panel is armed.

To provide a more homogenous interface for developers, the Envisalink attempts to infer that a zone has been restored based on a few heuristics. These include the partition state, the length of time since a fault was reported, and the sequence in which faults are recorded.

The TPI developer should understand that if a zone is restored (closed), it may be 60 seconds or more before the Envisalink decides that the zone has been restored. This is unfortunate, but we have very little alternatives. As well, the TPI developer should remember that absolutely no zone information is recorded while the partition is armed.

### 3.7 TPI Response Codes

After each application command, the TPI will respond with a response code

| CODE | DESCRIPTION |
| :---: | :--- |
| 0 | No Error – Command Accepted |
| 1 | Receive Buffer Overrun (a command is received while another is still being processed) |
| 2 | Unknown Command |
| 3 | Syntax Error. Data appended to the command is incorrect in some fashion |
| 4 | Receive Buffer Overflow |
| 5 | Receive State Machine Timeout (command not completed within 3 seconds) |
