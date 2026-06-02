# Themes

## Structure

Each theme is a folder containing:
- `theme.yaml` — Layout and sensor configuration
- `background.png` — Background image (must match display resolution)
- Optional: video file for video themes

## Display Sizes

| Size | Resolution | Orientation |
|------|-----------|-------------|
| 3.5" | 320×480 | portrait |
| 5" | 800×480 | landscape |

## Modes

### Static Image

The background image is sent once, then sensor data updates specific regions on top of it.

### Video Playback

A video plays in the background, with a transparent overlay showing sensor data.
The video must be H.264 MP4, matching the display resolution at 24fps.

Convert any video:
```bash
ffmpeg -i input.mp4 -c:v libx264 -profile:v main -level 3.0 \
       -pix_fmt yuv420p -s 800x480 -r 24 -an \
       -movflags +faststart output.mp4
```

## Theme YAML Reference

### Display Section

```yaml
display:
  SIZE: 5"                    # 3.5" or 5"
  ORIENTATION: landscape      # portrait | landscape
  RGB_LED: 255, 0, 0          # Backplate LED color (optional)
```

### Video Play (optional — enables video mode)

```yaml
video_play:
  BACKGROUND_VIDEO:
    PATH: video.mp4           # Video file (relative to theme folder)
    X: 0
    Y: 0
    WIDTH: 800
    HEIGHT: 480
```

### Static Images

```yaml
static_images:
  BACKGROUND:
    PATH: background.png
    X: 0
    Y: 0
    WIDTH: 800
    HEIGHT: 480
```

### Static Texts

Rendered once on the background. Supports template placeholders:
- `{{CPU_MODEL}}` — Auto-detected CPU model
- `{{GPU_MODEL}}` — Auto-detected GPU model
- `{{DISK_MODEL}}` — Auto-detected disk model
- `{{MEM_SIZE}}` — Total memory
- `{{HOSTNAME}}` — Machine hostname

```yaml
static_texts:
  CPU_MODEL:
    TEXT: "{{CPU_MODEL}}"
    X: 50
    Y: 32
    FONT: roboto/Roboto-Bold.ttf
    FONT_SIZE: 30
    FONT_COLOR: 184, 225, 254
    BACKGROUND_IMAGE: background.png
```

### Sensors (STATS)

Each sensor has:
- `INTERVAL` — Update frequency in seconds
- `TEXT` — Numeric text display
- `GRAPH` — Horizontal progress bar
- `RADIAL` — Circular gauge

#### Common Properties

```yaml
TEXT:
  SHOW: True
  SHOW_UNIT: True               # Append unit (%, °C, MB/s)
  X: 100
  Y: 50
  FONT: roboto/Roboto-Bold.ttf
  FONT_SIZE: 20
  FONT_COLOR: 255, 255, 255
  # BACKGROUND_COLOR: 0, 0, 0  # Solid color background
  BACKGROUND_IMAGE: background.png  # Or image crop as background

GRAPH:
  SHOW: True
  X: 50
  Y: 134
  WIDTH: 184
  HEIGHT: 19
  MIN_VALUE: 0
  MAX_VALUE: 100
  BAR_COLOR: 255, 255, 255
  BAR_OUTLINE: False
  BACKGROUND_IMAGE: background.png

RADIAL:
  SHOW: True
  X: 100
  Y: 100
  RADIUS: 40
  WIDTH: 10
  MIN_VALUE: 0
  MAX_VALUE: 100
  ANGLE_START: 120
  ANGLE_END: 60
  CLOCKWISE: True
  BAR_COLOR: 0, 255, 0
  SHOW_TEXT: True
  SHOW_UNIT: True
  FONT: roboto/Roboto-Bold.ttf
  FONT_SIZE: 13
  FONT_COLOR: 200, 200, 200
  BACKGROUND_IMAGE: background.png
```

#### Available Sensors

```yaml
STATS:
  CPU:
    PERCENTAGE:     # CPU usage %
    FREQUENCY:      # CPU frequency (GHz)
    TEMPERATURE:    # CPU temperature (°C)
    LOAD:           # System load (1/5/15 min)
  GPU:
    PERCENTAGE:     # GPU usage %
    MEMORY:         # GPU VRAM usage
    TEMPERATURE:    # GPU temperature
  MEMORY:
    VIRTUAL:        # RAM (USED, FREE, PERCENT_TEXT, GRAPH, RADIAL)
    SWAP:           # Swap (same options)
  DISK:
    USED:           # Disk usage (TEXT, PERCENT_TEXT, GRAPH, RADIAL)
    TOTAL:          # Total disk size
    FREE:           # Free disk space
    TEMPERATURE:    # Disk temperature
  NET:
    ETH:            # Wired network
      UPLOAD:       # Upload speed
      DOWNLOAD:     # Download speed
      UPLOADED:     # Total uploaded
      DOWNLOADED:   # Total downloaded
    WLO:            # Wireless (same options)
  DATE:
    DAY:            # Date (FORMAT: short|medium|long|full)
    HOUR:           # Time (FORMAT: short|medium|long|full)
```

## Creating a New Theme

1. Create a folder in `res/themes/` with your theme name
2. Add a `background.png` (800×480 for 5" landscape)
3. Create `theme.yaml` — copy from an existing theme and customize
4. Set your theme in `conf/config.yaml`: `theme: YourThemeName`

## Tips

- `BACKGROUND_IMAGE` crops the background at the sensor's (X, Y) position — this creates a "transparent" effect where the background shows through behind the text.
- For video themes, use a dark/transparent foreground image so sensors are visible over the video.
- Fonts are loaded from `res/fonts/`. Use TTF or OTF format.
- Colors are specified as `R, G, B` (0-255).
