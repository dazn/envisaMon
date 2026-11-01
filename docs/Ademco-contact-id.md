## ADEMCO CONTACT ID REPORTING
*(Revised 10/1/2012)* 

### Reporting Format
Contact ID reporting takes the following format: **CCCC Q EEE GG ZZZ** 

* **CCCC**: Customer (subscriber account number) 
* **Q**: Event qualifier ($E =$ new event, $R =$ restore) 
* **EEE**: Event code 
* **GG**: Partition number, 00-08 (always 00 for non-partitioned panels) 
* **ZZZ**: Zone ID number reporting the alarm (001-099), or user number for open/close reports 

> **Note:** System status messages (i.e., AC Loss, Low Battery) contain zeros in the ZZZ location. 

---

### Event Code Classifications

#### Medical Alarms
| Code | Description | Report String |
| :--- | :--- | :--- |
| **100** | Medical | Emerg-Personal Emergency-#  |
| **101** | Pendant Transmitter | Emerg-Personal Emergency-#  |
| **102** | Fail to report in | Emerg-Fail to check in-#  |

#### Fire Alarms
| Code | Description | Report String |
| :--- | :--- | :--- |
| **110** | FIRE | Fire-Fire Alarm-#  |
| **111** | SMOKE w/VERIFICATION | Fire-Fire Alarm-#  |
| **112** | Combustion | Fire-Combustion-#  |
| **113** | WATERFLOW | Fire-Water Flow-#  |
| **114** | Heat | Fire-Heat Sensor-#  |
| **115** | Pull Station | Fire-Pull Station-#  |
| **116** | Duct | Fire-Duct Sensor-#  |
| **117** | Flame | Fire-Flame Sensor-#  |
| **118** | Near Alarm | Fire-Near Alarm-#  |

#### Panic Alarms
| Code | Description | Report String |
| :--- | :--- | :--- |
| **120** | Panic Alarm | Panic-Panic-#  |
| **121** | DURESS | Panic-Duress- User 000 (or duress zone on low end panels)  |
| **122** | SILENT | Panic-Silent Panic-#  |
| **123** | AUDIBLE | Panic-Audible Panic-#  |
| **124** | Duress-Access Granted | Panic-Duress Access Grant-#  |
| **125** | Duress-Egress Granted | Panic-Duress Egress Grant-#  |

#### Burglar Alarms
| Code | Description | Report String |
| :--- | :--- | :--- |
| **130** | Burglary | Burg-Burglary-#  |
| **131** | PERIMETER | Burg-Perimeter-#  |
| **132** | INTERIOR | Burg-Interior-#  |
| **133** | 24 HR BURG (AUX) | Burg-24 Hour-#  |
| **134** | ENTRY/EXIT | Burg-Entry/Exit-#  |
| **135** | DAY/NIGHT | Burg-Day/Night-#  |
| **136** | Outdoor | Burg-Outdoor-#  |
| **137** | TAMPER | Burg-Tamper-#  |
| **138** | Near Alarm | Burg-Near Alarm-#  |
| **139** | Intrusion Verifier | Burg-Intrusion Verifier-#  |

#### General Alarms
| Code | Description | Report String |
| :--- | :--- | :--- |
| **140** | General Alarm | Alarm-General Alarm-#  |
| **141** | Polling Loop Open | Alarm-Polling Loop Open  |
| **142** | POLLING LOOP SHORT (AL) | Alarm-Polling Loop Short  |
| **143** | EXPANSION MOD FAILURE | Alarm-Exp. Module Tamper-#  |
| **144** | Sensor Tamper | Alarm-Sensor Tamper-#  |
| **145** | Expansion Module Tamper | Alarm-Exp. Module Tamper-#  |
| **146** | SILENT BURG | Burg-Silent Burglary-#  |
| **147** | Sensor Supervision | Trouble Sensor Super. -#  |

---

### 24 Hour Non-Burglary
| Code | Description | Report String |
| :--- | :--- | :--- |
| **150** | 24 HOUR (AUXILIARY) | Alarm-24 Hr. Non-Burg-#  |
| **151** | Gas Detected | Alarm-Gas Detected-#  |
| **152** | Refrigeration | Alarm-Refrigeration-#  |
| **153** | Loss of Heat | Alarm-Heating System-#  |
| **154** | Water Leakage | Alarm-Water Leakage-#  |
| **155** | Foil Break | Trouble-Foil Break-#  |
| **156** | Day Trouble | Trouble-Day Zone-#  |
| **157** | Low Bottled Gas Level | Alarm-Low Gas Level-#  |
| **158** | High Temp | Alarm-High Temperature-#  |
| **159** | Low Temp | Alarm-Low Temperature-#  |
| **161** | Loss of Air Flow | Alarm-Air Flow-#  |
| **162** | Carbon Monoxide Detected | Alarm-Carbon Monoxide-#  |
| **163** | Tank Level | Trouble-Tank Level-#  |
| **168** | High Humidity | Trouble-High Humidity-#  |
| **169** | Low Humidity | Trouble-Low Humidity-#  |

---

### Fire Supervisory
| Code | Description | Report String |
| :--- | :--- | :--- |
| **200** | FIRE SUPERVISORY | Super.-Fire Supervisory-#  |
| **201** | Low Water Pressure | Super-Low Water Pressure-#  |
| **202** | Low CO2 | Super.-Low CO2-#  |
| **203** | Gate Valve Sensor | Super.-Gate Valve-#  |
| **204** | Low Water Level | Super.-Low Water Level-#  |
| **205** | Pump Activated | Super.-Pump Activation-#  |
| **206** | Pump Failure | Super.-Pump Failure-#  |

---

### System Troubles
| Code | Description | Report String |
| :--- | :--- | :--- |
| **300** | System Trouble | Trouble-System Trouble  |
| **301** | AC LOSS | Trouble-AC Power  |
| **302** | LOW SYSTEM BATT | Trouble-Low Battery (AC is lost, battery is getting low)  |
| **303** | RAM Checksum Bad | Trouble-Bad RAM Checksum (Restore Not Applicable)  |
| **304** | ROM Checksum Bad | Trouble-Bad ROM Checksum (Restore Not Applicable)  |
| **305** | SYSTEM RESET | Trouble-System Reset (Restore Not Applicable)  |
| **306** | PANEL PROG CHANGE | Trouble-Programming Changed (Restore Not Applicable)  |
| **307** | Self-Test Failure | Trouble-Self Test Failure  |
| **308** | System Shutdown | Trouble-System Shutdown  |
| **309** | Battery Test Fail | Trouble-Battery Test Failure (Battery failed at test interval)  |
| **310** | GROUND FAULT | Trouble-Ground Fault-#  |
| **311** | Battery Missing | Trouble-Battery Missing  |
| **312** | Power Supply Overcurrent | Trouble-Pwr. Supp. Overcur.-#  |
| **313** | Engineer Reset | Status-Engineer Reset - User # (Restore Not Applicable)  |
| **314** | Primary Power Supply Failure | Trouble - Pri Pwr Supply Fail - # (Sent by UL864 Rev 9 Fire panels like FBP)  |
| **316** | System Tamper | Trouble - APL System trouble - #  |

---

### Sounder/Relay Troubles
| Code | Description | Report String |
| :--- | :--- | :--- |
| **320** | SOUNDER/RELAY | Trouble-Sounder/Relay-#  |
| **321** | BELL 1 | Trouble-Bell/Siren #1 (Event and Restore)  |
| **322** | BELL 2 | Trouble-Bell/Siren #2 (Event and Restore)  |
| **323** | Alarm Relay | Trouble-Alarm Relay  |
| **324** | Trouble Relay | Trouble-Trouble Relay  |
| **325** | Reversing Relay | Trouble-Reversing Relay  |
| **326** | Notification Appliance Ckt. #3 | Trouble-Notification Appl. Ckt#3  |
| **327** | Notification Appliance Ckt. #4 | Trouble-Notification Appl. Ckt#4  |

---

### System Peripheral Troubles
| Code | Description | Report String |
| :--- | :--- | :--- |
| **R330** | System Peripheral (E355) | Trouble-Sys. Peripheral-# From LRR, ECP data connection to panel  |
| **331** | Polling Loop Open | Trouble-Polling Loop Open  |
| **332** | POLLING LOOP SHORT | Trouble-Polling Loop Short  |
| **333** | Exp. Module Failure | Trouble-Exp. Module Fail-# ECP Path problem between panel to LRR, etc  |
| **334** | Repeater Failure | Trouble-Repeater Failure-#  |
| **335** | Local Printer Paper Out | Trouble-Printer Paper Out  |
| **336** | Local Printer Failure | Trouble-Local Printer  |
| **337** | EXP. MOD. DC LOSS | Trouble-Exp. Mod. DC Loss-#  |
| **338** | EXP. MOD. LOW BAT | Trouble-Exp. Mod. Low Batt-#  |
| **339** | EXP. MOD. RESET | Trouble-Exp. Mod. Reset-#  |
| **341** | EXP. MOD. TAMPER | Trouble-Exp. Mod. Tamper-# (5881ENHC)  |
| **342** | Exp. Module AC Loss | Trouble-Exp. Module AC Loss-#  |
| **343** | Exp. Module Self Test Fail | Trouble-Exp. Self-Test Fail-#  |
| **344** | RF Rcvr Jam Detect # | Trouble-RF Rcvr Jam Detect-#  |
| **345** | AES Encryption disabled/enabled | Trouble-AES Encryption  |

---

### Communication Troubles
| Code | Description | Report String |
| :--- | :--- | :--- |
| **350** | Communication | Trouble-Communication Failure  |
| **351** | TELCO 1 FAULT | Trouble-Phone line # 1 (Comes in as zone 1 on a V20P panel)  |
| **352** | TELCO 2 FAULT | Trouble-Phone Line # 2  |
| **353** | LR Radion Xmitter Fault (333) | Trouble-Radio Transmitter - Comm Path problem between panel and lrr (Old)  |
| **354** | FAILURE TO COMMUNICATE | Trouble-Fail to Communicate  |
| **E355** | Loss of Radio Super. (R330) | Trouble-Radio Supervision - From LRR - ECP data connection to panel  |
| **356** | Loss of Central Polling | Trouble-Central Radion Polling  |
| **357** | LRR XMTR. VSWR | Trouble-Radio Xmitter. VSWR-#  |

---

### Protection Loop
*Note: Uplink cell backup devices send zone 99 for a low battery and a zone 97 for communication failure (no response from poll). These will report as contact ID E370 (protection loop). Central station will assume that they were generated by the panel. Zones 81-86 correspond to hardwire inputs 1-6.* 

| Code | Description | Report String |
| :--- | :--- | :--- |
| **370** | Protection Loop | Trouble-Protection Loop-#  |
| **371** | Protection Loop Open | Trouble-Prot. Loop Open-#  |
| **372** | Protection Loop Short | Trouble-Prot. Loop Short-#  |
| **373** | FIRE TROUBLE | Trouble-Fire Loop-# (Supervision Loss, base tamper, Supervisory open)  |
| **374** | EXIT ERROR (BY USER) | Alarm-Exit Error-#  |
| **375** | Panic Zone Trouble | Trouble-PA Trouble-#  |
| **376** | Hold-Up Zone Trouble | Trouble-Hold-Up Trouble-#  |
| **377** | Swinger Trouble | Trouble - Swinger Trouble-#  |
| **378** | Cross-zone Trouble | Trouble - Cross Zone Trouble - # (restore not applicable)  |

---

### Sensor
| Code | Description | Report String |
| :--- | :--- | :--- |
| **380** | SENSOR TRBL - GLOBAL | Trouble-Sensor Trouble-# (zone type 5 and 19)  |
| **381** | LOSS OF SUPERVISION | Trouble-RF Sensor Super.-#  |
| **382** | LOSS OF SUPRVSN | Trouble-RPM Sensor Super.-#  |
| **383** | SENSOR TAMPER | Trouble-Sensor Tamper-# (Cover or Base)  |
| **384** | RF LOW BATTERY | Trouble-RF Sensor Battery-#  |
| **385** | SMOKE HI SENS. | Trouble-Smoke Hi Sens.-#  |
| **386** | SMOKE LO SENS. | Trouble-Smoke Lo Sens.-#  |
| **387** | INTRUSION HI SENS. | Trouble-Intrusion Hi Sens.-#  |
| **388** | INTRUSION LO SENS. | Trouble-Intrusion Lo Sens.-# (Similar to smart smoke detectors)  |
| **389** | DET. SELF TEST FAIL | Trouble-Sensor Test Fail-# (see Direct Wire #84)  |
| **391** | Sensor Watch Failure | Trouble-Sensor Watch Fail-#  |
| **392** | Drift Comp. Error | Trouble-Drift Comp. Error-# (Reported by Firelite panels)  |
| **393** | Maintenance Alert | Trouble-Maintenance Alert-#  |

---

### Open/Close
| Code | Description | Report String |
| :--- | :--- | :--- |
| **400** | Open/Close | Opening/Closing ($E=$ Open, $R=$ Close)  |
| **401** | OPEN/CLOSE BY USER | Opening-User # / Closing-User #  |
| **402** | Group O/C | Closing-Group User #  |
| **403** | AUTOMATIC OPEN/CLOSE | Opening-Automatic / Closing-Automatic (power up Armed)  |
| **404** | Late to O/C | Opening-Late / Closing-Late  |
| **405** | Deferred O/C | Event & Restore Not Applicable  |
| **406** | CANCEL (BY USER) | Opening-Cancel  |
| **407** | REMOTE ARM/DISARM | Opening-Remote / Closing-Remote  |
| **408** | QUICK ARM | Event Not Applicable for opening / Closing-Quick Arm  |
| **409** | KEYSWITCH OPEN/CLOSE | Opening-Keyswitch / Closing-Keyswitch  |
| **435** | Second Person Access | ACCESS- User #  |
| **436** | Irregular Access | ACCESS-Irregular Access - User #  |
| **441** | Armed Stay | Opening-Armed Stay / Closing-Armed Stay  |
| **442** | Keyswitch Armed Stay | Opening-Keysw. Arm Stay  |
| **450** | Exception O/C | Opening-Exception / Closing-Exception  |
| **451** | Early O/C | Opening-Early / Closing-Early-User #  |
| **452** | Late O/C | Opening-Late / Closing-Late-User #  |
| **453** | Failed to Open | Trouble-Fail to open (Restore not applicable)  |
| **454** | Failed to Close | Trouble-Fail to Close (Restore not applicable)  |
| **455** | Auto-Arm Failed | Trouble-Auto Arm Failed (Restore not applicable)  |
| **456** | Partial Arm | Closing-Partial arm-User #  |
| **457** | Exit Error (User) | Closing-Exit Error-User #  |
| **458** | User on Premises | Opening-User on Prem. - User #  |
| **459** | Recent Close | Trouble-Recent Close - User # (Restore not applicable)  |
| **461** | Wrong Code Entry | Access - Wrong Code entry (Restore not applicable)  |
| **462** | Legal Code Entry | Acces-Legal Code entry - user # (Restore not applicable)  |
| **463** | Re-arm after Alarm | Status-Re Arm After Alarm-User # (restore not applicable)  |
| **464** | Auto Arm Time Extended | Status-Auto Arm Time Ext. - User # (Restore not applicable)  |
| **465** | Panic Alarm Reset | Status-PA Reset (Restore not applicable)  |
| **466** | Service On/Off Premises | Access Service on/off Prem - User #  |

---

### Remote Access
| Code | Description | Report String |
| :--- | :--- | :--- |
| **411** | CALLBACK REQUESTED | Remote-Callback Requested (No Restore) Enabled with O/C reports  |
| **412** | Success-Download/access | Remote-Successful Access (Restore Not Applicable)  |
| **413** | Unsuccessful Access | Remote-Unsuccessful Access (Restore Not Applicable)  |
| **414** | System Shutdown | Remote-System Shutdown  |
| **415** | Dialer Shutdown | Remote-Dialer Shutdown  |
| **416** | Successful Upload | Remote-Successful Upload (Restore Not Applicable)  |

---

### Access Control
| Code | Description | Report String |
| :--- | :--- | :--- |
| **421** | Access Denied | Access-Access Denied-User # (Restore Not Applicable)  |
| **422** | Access Report by User | Access-Access Gained-User# (Restore Not Applicable)  |
| **423** | Forced Access | Panic-Forced Access-#  |
| **424** | Egress Denied | Access-Egress Denied (Restore Not Applicable)  |
| **425** | Egress Granted | Access-Egress Granted-# (Restore Not Applicable)  |
| **426** | Access Door Propped Open | Access-Door Propped Open-#  |
| **427** | Access Point DSM Trouble | Access-ACS Point DSM Trbl.-#  |
| **428** | Access Point RTE Trouble | Access-ACS Point RTE Trbl.-#  |
| **429** | Access Program Mode Entry | Access-ACS Prog. Entry-User # (Restore Not Applicable)  |
| **430** | Access Program Mode Exit | Access-ACS Prog. Exit-User # (Restore Not Applicable)  |
| **431** | Access Threat Level Change | Access-ACS Threat Level Chg.  |
| **432** | Access Relay/Trigger Fail | Access-ACS Relay/Trig. Fail-#  |
| **433** | Access RTE Shunt | Access-ACS RTE Shunt-#  |
| **434** | Access DSM Shunt | Access-ACS DSM Shunt-#  |

---

### System Disables

#### Access Reader Disables
| Code | Description | Report String |
| :--- | :--- | :--- |
| **501** | Access Reader Disable | Disable-Access Rdr. Disable-#  |

#### Sounder/Relay Disables
| Code | Description | Report String |
| :--- | :--- | :--- |
| **520** | Sounder/Relay Disable | Disable-Sounder/Relay-#  |
| **521** | Bell 1 Disable | Disable-Bell/Siren # 1  |
| **522** | Bell 2 Disable | Disable-Bell/Siren # 2  |
| **523** | Alarm Relay Disable | Disable-Alarm Relay  |
| **524** | Trouble Relay Disable | Disable-Trouble Relay  |
| **525** | Reversing Relay Disable | Disable-Reversing Relay  |
| **526** | Notification Appliance Ckt # 3 | Disable-Notification Appl. Ckt#3  |
| **527** | Notification Appliance Ckt # 4 | Disable-Notification Appl. Ckt#4  |

#### System Peripheral Disables
| Code | Description | Report String |
| :--- | :--- | :--- |
| **531** | Module Added | Super.-Module Added (Restore Not Applicable)  |
| **532** | Module Removed | Super.-Module Removed (Restore Not Applicable)  |

#### Communication Disables
| Code | Description | Report String |
| :--- | :--- | :--- |
| **551** | Dialer Disabled | Disable-Dialer Disable  |
| **552** | Radio Xmitter Disabled | Disable-Radio Disable  |
| **553** | Remote Upload/Download | Disable-Rem. Up/download Disable  |

---

### Bypasses
| Code | Description | Report String |
| :--- | :--- | :--- |
| **570** | ZONE/SENSOR BYPASS | Bypass-Zone Bypass-#  |
| **571** | Fire Bypass | Bypass-Fire Bypass-#  |
| **572** | 24 Hour Zone Bypass | Bypass-24 Hour Bypass-#  |
| **573** | Burg. Bypass | Bypass-Burg. Bypass-#  |
| **574** | Group Bypass | Bypass-Group Bypass-User #  |
| **575** | SWINGER BYPASS | Bypass-Swinger Bypass-#  |
| **576** | Access Zone Shunt | Access-ACS Zone Shunt-#  |
| **577** | Access Point Bypass | Access-ACS Point Bypass-#  |
| **578** | Zone Bypass | Bypass - Vault Bypass - #  |
| **579** | Zone Bypass | Bypass - Vent Zone Bypass - #  |

---

### Test / Misc
| Code | Description | Report String |
| :--- | :--- | :--- |
| **601** | MANUAL TEST | Test-Manually Triggered (Restore Not Applicable)  |
| **602** | PERIODIC TEST | Test-Periodic (Restore Not Applicable)  |
| **603** | Periodic RF Xmission | Test-Periodic Radio (Restore Not Applicable)  |
| **604** | FIRE TEST | Test-Fire Walk Test-User #  |
| **605** | Status Report To Follow | Test-Fire Walk Test-User #  |
| **606** | LISTEN-IN TO FOLLOW | Listen-Listen-In Active (Restore Not Applicable)  |
| **607** | WALK-TEST MODE | Test-Walk Test Mode-User #  |
| **608** | System Trouble Present | Test-System Trouble Present (Restore Not Applicable)  |
| **609** | VIDEO XMTR ACTIVE | Listen-Video Xmitter Active (Restore Not Applicable)  |
| **611** | POINT TESTED OK | Test-Point Tested OK-# (Restore Not Applicable)  |
| **612** | POINT NOT TESTED | Test-Point Not Tested-# (Restore Not Applicable)  |
| **613** | Intrusion Zone Walk Tested | Test-IntrnZone Walk Test-# (Restore Not Applicable)  |
| **614** | Fire Zone Walk Tested | Test-Fire Zone Walk Test-# (Restore Not Applicable)  |
| **615** | Panic Zone Walk Tested | Test-PA Zone Walk Test (Restore Not Applicable)  |
| **616** | Service Request | Trouble-Service Request  |

---

### Event Log
| Code | Description | Report String |
| :--- | :--- | :--- |
| **621** | EVENT LOG RESET | Trouble-Event Log Reset (Restore Not Applicable)  |
| **622** | EVENT LOG 50% FULL | Trouble-Event Log 50% Full (Restore Not Applicable)  |
| **623** | EVENT LOG 90% FULL | Trouble-Event Log 90% Full (Restore Not Applicable)  |
| **624** | EVENT LOG OVERFLOW | Trouble-Event Log Overflow (Restore Not Applicable)  |
| **625** | TIME/DATE RESET | Trouble-Time/Date Reset-User # (Restore Not Applicable)  |
| **626** | TIME/DATE INACCURATE | Trouble-Time/Date Invalid (Clock not stamping to log correctly) |  |
| **627** | PROGRAM MODE ENTRY | Trouble-Program Mode Entry (Restore Not Applicable)  |
| **628** | PROGRAM MODE EXIT | Trouble-Program Mode Exit (Restore Not Applicable)  |

---

### Scheduling
| Code | Description | Report String |
| :--- | :--- | :--- |
| **630** | Schedule Change | Trouble-Schedule Changed (Restore Not Applicable)  |
| **631** | Exception Sched. Change | Trouble-Esc. Sched. Changed (Restore Not Applicable)  |
| **632** | Access Schedule Change | Trouble-Access Sched. Changed (Restore Not Applicable)  |

---

### Personnel Monitoring
| Code | Description | Report String |
| :--- | :--- | :--- |
| **641** | Senior Watch Trouble | Trouble-Senior Watch Trouble ("This code is also refered to as 'up and about'. It means that a person has not moved about their home for a preset period of time".)  |
| **642** | Latch-key Supervision | Status-Latch-key Super-User # (Restore Not Applicable)  |

---

### Special Codes & Miscellaneous
| Code | Description | Report String/Notes |
| :--- | :--- | :--- |
| **651** | ADT Dealer ID | Code sent to Identify the control panel as an ADT Authorized Dealer.  |
| **654** | System Inactivity | Trouble - System Inactivity  |
| **750-789** | Protection One Use | Assigned any unique non-standard Event code tracked by Pro 1.  |
| **900** | Download Abort | Remote - Download Abort (Restore not applicable)  |
| **901** | Download Start/End | Remote - Download Start - # / Remote - Download End - #  |
| **902** | Download Interrupted | Remote - Download Interrupt - #  |
| **910** | Auto-Close with Bypass | Closing - Auto Close - Bypass - #  |
| **911** | Bypass Closing | Closing - Bypass Closing - #  |
| **912** | Fire Alarm Silenced | Event  |
| **913** | Supervisory Point test Start/End | Event - User-#  |
| **914** | Hold-up test Start/End | Event - User-#  |
| **915** | Burg. Test Print Start/End | Event  |
| **916** | Supervisory Test Print Start/End | Event  |
| **917** | Burg. Diagnostics Start/End | Event  |
| **918** | Fire Diagnostics Start/End | Event  |
| **919** | Untyped diagnostics | Event  |
| **920** | Trouble Closing | Trouble Closing (closed with burg. during exit)  |
| **921** | Access Denied Code Unknown | Event  |
| **922** | Supervisory Point Alarm | Alarm - Zone #  |
| **923** | Supervisory Point Bypass | Event - Zone #  |
| **924** | Supervisory Point Trouble | Trouble Zone #  |
| **925** | Hold-up Point Bypass | Event - Zone #  |
| **926** | AC Failure for 4 hours | Event  |
| **927** | Output Trouble | Trouble  |
| **928** | User code for event | Event  |
| **929** | Log-off | Event  |
| **954** | CS Connection Failure | Event  |
| **961** | Rcvr Database Connection Fail/Restore |  |
| **962** | License Expiration Notify | Event  |
| **999** | LOG EVENT ONLY | 1 and $1/3$ DAY NO READ LOG EVENT LOG ONLY, No report to CS.  |