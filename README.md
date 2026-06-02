# Turing Smart Screen 5" — Linux Driver

Open-source driver for the Turing Smart Screen 5" (TURZX) USB display, written in Go.

Displays real-time system monitoring data (CPU, GPU, memory, disk, network) on the device, with support for both static image backgrounds and video playback with overlay.

## Features

- **Static image mode** — Full-screen background with real-time sensor overlays
- **Video mode** — MP4 video playback with transparent sensor overlay
- **Auto-upload** — Automatically uploads video to device if not present
- **Hardware detection** — Auto-detects CPU, GPU (AMD/NVIDIA), disk, network interfaces
- **Configurable themes** — YAML-based theme system with fonts, colors, layouts
- **Resilient** — Reconnection, retry, and device wake-up handling

## Requirements

- Linux (tested on Arch-based)
- Go 1.21+
- Device: Turing Smart Screen 5" (USB VID `1D6B`, PID `0106`)
- For video conversion: `ffmpeg`

## Quick Start

```bash
# Build
make build

# Run (device must be connected)
./bin/turing-screen

# Or with sudo (if permission issues)
sudo ./bin/turing-screen
```

## Configuration

Edit `conf/config.yaml`:

```yaml
device:
  port: AUTO          # or /dev/ttyACM0
  theme: NZXT_B       # theme folder name in res/themes/
  log: debug
  display:
    width: 800
    height: 480
    brightness: 20    # 0-100
  sensors:
    network:
      eth: "enp3s0"
    gpu:
      provider: "auto"  # auto | amd | nvidia | none
```

## Themes

Themes are in `res/themes/<name>/`. Each theme has:
- `theme.yaml` — Layout configuration
- `background.png` — Background image (800×480)
- Fonts referenced from `res/fonts/`

### Static Theme

```yaml
display:
  SIZE: 5"
  ORIENTATION: landscape

static_images:
  BACKGROUND:
    PATH: background.png
    X: 0
    Y: 0

STATS:
  CPU:
    PERCENTAGE:
      INTERVAL: 1
      GRAPH:
        SHOW: True
        X: 50
        Y: 134
        WIDTH: 184
        HEIGHT: 19
        BACKGROUND_IMAGE: background.png
```

### Video Theme

Add a `video_play` section to enable video mode:

```yaml
video_play:
  BACKGROUND_VIDEO:
    PATH: video.mp4
    X: 0
    Y: 0
    WIDTH: 800
    HEIGHT: 480
```

The video file must be H.264 MP4, 800×480, 24fps. Convert with:

```bash
ffmpeg -i input.mp4 -c:v libx264 -profile:v main -level 3.0 \
       -pix_fmt yuv420p -s 800x480 -r 24 -an \
       -movflags +faststart output.mp4
```

## Architecture

```
cmd/turing-screen/main.go              — Entry point
internal/
  application/
    config/                             — Configuration loading
    theme/                              — Theme parsing
    hwinfo/                             — Hardware detection
    logger/                             — Structured logging
  domain/
    command/                            — Protocol commands (value objects)
    entity/                             — Domain entities (theme, device)
    service/
      initializer/                      — Synchronous device init (static.go, video.go)
      renderer/                         — Image rendering (text, graphs, radials)
      sensors/                          — Sensor data collection goroutines
      sender/                           — Async command worker
      video/                            — Video overlay buffer and encoder
  resource/
    serial/                             — Serial port driver (USB CDC ACM)
    gpu/                                — GPU hardware access
    process/device/                     — Image encoding (BGR/BGRA, positioning)
    usb/                                — USB device detection
```

### Flow

1. **Initialization** (synchronous) — Handshake, stop media, set brightness, send background or start video
2. **Sensor loop** (async) — Goroutines collect data, render images, send UPDATE_BITMAP commands
3. **Video overlay refresh** (async, 1Hz) — Computes diff, sends only changed pixels

## Protocol

See [PROTOCOL.md](PROTOCOL.md) for the complete device communication protocol.

## Scripts

Python utility scripts for testing and debugging:

```bash
# List files on device storage
python3 scripts/list_storage.py

# Upload a video
python3 scripts/upload_video.py video.mp4

# Play a video already on device
python3 scripts/play_video.py /root/video/video.mp4

# Upload and play in one step
python3 scripts/upload_and_play.py video.mp4

# Device diagnostic
python3 scripts/test_video.py
```

## License

See LICENSE file.
