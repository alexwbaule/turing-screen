# Turing Smart Screen — Wire Protocol Reference

This document describes the two physical protocols used to communicate with
Turing Smart Screen devices.  Both are supported in the same daemon binary;
the correct driver is selected automatically at startup.

---

## Device identification

| Family | Detection method |
|---|---|
| **Rev-C** | CDC-ACM serial port whose USB serial number is `20080411` |
| **TURZX** | USB bulk device with VID `0x1CBE` and a known PID (see table below) |

Detection order: TURZX is probed first (libusb enumeration); if no TURZX
device is found, the daemon falls back to looking for a Rev-C serial port.

---

## 1 — Rev-C (USB CDC-ACM serial)

### 1.1 Physical layer

| Parameter | Value |
|---|---|
| Interface | USB CDC-ACM (`/dev/ttyACM*`) |
| USB serial number | `20080411` |
| Baud rate | 115 200 |
| Data bits | 8 |
| Parity | None |
| Stop bits | 1 |
| Flow control | None |
| Read timeout | 5 s (100 ms during buffer flush) |

The device is identified by its USB serial number string, not by VID/PID.
Any VID/PID may be present depending on the enclosure manufacturer.

**Sleep / wake quirk** — if the serial number is `USB7INCH` the device is
asleep.  The driver sets DTR+RTS to wake it, then waits 20 seconds before
re-enumerating.

### 1.2 Packet framing

All commands are sent as fixed-length **250-byte** packets:

```
[ payload_bytes … | padding_byte × (250 − len(payload)) ]
```

- Most commands use `0x00` as the padding byte.
- `START_DISPLAY_BITMAP` uses `0x2C` as both the command byte and the padding.
- The device responds with ASCII strings terminated by `\0` bytes, read until
  the expected response size or a recognisable response string is matched.

### 1.3 Command reference

Commands are prefixed with a 6-byte header `[cmd, 0xEF, 0x69, 0x00, 0x00, 0x00]`
unless noted otherwise.

#### Device control

| Command | Byte | Notes |
|---|---|---|
| `HELLO` | `0x01` | Identifies the device. Response: ASCII string matching `chs_5inch.dev1_romX.YY` (23 bytes). |
| `TURN_OFF` | `0x83` | Turns off the display. No response. |
| `RESTART` | `0x84` | Firmware-level restart. No response. |
| `RESTART_DEVICE` (soft) | `0x82` | Soft restart used in the video upload sequence. No response. |

#### Brightness

| Command | Byte | Payload layout |
|---|---|---|
| `SET_BRIGHTNESS` | `0x7B` | `[0x7B, 0xEF, 0x69, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, brightness]` where `brightness = (value/100.0) × 255` (0–255). No response. |

#### Frame rendering (static mode)

Frame rendering uses a three-packet sequence followed by raw BGRA pixel data:

```
1. PRE_UPDATE_BITMAP  [0x86, 0xEF, 0x69, 0x00, 0x00, 0x00, 0x01]  pad=0x00
2. START_DISPLAY_BITMAP  [0x2C]                                      pad=0x2C
3. DISPLAY_BITMAP  [0xC8, 0xEF, 0x69, 0x00, 0x17, 0x70]            pad=0x00
4. <BGRA payload in 249-byte chunks, each padded to 250 bytes>
```

After the last chunk the device responds with `full_png_sucess` (1024 bytes).

#### Frame rendering (video overlay mode)

Identical to static mode except step 3 uses `DISPLAY_BITMAP_ON_VIDEO`:

```
3. DISPLAY_BITMAP_ON_VIDEO  [0xCA, 0xEF, 0x69, 0x00, 0x17, 0x70]  pad=0x00
```

No response expected (the subsequent `INIT_VIDEO_OVERLAY` command carries the
acknowledgement).

#### Video overlay initialisation

The `INIT_VIDEO_OVERLAY` sequence activates the overlay after the BGRA payload
is sent in video mode:

```
1. PRE_UPDATE_BITMAP    [0x86, 0xEF, 0x69, 0x00, 0x00, 0x00, 0x01]
2. START_DISPLAY_BITMAP [0x2C]
3. DISPLAY_BITMAP_ON_VIDEO [0xCA, 0xEF, 0x69, 0x00, 0x17, 0x70]
4. <BGRA overlay chunks>
5. INIT_VIDEO_OVERLAY   [0xD0, 0xEF, 0x69, 0x00, 0x00, 0x00, 0x02]
6. SEND_PAYLOAD         [0xEF, 0x69]
```

#### Media control

| Command | Byte | Payload / notes |
|---|---|---|
| `START_VIDEO` | `0x78` | `[0x78, 0xEF, 0x69, path_len(4 BE), loop_flag(1), 0x00, 0x00, path]`. `loop_flag=0x01` loops, `0x00` plays once. Response: `play_video_success`. |
| `STOP_VIDEO` | `0x79` | No response. |
| `STOP_MEDIA` | `0x96` | Response: `media_stop` (1024 bytes). |
| `POST_UPDATE_BITMAP` | `0x86` | Alias; same byte as PRE_UPDATE_BITMAP. No response. |
| `QUERY_STATUS` | `0xCF` | Returns current device status string (1024 bytes). Also used as a mid-sequence heartbeat after payload transmission. |

#### Storage

Path-based commands share this payload layout after the 6-byte header:
`[path_len: 1 byte][0x00, 0x00, 0x00][path bytes]`

| Command | Byte | Response |
|---|---|---|
| `GET_STORAGE_STATUS` | `0x64` | ASCII `"total-used-free-0-0-0"` (KB values, 1024 bytes). |
| `LIST_DIR` | `0x65` | ASCII `"result:dir:file:name1/name2/…"` (1024 bytes). |
| `DELETE_FILE` | `0x66` | ASCII `"delete_success"` (1024 bytes). |
| `GET_FILE_INFO` | `0x6E` | ASCII decimal file size in bytes, e.g. `"480610"` (1024 bytes). |
| `CREATE_FILE` | `0x6F` | Payload: `[path_len][0×3][path][file_size: 4 bytes LE]`. Response: `"create_success"` (1024 bytes). |
| `UPLOAD_FILE` | — | After `CREATE_FILE`: send raw file bytes in 249-byte chunks (padded to 250). Response: `"file_rev_done"` (or `"file_rev_doneimg_show_"` for images). |
| `SET_PRE_UPLOAD` | `0x7D` | Prepares device for file upload. Must precede `CREATE_FILE`. Payload: brightness byte at offset 10. No response. |
| `SET_START_MODE_DEFAULT` | `0x7B` | Sets startup mode to default. Payload tail: `0x8A, 0x00`. No response. |
| `SET_START_MODE_VIDEO` | `0x7B` | Sets startup mode to video. Payload tail: `0x8A, 0x02`. No response. |

#### Options (startup behaviour)

| Command | Byte | Payload |
|---|---|---|
| `OPTIONS` | `0x7D` | `[0x7D, 0xEF, 0x69, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x2D, start_mode, 0x00, flip, sleep]`. No response. |

`start_mode`: `0x00`=default, `0x01`=image, `0x02`=video.
`flip`: `0x00`=no flip, `0x01`=flip 180°.
`sleep`: `0x00`=disabled, `0x01`–`0x0A`=interval (implementation-defined minutes).

### 1.4 Storage paths (Rev-C)

```
/root/video/        — video files
/root/image/        — image files
/root/font/         — font files
/root/              — root (enumerable)
```

### 1.5 Frame format (Rev-C)

Frames are encoded as raw **BGRA** (blue-green-red-alpha) pixel data at the
device's native resolution (commonly 800×480).  The daemon handles rotation
internally before transmission.

---

## 2 — TURZX (USB bulk / libusb)

### 2.1 Physical layer

| Parameter | Value |
|---|---|
| Interface | USB bulk (libusb, interface 0, alternate 0) |
| Vendor ID | `0x1CBE` |
| Auto-detach kernel driver | Yes (`SetAutoDetach(true)`) |
| Command packet size | 512 bytes |
| Upload chunk size | 1 MB |

#### Supported Product IDs

| PID | Native resolution (portrait W×H) |
|---|---|
| `0x0028` | 480 × 480 |
| `0x0046` | 320 × 960 |
| `0x0050` | 720 × 1280 (Turing 5.2") |
| `0x0080` | 800 × 1280 |
| `0x0088` | 480 × 1920 |
| `0x0092` | 462 × 1920 |
| `0x0123` | 720 × 1920 |

All frames are transmitted in **portrait** orientation and the device firmware
rotates them per the configured `ORIENTATION` setting.

### 2.2 Packet encryption

Every TURZX command is encrypted before transmission.

**Build the plaintext packet (500 bytes):**

```
[0]     = cmdID
[1]     = 0x00
[2]     = 0x1A
[3]     = 0x6D
[4:8]   = milliseconds since midnight (little-endian uint32)
[8:12]  = command-specific parameters (see below)
[12:]   = 0x00 (remainder)
```

**Encrypt to 512 bytes (DES-CBC):**

1. Pad the 500-byte packet to 504 bytes (next multiple of 8).
2. DES-CBC encrypt with key = IV = `"slv3tuzx"` (8 bytes, ASCII).
3. Copy the 504 encrypted bytes into a 512-byte buffer.
4. Set `buf[510] = 0xA1`, `buf[511] = 0x1A` (protocol markers).

**Response decryption** uses the same key/IV (DES-CBC decrypt on the raw
512-byte response).

### 2.3 Command reference

All commands send a 512-byte encrypted packet and read a 512-byte encrypted
response (unless noted).  Commands that also carry a payload append raw bytes
directly after the encrypted header.

#### Initialisation sequence

```
1. SYNC         (cmd 10)  — handshake; required before any other command
2. BRIGHTNESS   (cmd 14)  — set initial brightness
3. FRAME_RATE   (cmd 15)  — set frame rate (default 25 fps)
```

#### Device control

| Command | ID | Parameter bytes | Notes |
|---|---|---|---|
| `SYNC` | 10 | — | Handshake. Must be first. |
| `RESTART_USB` | 11 | — | Firmware restart. May cause USB re-enumeration. Use with care. |
| `BRIGHTNESS` | 14 | `pkt[8] = byte(value × 102 / 100)` (0–102 maps to 0–100%) | No response body used. |
| `FRAME_RATE` | 15 | `pkt[8] = fps` (e.g. 25) | No response body used. |

#### Frame transmission

| Command | ID | Payload |
|---|---|---|
| `UPLOAD_PNG` | 102 | `pkt[8:12]` = PNG size big-endian. Appended after the 512-byte header: raw PNG bytes. |

The PNG must be:
- Color type 6 (RGBA, 8-bit depth).
- Filter method None (filter byte `0x00` per row) for performance.
- Alpha channel forced to 254 (not 255) to prevent Go's encoder from
  downgrading to RGB type 2, which the device firmware rejects.

#### H.264 video streaming

The video streaming protocol mirrors the Python `send_video` reference:

```
Setup sequence (sent once):
  111 — VIDEO_MODE
  112 — VIDEO_MODE_ACK
   13 — VIDEO_INIT
   14 — BRIGHTNESS (value=32, dim during streaming)
   41 — VIDEO_OVERLAY
  102 — UPLOAD_PNG (full-black clear frame)
   15 — FRAME_RATE (25 fps)
   17 — GET_H264_CHUNK_SIZE  →  pkt[8:12] big-endian = negotiated chunk size
                                 (default: 202752 bytes if response is absent/invalid)

Streaming loop (until cancelled):
  121 — PLAY_H264_CHUNK  pkt[8:12]=chunk_len BE, pkt[12]=isLast(0/1), then raw H264 data
  122 — GET_STREAM_STATUS  →  pkt[8] = queue depth (pause if > 3)

Teardown:
  123 — STOP_STREAM
```

PNG overlay frames (from `SendFrame`) can be interleaved with H.264 chunks
because both paths serialise on the same mutex.

#### Storage operations

Path-based commands embed the path string starting at `pkt[16]` with the
path length at `pkt[8:12]` (big-endian, bytes 12–15 are zero).

| Command | ID | Notes |
|---|---|---|
| `GET_STORAGE_INFO` | 100 | Response: `[8:12]`=TotalKB LE, `[12:16]`=UsedKB LE, `[16:20]`=FreeKB LE. All-zero = SD card not detected. |
| `LIST_DIR` | 101 | Path at `pkt[16:]`. Response: ASCII `"result:dir:file:name1/name2/…"` (up to 1024 bytes). |
| `OPEN_FILE` | 38 | Path at `pkt[16:]`. Opens/creates a remote file for writing. |
| `WRITE_FILE_CHUNK` | 39 | `pkt[8:12]`=capacity BE (1 MB), `pkt[12:16]`=chunk_len BE, `pkt[16]`=isLast (0/1), then raw data. |
| `DELETE_FILE` | 40 | Path at `pkt[16:]`. |
| `PLAY_VIDEO_USB` | 98 | Path at `pkt[16:]`. |
| `PLAY_VIDEO_2` | 110 | Path at `pkt[16:]`. Alternate play command. |
| `PLAY_IMAGE` | 113 | Path at `pkt[16:]`. |
| `SAVE_SETTINGS` | 125 | `pkt[8]`=brightness, `pkt[9]`=startup, `pkt[10]`=reserved, `pkt[11]`=rotation, `pkt[12]`=sleep, `pkt[13]`=offline. Persists to device flash. |
| `STOP_STREAM` | 123 | Halts video playback from storage. |

**Upload sequence:** `OPEN_FILE` (38) → `WRITE_FILE_CHUNK` (39) × N (last chunk has `pkt[16]=1`).

### 2.4 Storage paths (TURZX)

```
/tmp/sdcard/mmcblk0p1/video/   — video files
/tmp/sdcard/mmcblk0p1/img/     — image files
```

The SD card must be inserted **before** power-on; hot-swap is not supported.

### 2.5 Frame orientation

All frames are encoded in portrait orientation and transmitted to the device.
The driver rotates the compositor's output image before encoding:

| Theme `ORIENTATION` | Rotation applied before transmission |
|---|---|
| `REVERSE_PORTRAIT` (default) | None (native portrait) |
| `LANDSCAPE` | Rotate 270° CCW (= 90° CW) |
| `REVERSE_LANDSCAPE` | Rotate 90° CCW (= 270° CW) |
| `PORTRAIT` | Rotate 180° |

Rotation is performed using pre-allocated scratch buffers to avoid per-frame
heap allocations after the first frame.

### 2.6 Response flushing

After every command, the driver reads and discards up to 5 additional
512-byte packets from the IN endpoint (100 ms timeout each) to clear any
residual buffered responses.

---

## 3 — WebSocket API (daemon ↔ editor / client)

The daemon exposes a WebSocket server (default port **9120**) at `/ws`.

### 3.1 Message format

All messages are JSON objects:

```json
{
  "id":      "client-generated request ID (string)",
  "action":  "action.name",
  "status":  "ok | error",
  "payload": { … },
  "error":   "error description (only on error)"
}
```

Push events (server-initiated) omit `"id"`.

### 3.2 Request/response actions

| Action | Direction | Payload (request) | Payload (response) |
|---|---|---|---|
| `status.get` | C→S | — | `StatusResponse` |
| `mode.editor` | C→S | — | `{"mode":"editor"}` |
| `mode.normal` | C→S | — | `{"mode":"normal"}` |
| `device.brightness` | C→S | `{"value": 0–100}` | `{"status":"ok"}` |
| `device.settings` | C→S | `DeviceSettings` | `{"status":"ok"}` |
| `device.restart` | C→S | — | `{"status":"ok"}` |
| `device.reboot` | C→S | — | `{"status":"ok"}` |
| `device.reset` | C→S | — | `{"status":"ok"}` |
| `device.hardreset` | C→S | — | `{"status":"ok"}` |
| `device.turnoff` | C→S | — | `{"status":"ok"}` |
| `theme.current` | C→S | — | `{"theme":"name"}` |
| `theme.list` | C→S | — | `{"themes":["a","b",…]}` |
| `theme.apply` | C→S | `{"name":"theme_name"}` | `{"status":"ok","theme":"name"}` |
| `theme.preview` | C→S | `{"image":"<base64 PNG>"}` | `{"status":"ok"}` |
| `theme.video.start` | C→S | `{"path":"/device/path/file.mp4"}` | `{"status":"ok"}` |
| `theme.video.stop` | C→S | — | `{"status":"ok"}` |
| `storage.info` | C→S | — | `{"total":N,"used":N,"free":N}` (bytes) |
| `storage.files` | C→S | `{"path":"/root/video/"}` | `{"files":["a.mp4","b.mp4"]}` |
| `storage.upload` | C→S | `{"name":"file.mp4","data":"<base64>"}` | `{"status":"ok"}` |
| `storage.delete` | C→S | `{"path":"/root/video/file.mp4"}` | `{"status":"ok"}` |
| `sensors.values` | C→S | — | Map of sensor key → current value |

### 3.3 Push events (server → all clients, 500 ms interval)

| Action | Payload |
|---|---|
| `event.status` | `StatusResponse` |
| `event.sensors` | Same as `sensors.values` response |

### 3.4 StatusResponse schema

```json
{
  "mode":        "normal | editor | starting",
  "theme":       "current theme name",
  "firmware":    "device firmware string (Rev-C only)",
  "uptime":      "human-readable uptime string",
  "api_version": "daemon API version",
  "device_type": "turzx | revc"
}
```

### 3.5 DeviceSettings schema

```json
{
  "brightness": 0,
  "startup":    0,
  "rotation":   0,
  "sleep":      0,
  "offline":    0
}
```

All fields 0–255.  Zero means "keep current / default".
`device.settings` persists to device flash (TURZX: `SAVE_SETTINGS` cmd 125;
Rev-C: `OPTIONS` cmd `0x7D`).
