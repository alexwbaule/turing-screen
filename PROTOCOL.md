# Turing Smart Screen 5" (TURZX) - Protocol Documentation

## Device Info
- USB VID: `0x1D6B` (Linux Foundation)
- USB PID: `0x0106` (Android)
- Serial (awake): `20080411`
- Serial (sleeping): `USB7INCH`
- Endpoints: EP 0x01 OUT (Bulk), EP 0x81 IN (Bulk), EP 0x84 IN (Interrupt)
- Baud: 115200, 8N1
- **CRITICAL**: DTR=1, RTS=1 must be set for data transfer to work

## Command FormatP
All commands are padded to **250 bytes** with zeros.
Response size is always **1024 bytes** (when expected).

## Complete Video Theme Sequence

Validated from `upload_full.pcapng` (full Linux usbmon capture, no truncation).

### Phase 1: INIT (always the same)
```
HELLO                              → "chs_5inch.dev1_rom1.87"
STOP_VIDEO                         → (no response)
STOP_MEDIA                         → "media_stop"
SET_BRIGHTNESS (value)             → (no response)
CMD_0x7D (pre-upload setup)        → (no response)
```

`CMD_0x7D` payload: `7d ef 69 00 00 00 05 00 00 00 aa 00`
(byte 10 = brightness value, same as SET_BRIGHTNESS)

### Phase 2: CHECK & UPLOAD (if video not on device)
```
GET_STORAGE_STATUS                 → "total-used-free-0-0-0"
GET_FILE_INFO "/root/video/X.mp4"  → file_size or "0"
```

If file doesn't exist (or needs update):
```
GET_STORAGE_STATUS                 → verify space
STOP_VIDEO                         → (no response)
STOP_MEDIA                         → "media_stop"
LIST_DIR "/root/video/"            → "result:dir:file:name1/name2/"
CREATE_FILE "/root/video/X.mp4" size=N → "create_success"
[raw file data, single write]      → (wait 5-15 seconds)
                                   ← "file_rev_doneimg_show_"
GET_FILE_INFO "/root/video/X.mp4"  → file_size (verify)
```

#### Upload Details
- Data is sent as **one continuous write** (no chunking, no framing)
- DTR=1 and RTS=1 **must** be set before writing
- After sending, poll with 1-second timeout reads until `file_rev_done` arrives
- Device takes 5-15 seconds to write to internal flash
- The `file_size` in CREATE_FILE must match the actual bytes sent

### Phase 3: PLAY VIDEO
```
RESTART_DEVICE (0x82)              → (no response)
(wait ~2 seconds)
HELLO                              → "chs_5inch.dev1_rom1.87"
GET_FILE_INFO "/root/video/X.mp4"  → file_size
PLAY_VIDEO loop=1 "/root/video/X"  → "play_video_success"
```

### Phase 4: OVERLAY (background image on top of video)
```
PRE_UPDATE_BITMAP (0x86)           → (no response)
SEPARATOR (250 bytes of 0x2c)      → (no response)
SET_BRIGHTNESS (value)             → (no response)
SEND_OVERLAY (0xca header + BGRA bitmap data)
                                   → "seq_png_init_sucess"
QUERY_STATUS (0xcf)                → "needReSend:0|renderCnt:N"
```

The overlay is a full-screen 800×480 BGRA image (1,536,000 bytes) that the device
composites on top of the playing video. Transparent areas show the video beneath.

Header: `ca ef 69 00 17 70 00 00` (0x1770 = 6000 = 1536000/256)

### Phase 5: SENSOR UPDATES (continuous loop)
```
UPDATE_BITMAP (0xcc + partial region data)
QUERY_STATUS (0xcf)                → "needReSend:0|renderCnt:N"
```

Partial updates render sensor data (CPU temp, GPU usage, etc.) as small
rectangular regions on top of the overlay. Same protocol as static themes.

---

## Static Theme Sequence (no video)

```
HELLO                              → "chs_5inch.dev1_rom1.87"
STOP_VIDEO                         → (no response)
STOP_MEDIA                         → "media_stop"
SET_BRIGHTNESS (value)             → (no response)
PRE_UPDATE_BITMAP (0x86)           → (no response)
SEPARATOR (250 bytes of 0x2c)      → (no response)
SEND_BITMAP (0xc8 + BGRA data)    → "full_png_sucess"
QUERY_STATUS (0xcf)                → "needReSend:0|renderCnt:N"
UPDATE_BITMAP (sensor loop)...
```

The static bitmap is a full-screen 800×480 BGRA image (1,536,000 bytes).
Header: `c8 ef 69 00 17 70` (0x1770 = 6000 = 1536000/256)

After the initial full image, sensor updates are sent as partial UPDATE_BITMAP
commands targeting specific rectangular regions.

---

## Video Conversion (FFmpeg)

The device accepts H.264 video in MP4 container with these parameters:

| Parameter | Value |
|-----------|-------|
| Codec | H.264 (libx264) |
| Profile | Main or High |
| Level | 3.0 |
| Resolution | 800×480 |
| Frame rate | 24 fps |
| Pixel format | yuv420p |
| Audio | None (stripped) |
| Container | MP4 |
| Max size | ~10 MB (device storage limit) |

### FFmpeg command:
```bash
ffmpeg -i input.mp4 \
  -c:v libx264 -profile:v main -level 3.0 \
  -pix_fmt yuv420p -s 800x480 -r 24 \
  -an -movflags +faststart \
  output.mp4
```

Notes:
- `-an` removes audio (device has no speaker)
- `-movflags +faststart` puts moov atom at beginning
- `-r 24` is mandatory — all device videos use 24 fps
- Bitrate varies by content (200 kbps to 2.2 Mbps), no fixed target needed
- The device's internal decoder handles the bitrate adaptation

### Source format (from Windows app):
The Windows app stores source videos as raw H.264 Annex B streams (`.mp4.h264`)
at 48 fps, Constrained Baseline profile. It converts them to the MP4 format above
before uploading.

---

## Command Reference

| Cmd  | Name | Response | Notes |
|------|------|----------|-------|
| 0x01 | HELLO | 23 bytes: device string | Init/ping |
| 0x64 | GET_STORAGE_STATUS | "total-used-free-0-0-0" (KB) | |
| 0x65 | LIST_DIR | "result:dir:file:name1/name2/" | Flush buffer after |
| 0x66 | DELETE_FILE | "" | |
| 0x6e | GET_FILE_INFO | file size (ASCII) or "0" | |
| 0x6f | CREATE_FILE | "create_success" | Followed by raw data |
| 0x78 | PLAY_VIDEO | "play_video_success" | |
| 0x79 | STOP_VIDEO | (none) | |
| 0x7b | SET_BRIGHTNESS | (none) | byte[10] = 0-255 |
| 0x7d | CMD_0x7D | (none) | Pre-upload setup |
| 0x82 | RESTART_DEVICE | (none) | Soft restart, wait 2s |
| 0x86 | PRE_UPDATE_BITMAP | (none) | Before full image |
| 0x96 | STOP_MEDIA | "media_stop" | |
| 0xc8 | SEND_BITMAP | "full_png_sucess" | Static background |
| 0xca | SEND_OVERLAY | "seq_png_init_sucess" | Video overlay |
| 0xcc | UPDATE_BITMAP | "needReSend:0\|renderCnt:N" | Partial update |
| 0xcf | QUERY_STATUS | "needReSend:0\|renderCnt:N" | After bitmap batch |

---

## Key Findings

1. **DTR=1, RTS=1** must be set on the serial port for bulk data transfer to work.
   Without this, `write()` blocks indefinitely on Linux.

2. **Upload uses raw writes** — no 250-byte framing for file data. Just one big `write()`.
   The OS/USB driver handles segmentation into USB bulk transfers.

3. **`file_rev_done` takes 5-15 seconds** — the device writes to internal flash.
   Must poll with short timeout reads, not one long blocking read.

4. **CMD_0x7D** is sent before every upload session. Purpose unknown but required.

5. **Video overlay (0xca)** is sent twice in some captures (Windows app quirk).
   One send is sufficient.

6. **The device is a USB CDC ACM device** but the Windows app uses LibUsbDotNet
   (direct USB access). On Linux, pyserial via `/dev/ttyACM0` works fine with
   proper DTR/RTS configuration.
