# EnvisaLink™ UNO TPI Programmer's Document

**DEVELOPER DOCUMENTION**

**VERSION 1.08**
Sep 23, 2025

*   Minimum UNO Version: 1.0.154
*   Minimum SOLO Version: 1.0.146
*   Minimum EVL4MAX-SA Version: 1.0.220 (1.0.60 Native)
*   Minimum DUO-SA Version: 1.0.104

---

## 1.0 Overview

The EnvisaLink™ Third Party Interface (TPI) consists of a set of commands and responses designed to allow third-party command and control applications to interface directly with the EnvisaLink™ UNO hybrid panel over a TCP/IP connection.

The goal in releasing this programmer's interface is not only to allow existing home-automation software greater interaction with the UNO panel, but also to encourage the development of third-party applications on mobile platforms.

This version of the TPI applies only to products running UNO firmware which includes the UNO hybrid panel, the Envisalink 4MAX running in Standalone Mode, the SOLO RF system, and the Envisalink DUO running in Standalone Mode. The version is similar to our Honeywell TPI but has some significant differences.

Whenever we refer to the UNO product, it is implied that this will be the same as the SOLO basestation, Envisalink 4MAX in Standalone Mode, and the DUO in Standalone mode.

## 2.0 Connecting to the UNO TPI

### 2.1 TCP Connection

The UNO acts as a server for the TCP connection and the user application is the client. The UNO listens on port 4025 and will only accept one client connection on that port. Any subsequent connections will be denied.

The UNO will close the connection if the client closes its side.

To initiate a connection, the application must first start a session by establishing a TCP socket. Once established the TPI will send a “Login” prompt.

The client should then, within 10 seconds, issue the login password (no username is used), followed by a CR (carriage return). The password is the same password used to log into the UNO's local page. Upon successful login, the UNO's TPI will respond with “OK”. If the password is incorrect it will respond with “FAILED” and the socket will close. If a password is not received within 10 seconds, the TPI will issue a “Timed Out” and close the TCP socket.

Once the password is accepted, the session is created and will continue until the TCP connection is dropped.

Note, as with all network communications, it is possible the TCP socket could be lost due to a network disruption, or an exception at either the client or server end. Application programmers are advised to include some handling for dropped connections. The Poll command is a useful command to test if the connection is still alive.

### 2.2 Packet Orientation

The UNO, as with many embedded TCP/IP applications, is resource constrained and cannot implement some features of the TCP. Specifically, it cannot handle out-of-order packet delivery as well as fragmented commands i.e. commands to the TPI that span more than one IP packet. The developer must make sure than any commands that are sent (see section 3.0) are contained in a single IP packet. A command that spans more than one IP packet will create an error and be ignored.

### 2.3 Concurrent Sockets

The UNO can only handle a small number of concurrent TPI sockets due to resource limitations. This is the same for all of the platforms listed in this document. As of this date the limit for UNO is 8 concurrent TPI sessions while it is less for older products like SOLO which are limited to 4.

## 3.0 Detailed Description of the Feature Set

### 3.1 Communications Protocol

All data is sent as hex ASCII codes. The format of packets from the TPI will be as follows

```
%CC,DATA$
```

All packets are encapsulated within the `%$` sentinels and it is guaranteed that these symbols will not appear within a packet. NOTE: The client **MAY** include a CR and/or LF after the `$` as these are disregarded by the host.

*   `CC` => 2 digit command code in HEX.
*   `DATA` => Arbitrary data based on the individual command

Commands to the TPI are either interpreted as keystrokes for the active partition (default 1), or follow an escaped packet format like below.

```
^CC,DATA$<CR><LF>
```

where

*   `CC` => 2 digit command code in HEX.
*   `DATA` => Arbitrary data based on the individual command

Upon successful reception of a client command the TPI will respond with:

```
^CC,EE$<CR><LF>
```

Where `CC` is the original command, and `EE` is an error/success code.

YOU MUST INCLUDE THE COMMA EVEN IF THE COMMAND HAS NO DATA.

When a character is transmitted outside of the `^$` sentinels, it will be interpreted as a keystroke if it is within the set `<0..9,#,*>` and ignored otherwise. There are no response indications from the TPI for keystrokes. Please note that keystrokes may be sent in individual IP packets and accepted, they do not have to be in the same IP packet like commands do as mentioned in section 2.2. As well, any naked CR or LF received by the TPI will be ignored so they can be safely sent which might be a requirement for some implementations (e.g. canonical input modes).

### 3.2 Application Commands (To the UNO)

| Description | Command | # of Data Bytes | Data Bytes |
| :--- | :---: | :---: | :--- |
| **Poll**<br>The TPI will respond with a Command Acknowledge code.<br><br>The Poll will also include command 0x05, Host Information Report | 00 | 0 | |
| **Change Default Partition**<br>This will change which partition keystrokes are sent to when using the virtual keypad. On power-up it defaults to 1. | 01 | 1 | Partition<br>`<1-8>` |
| **Dump Zone Timers**<br>This will dump the internal UNO Zone Timers. See TPI Host command FF. | 02 | 0 | |
| **Keypress to a Specific Partition**<br>The will send a keystroke to the panel from an arbitrary partition. Use this if you don't want to change the TPI default partition. | 03 | 1,1 | `<Partition>`,<br>`<0..9,*,#>` |
| **Bypass A Zone** | 04 | 3 | 3 Digit Zone |
| **Un-Bypass A Zone** | 05 | 3 | 3 Digit Zone |
| **Bypass All Open Zones in a Partition** | 06 | 1 | `<Partition>` |
| **Un-Bypass All Zones in a Partition** | 07 | 1 | `<Partition>` |
| **Stay-Arm Partition**<br>If the partition is READY TO ARM, this will start Exit Delay in the partition, this arms without a code. | 08 | 1 | `<Partition>` |
| **Away-Arm Partition**<br>If the partition is READY TO ARM, this will start Exit Delay in the partition, this arms without a code. | 09 | 1 | `<Partition>` |
| **Toggle/Activate Programmable Output (PGM)**<br>Depending on how the PGM is programmed, this command will either toggle the PGM or activate it (e.g. pulse mode)<br><br>**Only available on UNO** | 0A | 2 | Two Digit PGM number in decimal (01-14) |
| **Future** | 0B | | |
| **Request Initial State Dump**<br>The application can request that the TPI dump its zone, partition, and bypass state. Host commands 01, 02, and 04 | 0C | 0 | |
| **Request Host Information**<br>The application can request the TPI dump the host information command | 0D | 0 | |
| **Future** | | | |
| **Control Door Chime Feature**<br>Request to turn the global partition door chime ON/OFF. 1 = ON, 0 = OFF<br><br>**Not Available on EVL4MAX-SA and DUO-SA** | 10 | 1,1 | `<Partition>`<br>`<0,1>` |
| **Panic Alarm Request**<br>**WARNING: If the UNO is monitored this will trigger an immediate response from emergency services. Developers may choose to not implement this feature to avoid liability.**<br><br>Request either Fire, Medical, or Hold Up (PANIC) alarm by partition<br>0 = FIRE, 1 = MEDICAL EMERGENCY, 2 = Silent Police Panic<br><br>Example: Medical Emergency partition5<br><br>`^11,5,1$` | 11 | 1,1 | `<Partition>`<br>`<0,1,2>` |
| **Disarm Partition with Code**<br>Request to disarm the partition with a 4 or 6 digit code | 12 | 1,4-6 | `<Partition>`<br>`<0..9>` |

### 3.3 TPI Commands (From the UNO)

| Description | Command | # of Data Bytes | Data Bytes |
| :--- | :---: | :---: | :--- |
| **Zone State Change**<br>This command is issued whenever the UNO determines that zone change-of-state has occurred The data payload is a packed 16 byte HEX string, represeting a 128 bit bitfield. Each bit represents a zone from 1 to 128. The string is little endian and a binary 1 indicates that the zone is open/faulted.<br><br>NOTE: While the string is little-endian, the individual 8 bytes are normal big-endian, MSbit on the left.<br><br>Example: No Zones Open/Faulted<br>`00000000000000000000000000000000`<br><br>Example: All Zones Open/Faulted<br>`FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF`<br><br>Example: Zone 1 and 64 Open/Faulted<br>`01000000000000800000000000000000`<br><br>The UNO panel supports 128 zones, all other variants will only support up to 32 zones. | 01 | 32 | HEX string little endian |
| **Partition State Change**<br>This command is issued whenever the UNO determines that partition change-of-state has occurred The data payload is a packed 8 byte HEX string, representing the status bytes of each partition. The sting is little-endian, with the MSB being in the right-most partition.<br>Please see section 3.4 for a description of the partition states<br><br>Example: Partition 1 READY, no other partitions<br>`0100000000000000`<br><br>Example: Partition 1 and 3 READY, no other partitions<br>`0100010000000000` | 02 | 16 | HEX string of all partition states |
| **Realtime CID Event**<br>When a system event happens that is signaled to either the Envisalerts servers or the central monitoring station, it is also presented through this command. The CID event differs from other TPI commands as it is a binary coded decimal, not HEX.<br><br>`QXXXPPZZZ0`<br><br>Where:<br>Q = Qualifier. 1 = Event, 3 = Restoral<br>XXX = 3 digit CID code<br>PP = 2 digit Partition<br>ZZZ = Zone or User (depends on CID code)<br>0 = Always 0 (padding)<br><br>NOTE: The CID event Codes are ContactID codes. Lists of these codes are widely available but will not be reproduced here.<br><br>Example:<br><br>`3441010020`<br><br>3 = Restoral (Closing in this case)<br>441 = Armed in STAY mode<br>01 = Partition 1<br>002 = User 2 did it<br>0 = Always 0 | 03 | 10 | ASCII Event String |
| **Zone Bypass State Change**<br>This command is issued whenever the UNO determines that a zone bypass change-of-state has occurred. The data payload is a packed 16 byte HEX string, representing a 128 bit bitfield. Each bit represents a zone from 1 to 128. If the zone has a 1 then it is bypassed.<br><br>See TPI command 01 for more information | 04 | 32 | HEX string little endian |
| **Host Information Report**<br>Reports host MAC address, device type, and firmware version string<br><br>`MAC,XXX,V-----V,P`<br><br>Where:<br>MAC is 12 digit HEX string uniquely identifying the host<br>XXX is the device type, UNO, DUO, EVL, SOLO<br>Version string of variable length but generally of format<br>MAJOR, MINOR, BUILD NUMBER<br>P is the default partition for keystrokes (see Application Command 1)<br>TIME is the current local time of the panel in YYYYMMDDhhmmss<br><br>Example:<br>`%05,001C2A45F894,UNO, 01.00.117 – BETA,1,20250923183203$`<br><br>The host information report is issued after a successful login and after application command 0D. | 05 | 12,3,Variable,1 | `<MAC>`<br>`<TYPE>`<br>`<VERSION STRING>`<br>`<PARTITION>`<br>`<TIME>` |
| **Partition Trouble State Change**<br>This command is issued whenever the UNO determines that partition trouble state change has occurred The data payload is a packed 8 byte HEX string, representing the trouble bit-field of each partition. The sting is little-endian, with the MSB being in the right-most partition.<br><br>Please see section 3.5 for a description of the trouble bit field | 06 | 16 | HEX string of all partition states |
| **Zone Temperature Periodic Broadcast**<br>This command is issued periodically to give the last known temperature for RF sensors supporting the Envisacor SOLO RF format. This is a packed string of 256 characters representing the two digit HEX temperature of each zone. The temperature is in Envisacor temperature format. 00 means the zone temperature is unknown or not applicable.<br><br>Example:<br>`%07,00000000003400007A00……0000$`<br><br>Zone 6 is -14°C<br>Zone 9 is 21°C<br>All other zones are Not Applicable<br><br>**Envisacor Temperature Format:**<br>The single unsigned byte represents a temperature between -40° and 87.5°. The equation is<br><br>`ETF = (T+40)*2`<br><br>Where T is temperature in degrees Celsius.<br><br>**Note:** This command is only available on the UNO and SOLO platforms | 07 | 256 | HEX string of temperatures |
| **Partition Temperature Periodic Broadcast**<br>This command is issued periodically to give the last known temperature settings for each partition, inside and outside, in Envisacor temperature format. The UNO allows users to select specific zones that represents the nominal indoor and outdoor temperature for each partition. An example would be an EMS multi-sensor placed outside and a SOLO glass-break placed on the main floor, representing inside temperature.<br><br>`AABBCC………aabbcc`<br><br>Where:<br>AA is inside temperature of partition 1<br>BB is inside temperature of partition 2<br>CC is inside temperature of partition 3<br>aa is outside temperature of partition 1<br>bb is outside temperature of partition 2<br>cc is outside temperature of partition 3<br><br>Example:<br>`%08,7A000000000000003400000000000000$`<br><br>This means partition 1’s inside temperature is 21° and the outside temperature is -14°<br><br>**Note:** This command is only available on the UNO platforms | 08 | 32 | HEX string of temperatures |
| **Keypad Emulation – One Time Sound**<br>For applications that are trying to emulate a keypad over TCP/IP, this command allows for real-time status of the audible sounds emitted by a keypad on the KBUS (Envisacor’s hardwired bus).<br><br>Command 9 is sent when the keypad is to emit a one-time sound, the options being either a sequence of beeps (B), or a continuous tone of a certain length of time (Z).<br><br>The implementation of cadence and frequency is at the discretion of the developer but if you are trying to emulate the UNO5500 keypad, the cadence for beeps is 62.5ms on, 62.5ms off.<br><br>The HEX byte value defines the number of beeps for B option, or the length in seconds for the continuous tone, Z option.<br><br>Examples<br><br>`%09,1,B,06$`<br><br>Means that partition 1 keypads should BEEP for 6 beeps. This is a common acknowledgement tone.<br><br>`%09,8,Z,03$`<br><br>Means that partition 8 keypads should emit a solid tone for 3 seconds. This is a common error tone.<br><br>**NOTE:** The One-Time-Sound command 9 should always take precedence over the Persistent Sound command 0x0A (see below). For example, if the keypad has been told to emit a continuous tone (entry delay), a beep command should interrupt this output and then return to the continuous tone. | 09 | 1,1,2 | `<PARTITION>`<br>`[B,Z]`<br>`<VALUE>` |
| **Keypad Emulation – Persistent Sound**<br>For applications that are trying to emulate a keypad over TCP/IP, this command allows for real-time status of the audible sounds emitted by a keypad on the KBUS (Envisacor’s hardwired bus). It works in conjunction with command 9.<br><br>Command 0x0A is sent when the keypad is to emit a persistent sound which has an idle condition of either continually ON, or OFF. If option 1 is set to 0, the “buzzer” should be off between a sequence of beeps. If the option is set to 1, then the “buzzer” should be ON between a sequence of beeps.<br><br>**BEEPS** := The number of beeps, 1-F (1-15 decimal), that should be emitted. This would be the same frequency and cadence as used in command 9.<br><br>**INTERVAL** := The number of seconds between the sequence of beeps 1-F (1-15 decimal).<br>**NOTE:** If all three option fields are zero (0), that means cancel the persistent mode.<br><br>Examples<br><br>`%0A,1,1,0,0$`<br><br>Means that partition 1 keypads should emit a continuous tone until told to stop.<br><br>`%0A,1,0,0,0$`<br><br>Means that partition 1 keypads should stop any persistent sound.<br><br>`%0A,8,0,3,2$`<br><br>Means that partition 8 keypads should emit three beeps every 2 seconds until told to stop.<br><br>**NOTE:** The persistent sound OFF command maybe be sent by the panel even if no persistent sound is in place. Be aware. | 0A | 1,1,1,1 | `<PARTITION>`<br>`[0,1]`<br>`<BEEPS>`<br>`<INTERVAL>` |
| **Future** | 0B-0F | | |
| **Partition Chime**<br>This command is issued whenever the UNO determines that a zone in a partition should “chime” the keypads on the partition to indicate a change of state. The data payload is a packed 8 byte string, with the ASCII characters ‘0’ to ‘7’ indicating which partition(s), from 1-8, should chime, and with what chime type.<br><br>Examples<br><br>`%10,01000005$`<br><br>Means that partition 2 and partition 8 should chime on reception of this command. Chime type 1 for partition 2 and chime type 5 for partition 8.<br><br>**Note:** Chime types other than 1 is only available on the UNO platform | 10 | 8 | `[0..7]` |
| **Envisalink Zone Timer Dump**<br>This command contains the raw zone timers used inside the Envisalink. The dump is a 512 character packed HEX string representing 128 UINT16 (little endian) zone timers. Zone timers count down from 0xFFFF (zone is open) to 0x0000 (zone is closed too long ago to remember). Each “tick” of the zone time is actually 5 seconds so a zone timer of 0xFFFE means “5 seconds ago”. Remember, the zone timers are LITTLE ENDIAN so the above example would be transmitted as FEFF.<br><br>**Note:** The Zone Timer dump is only 64 bytes long (32 x 2 bytes) on all other platforms. | FF | 512 | HEX string of 128 little endian UINT16 words |

### 3.4 Partition Status Codes

The UNO uses abstracted partition states to provide a uniform interface across hardware platforms. The 02 command will present a list of all partition states on change. Here is the enumerated list of possible states.

*   00 – Partition is not Used/Doesn't Exist
*   01 – Ready
*   02 – Ready to Arm (Zones are Bypassed)
*   03 – Not Ready
*   04 – Armed in Stay Mode
*   05 – Armed in Away Mode
*   06 – Not Used
*   07 – Not Used
*   08 – Exit Delay
*   09 – Armed Zero Entry Delay – Away
*   0C – Entry Delay
*   11 – Partition in Alarm

NOTE: There may be other states that are “transitory”. They should be treated as BUSY from a User Interface point-of-view.

### 3.5 Major Trouble Bitfield

Major troubles are reported as an 8-bit field. Each bit represents a different issue on the panel.

| BIT | DESCRIPTION |
| :--- | :--- |
| 1 | Service Required – Major Issue, see logs |
| 2 | AC Failure |
| 3 | Wireless Device in Low-Battery Fault |
| 4 | Server Offline |
| 5 | Zone has a trouble (fault, tamper) |
| 6 | System Battery Overcurrent |
| 7 | System Bell/Siren absent/fault |
| 8 | Wireless Device in Supervisory Fault |

### 3.7 TPI Response Codes

After each application command, the TPI will respond with a response code. If multiple codes apply the highest number code is sent.

| CODE | DESCRIPTION |
| :---: | :--- |
| 0 | No Error – Command Accepted |
| 1 | Receive Buffer Overrun (a command is received while another is still being processed) |
| 2 | Unknown Command |
| 3 | Syntax Error. Data appended to the command is incorrect in some fashion |
| 4 | Receive Buffer Overflow – **NOT USED** |
| 5 | Receive State Machine Timeout (command not completed within 3 seconds) – **NOT USED** |
| 6 | The action command received cannot be completed for some reason. Examples would be trying to arm a partition that is already armed, or trying to arm a partition that is not ready. |
| 7 | Invalid User Code – An action that required a user code failed due to the code being incorrect |
| 8 | Out of Range Entry – Either the partition, zone, or PGM in a command doesn’t exist or is out of range for this device. |
