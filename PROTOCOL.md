# Turing Smart Screen 5" (TURZX) — Protocol Documentation

## Device Info

| Field | Value |
|-------|-------|
| USB VID | `0x1D6B` (Linux Foundation) |
| USB PID | `0x0106` (Android) |
| Serial (awake) | `20080411` |
| Serial (sleeping) | `USB7INCH` |
| Endpoints | EP 0x01 OUT (Bulk), EP 0x81 IN (Bulk), EP 0x84 IN (Interrupt) |
| Baud | 115200, 8N1 |
| Flow control | None (IXON/IXOFF/CRTSCTS disabled) |
| DTR/RTS | Must be set to 1 for data transfer |

## Command Format

- All commands are padded to **250 bytes** (with 0x00 or 0x2c depending on command).
- Payload data is split into **249-byte chunks** padded to 250.
- Response buffer is **1024 bytes** (for commands that expect a response).
- Commands that don't expect a response have `ValidateWrite().Size = 0`.

## Initialization Sequence (Common)

Both static and video modes start with the same handshake:

```
HELLO (0x01)                       → "chs_5inch.dev1_rom1.87" (23 bytes)
OPTIONS (0x7D)                     → (no response)
STOP_VIDEO (0x79)                  → (no response)
STOP_MEDIA (0x96)                  → "media_stop" (1024 bytes)
SET_BRIGHTNESS (0x7B)              → (no response)
```

---

## Static Image Flow

After common init:

```
PRE_UPDATE_BITMAP (0x86)           → (no response, padding 0x00)
START_DISPLAY_BITMAP (0x2C)        → (no response, padding 0x2C)
DISPLAY_BITMAP (0xC8)              → payload (BGRA 800×480)
QUERY_STATUS (0xCF)                → "full_png_sucess" (1024 bytes)
```

Then sensor updates loop:

```
UPDATE_BITMAP (0xCC) + payload     → (data chunks)
QUERY_STATUS (0xCF)                → "needReSend:0|renderCnt:N" (1024 bytes)
```

### UPDATE_BITMAP Format (Static)

Header (14 bytes in first 250-byte chunk):
```
[0xCC][0xEF][0x69][0x00][0x00]  — command
[size: 2 bytes BE]               — payload size
[pad: 3 bytes 0x00]
[count: 4 bytes BE]              — frame counter
```

Payload format (from `GeneratePartialImage`):
```
For each scanline:
  [position: 3 bytes BE]  — (row * display_width + col)
  [width: 2 bytes BE]     — pixels in this line
  [BGR pixels: width * 3 bytes]
End marker:
  [0xEF][0x69]
```

---

## Video Flow

After common init:

### 1. Check/Upload Video

```
GET_FILE_INFO (0x6E) "/path"       → file size ASCII or "0"
```

If file doesn't exist (size = 0):
```
CREATE_FILE (0x6F) path + size     → "create_success" (1024 bytes)
[raw file data in 249-byte chunks] → "file_rev_done" (1024 bytes, wait 5-15s)
```

### 2. Start Video Playback

```
STOP_VIDEO (0x79)                  → (no response)
[clear display with white bitmap]
START_VIDEO (0x78) path            → (no response)
```

### 3. Initialize Video Overlay

```
PRE_UPDATE_BITMAP (0x86)           → (no response)
START_DISPLAY_BITMAP (0x2C)        → (no response, padding 0x2C)
DISPLAY_BITMAP_ON_VIDEO (0xCA)     → payload (BGRA 800×480, 249+1 chunking)
INIT_VIDEO_OVERLAY (0xD0)          → (no response)
[visible pixels: 0xEF 0x69]       → (no response)
```

### 4. Video Overlay Refresh (periodic, every 1s)

The overlay uses 4-bit pixel packing with diff computation:
- Only changed pixels are sent
- Alpha channel determines visibility (alpha=0 → video shows through)
- Position encoded as `row * stride + col`
- Boundary fix applied when payload crosses 250-byte boundaries

---

## Command Reference

| Byte | Name | Response Size | Response |
|------|------|--------------|----------|
| 0x01 | HELLO | 23 | Device string |
| 0x64 | GET_STORAGE_STATUS | 1024 | "total-used-free-0-0-0" |
| 0x65 | LIST_DIR | 1024 | "result:dir:file:name1/name2/" |
| 0x66 | DELETE_FILE | 1024 | "" |
| 0x6E | GET_FILE_INFO | 1024 | File size (ASCII) or "0" |
| 0x6F | CREATE_FILE | 1024 | "create_success" |
| 0x78 | START_VIDEO / PLAY_VIDEO | 0 | — |
| 0x79 | STOP_VIDEO | 0 | — |
| 0x7B | SET_BRIGHTNESS | 0 | — |
| 0x7D | OPTIONS | 0 | — |
| 0x82 | RESTART_DEVICE (soft) | 0 | — |
| 0x83 | TURN_OFF | 0 | — |
| 0x84 | RESTART (hard) | 0 | — |
| 0x86 | PRE_UPDATE_BITMAP | 0 | — |
| 0x96 | STOP_MEDIA | 1024 | "media_stop" |
| 0xC8 | DISPLAY_BITMAP (static) | 1024* | "full_png_sucess" |
| 0xCA | DISPLAY_BITMAP_ON_VIDEO | 0 | — |
| 0xCC | UPDATE_BITMAP | 1024* | "needReSend:0\|renderCnt:N" |
| 0xCF | QUERY_STATUS | 1024 | varies |
| 0xD0 | INIT_VIDEO_OVERLAY | 0 | — |

*Response is read after QUERY_STATUS (0xCF) is sent.

---

## Video Conversion (FFmpeg)

The device accepts H.264 video in MP4 container:

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
| Max size | ~10 MB |

```bash
ffmpeg -i input.mp4 \
  -c:v libx264 -profile:v main -level 3.0 \
  -pix_fmt yuv420p -s 800x480 -r 24 \
  -an -movflags +faststart \
  output.mp4
```

---

## Key Implementation Notes

1. **Flow control must be disabled** — `IXON`/`IXOFF` cause reads to hang. The `alexwbaule/serial` fork handles this.

2. **DTR=1, RTS=1** — Required for bulk data transfer (file upload). Without this, writes block on Linux.

3. **Upload uses raw writes** — File data is sent in 249-byte chunks padded to 250. Device responds with `file_rev_done` after 5-15 seconds (flash write time).

4. **QueryStatus (0xCF) triggers response** — For UPDATE_BITMAP and DISPLAY_BITMAP, the device only responds after receiving the QueryStatus command.

5. **Video overlay uses alpha** — Alpha=0 means transparent (video shows through). The overlay diff engine only sends changed pixels using 4-bit color packing.

6. **GET_STORAGE_STATUS can take 6+ seconds** — First call after wake-up is slow. Read timeout must be ≥10s.
