# Turing Screen — Theme System Documentation

## Overview

The Turing Smart Screen Go app displays hardware sensor data on a 5" (800x480) or 3.5" (320x480) LCD connected via USB serial. Themes define what is displayed, where, and how.

A theme is a directory in `res/themes/<name>/` containing:
- `theme.yaml` — configuration (sensors, positions, fonts, colors)
- `background.png` — the composed overlay image (transparent areas show video through)
- `assets/` — additional images (layers, overlays)
- `video/` — optional video file (.mp4)
- `fonts/` — optional fonts (referenced from `res/fonts/`)

---

## Theme YAML Structure

```yaml
---
display:
  SIZE: 5"                    # 3.5" or 5"
  ORIENTATION: landscape      # landscape or portrait
  RGB_LED: 255, 128, 0       # Backplate LED color (R, G, B)

video:
  PATH: video/earth.mp4      # Relative to theme dir
  X: 0
  Y: 0
  WIDTH: 800
  HEIGHT: 480

static_images:
  BACKGROUND:                 # Drawn first (base layer)
    PATH: assets/image_3.png
    X: 0
    Y: 0
    WIDTH: 800
    HEIGHT: 480
  LAYER_2:                    # Drawn on top (alphabetical order)
    PATH: assets/image_5.png
    X: 34
    Y: 29
    WIDTH: 719
    HEIGHT: 431

static_texts:
  CPU_MODEL:
    TEXT: "{{CPU_MODEL}}"     # Replaced at runtime with actual model
    X: 100
    Y: 30
    FONT: geforce/GeForce-Bold.ttf
    FONT_SIZE: 20
    FONT_COLOR: 255, 255, 255

STATS:
  CPU:
    PERCENTAGE:
      INTERVAL: 1
      TEXT:
        SHOW: true
        X: 100
        Y: 80
        FONT: jetbrains-mono/JetBrainsMono-Bold.ttf
        FONT_SIZE: 24
        FONT_COLOR: 255, 255, 255
        SHOW_UNIT: true
        ALIGN: center
      RADIAL:
        SHOW: true
        X: 124             # CENTER of the radial
        Y: 127             # CENTER of the radial
        RADIUS: 65
        WIDTH: 12          # Arc thickness
        MIN_VALUE: 0
        MAX_VALUE: 100
        ANGLE_START: 120
        ANGLE_END: 60
        ANGLE_STEPS: 1
        ANGLE_SEP: 0
        CLOCKWISE: true
        BAR_COLOR: 255, 255, 255
        SHOW_TEXT: false
        SHOW_UNIT: false
      GRAPH:
        SHOW: true
        X: 200             # Left edge of bar
        Y: 100             # Top edge of bar
        WIDTH: 150
        HEIGHT: 15
        MIN_VALUE: 0
        MAX_VALUE: 100
        BAR_COLOR: 255, 255, 255
        BAR_OUTLINE: false
```

---

## Available Sensors (STATS)

### CPU
| Field | Type | Description |
|-------|------|-------------|
| `PERCENTAGE` | Mesurement | CPU usage % |
| `FREQUENCY` | Mesurement | CPU clock speed (MHz/GHz) |
| `TEMPERATURE` | Mesurement | CPU temp (°C) |
| `LOAD` | Load | Load averages (ONE/FIVE/FIFTEEN) |
| `FAN` | Mesurement | CPU fan speed (RPM) |
| `POWER` | Mesurement | CPU power draw (W) |
| `VOLTAGE` | Mesurement | CPU voltage (V) |

### GPU
| Field | Type | Description |
|-------|------|-------------|
| `PERCENTAGE` | Mesurement | GPU usage % |
| `TEMPERATURE` | Mesurement | GPU temp (°C) |
| `MEMORY` | Mesurement | VRAM usage % |
| `POWER` | Mesurement | GPU power draw (W) |
| `FREQUENCY` | Mesurement | GPU core clock (MHz) |
| `VOLTAGE` | Mesurement | GPU voltage (mV) |
| `FAN` | Mesurement | GPU fan speed (RPM) |

### MEMORY
| Field | Type | Description |
|-------|------|-------------|
| `VIRTUAL.USED` | Text | Used memory (bytes formatted) |
| `VIRTUAL.FREE` | Text | Free memory |
| `VIRTUAL.PERCENT_TEXT` | Text | Usage percentage |
| `VIRTUAL.GRAPH` | Graph | Bar visualization |
| `VIRTUAL.RADIAL` | Radial | Arc visualization |
| `SWAP.*` | Same as VIRTUAL | Swap memory |

### DISK
| Field | Type | Description |
|-------|------|-------------|
| `USED` | Mesurement | Disk usage % |
| `FREE` | Mesurement | Free space |
| `TOTAL` | Mesurement | Total space |
| `TEMPERATURE` | Mesurement | Disk temp (NVMe) |

### NET
| Field | Type | Description |
|-------|------|-------------|
| `ETH.UPLOAD.TEXT` | Text | Ethernet upload speed |
| `ETH.DOWNLOAD.TEXT` | Text | Ethernet download speed |
| `ETH.UPLOADED.TEXT` | Text | Total uploaded |
| `ETH.DOWNLOADED.TEXT` | Text | Total downloaded |
| `WLO.*` | Same as ETH | WiFi interface |

### DATE
| Field | Type | Description |
|-------|------|-------------|
| `DAY.TEXT` | Text | Date (FORMAT: short/medium/long/full) |
| `HOUR.TEXT` | Text | Time (FORMAT: short/medium/long/full) |

### WEATHER
| Field | Type | Description |
|-------|------|-------------|
| `TEMPERATURE` | Mesurement | Temperature (°C) |
| `CONDITION` | Text | Full condition string ("overcast, 15°C, Wind 10km/h") |

### VOLUME
| Field | Type | Description |
|-------|------|-------------|
| `TEXT` | Text | Volume % or "MUTE" |

---

## Widget Types

### Mesurement (container)
Contains one or more display widgets for the same data:
```yaml
PERCENTAGE:
  INTERVAL: 1        # Update interval in seconds
  TEXT: {...}        # Numeric text display
  GRAPH: {...}      # Progress bar
  RADIAL: {...}     # Arc/radial gauge
  PERCENT_TEXT: {...} # Alternative text (shows %)
```

When multiple widgets (e.g. RADIAL + TEXT) overlap, they are **automatically composed** into a single image update. The system calculates bounding box intersection and combines overlapping widgets.

### TEXT
```yaml
TEXT:
  SHOW: true
  X: 100               # Position
  Y: 50
  FONT: path/font.ttf  # Relative to res/fonts/
  FONT_SIZE: 20        # In pixels
  FONT_COLOR: R, G, B
  SHOW_UNIT: true      # Append unit (%, °C, W, etc)
  ALIGN: center        # left | center | right
```

**ALIGN behavior:**
- `left` (default for manually created themes): X = left edge of text
- `center` (default for .turtheme converted): X = center point, crop starts at X - width/2
- `right`: X = right edge

### GRAPH (Progress Bar)
```yaml
GRAPH:
  SHOW: true
  X: 200           # Left edge
  Y: 100           # Top edge
  WIDTH: 150       # Bar width in pixels
  HEIGHT: 15       # Bar height in pixels
  MIN_VALUE: 0
  MAX_VALUE: 100
  BAR_COLOR: R, G, B
  BAR_OUTLINE: false
```

### RADIAL (Arc/Gauge)
```yaml
RADIAL:
  SHOW: true
  X: 124           # CENTER of arc (not top-left!)
  Y: 127           # CENTER of arc
  RADIUS: 65       # Arc radius
  WIDTH: 12        # Arc thickness
  MIN_VALUE: 0
  MAX_VALUE: 100
  ANGLE_START: 120
  ANGLE_END: 60
  ANGLE_STEPS: 1
  ANGLE_SEP: 0
  CLOCKWISE: true
  BAR_COLOR: R, G, B
  SHOW_TEXT: false  # Show value in center
  SHOW_UNIT: false
```

---

## Static Texts (Placeholders)

The `static_texts` section supports runtime replacement:

| Placeholder | Replaced with |
|-------------|---------------|
| `{{CPU_MODEL}}` | e.g. "Ryzen 9 5900X" |
| `{{GPU_MODEL}}` | e.g. "RX 6700" |
| `{{MEM_TOTAL}}` | e.g. "31 GB" |
| `{{DISK_MODEL}}` | e.g. "SKC3000S 1TB" |
| `{{HOSTNAME}}` | e.g. "pczao" |

---

## Static Images (Layering)

Images are drawn in **alphabetical order** by key name. Use naming to control z-order:

```yaml
static_images:
  BACKGROUND:          # Drawn first (bottom layer)
    PATH: assets/image_1.png
    X: 0
    Y: 0
    WIDTH: 800
    HEIGHT: 480
  LAYER_2:             # Drawn second (on top)
    PATH: assets/image_2.png
    X: 34              # Can be positioned anywhere
    Y: 29
    WIDTH: 719         # Can be smaller than display
    HEIGHT: 431
```

The composed result becomes the **background** for all sensor updates. Each sensor update crops the relevant region from this composed background before drawing text/widgets on top.

---

## .turtheme Conversion

### Binary Format
`.turtheme` files are .NET BinaryFormatter (NRBF) serialized `UsbMonitorL.Theme` objects containing:
- Theme metadata (name, resolution, video reference)
- Embedded images (PNG) as `System.Drawing.Bitmap`
- `GraphItem` list (sensors with position, font, color)
- `GraphImage` items (static image layers with position)
- `GraphStatuBar` items (progress bars)
- `GraphArchBar` items (radial gauges)
- `GraphAnimation` (video overlay configuration)

### Image Order in .turtheme Binary
Images are extracted sequentially by PNG signature scanning:

| Index | Source | Purpose | Use? |
|-------|--------|---------|------|
| image_0 | Theme.themePic | Preview (all composed) | ❌ Skip |
| image_1 | Animation.bitmap | Video overlay (scaled) | ✅ Background for video themes |
| image_2 | Animation.O_bitmap | Original copy | ❌ Skip |
| image_3 | Animation.S_bitmap | Scaled copy | ❌ Skip |
| image_4 | GraphImage[0].bitmap | Layer 1 (scaled/positioned) | ✅ Use |
| image_5 | GraphImage[0].O_bitmap | Original (oversized) | ❌ Skip |
| image_6 | GraphImage[1].bitmap | Layer 2 | ✅ Use |
| ... | ... | ... | ... |

**Pattern:**
- With video: skip 0,1,2,3 → use image_4, image_6, image_8... (every other starting at 4)
- Without video: skip 0 → use image_1, image_3, image_5... (every other starting at 1)

### Color Handling
.NET `System.Drawing.Color` has 3 states:
- `state=0`: empty (value=0)
- `state=1`: **KnownColor** index (lookup table, NOT the value field!)
- `state=2`: explicit ARGB in value field

Common KnownColors: 34=Black, 164=White, 141=Red, 37=Blue, 79=Green, 78=Gray

### Sensor Mapping (.turtheme DataName → Go STATS)

| DataName | Maps to |
|----------|---------|
| CPULOAD | CPU.PERCENTAGE |
| CPUTEMP | CPU.TEMPERATURE |
| CPUCLOCK | CPU.FREQUENCY |
| CPUPWR | CPU.POWER |
| CPUFAN | CPU.FAN |
| CPUVOLTAGE | CPU.VOLTAGE |
| CPUMODEL | static_text {{CPU_MODEL}} |
| GPULOAD | GPU.PERCENTAGE |
| GPUTEMP | GPU.TEMPERATURE |
| GPUPWR | GPU.POWER |
| GPURAM / GPURAMLOAD / GPUVALIDRAM | GPU.MEMORY |
| GPUCLOCK | GPU.FREQUENCY |
| GPUFAN | GPU.FAN |
| GPUVOLTAGE | GPU.VOLTAGE |
| GPUMODEL | static_text {{GPU_MODEL}} |
| RAMLOAD | MEMORY.VIRTUAL.PERCENT_TEXT |
| RAM / RAMVALID | MEMORY.VIRTUAL.USED |
| RAMMODEL | static_text {{MEM_TOTAL}} |
| DRVLOAD / DRVCLOAD | DISK.USED.PERCENT_TEXT |
| HDDTEMP | DISK.TEMPERATURE |
| UPSPEED | NET.ETH.UPLOAD |
| DOWNDSPEED | NET.ETH.DOWNLOAD |
| TIME | DATE.HOUR |
| DATE / DAY | DATE.DAY |
| Weather | WEATHER.TEMPERATURE |
| Volume | VOLUME |

### Widget Mapping (.turtheme TypeName → Go widget)

| TypeName | Maps to |
|----------|---------|
| Data | TEXT (sensor value) |
| Text | static_text (fixed label) |
| Image | static_images layer |
| StatuBar | GRAPH (progress bar) |
| ArchBar | RADIAL (arc gauge) |
| Animation | Video overlay config |

### Position Differences

| Element | .turtheme | Go app |
|---------|-----------|--------|
| TEXT | X,Y = position (default center-aligned) | X,Y depends on ALIGN field |
| RADIAL | X,Y = top-left of bounding box | X,Y = CENTER of arc |
| GRAPH | X,Y = top-left | X,Y = top-left (same) |
| Image | X,Y = top-left | X,Y = top-left (same) |

**RADIAL conversion:** `Go.X = turtheme.posX + radius`, `Go.Y = turtheme.posY + radius`

### Font Size
.turtheme stores font size in **points (pt)**. Go app uses **pixels (px)**.
Conversion factor: `px = pt * 1.5`

### Font Resolution
Fonts in .turtheme are referenced by name (e.g. "GeForce", "微软雅黑").
Resolved to files in `res/fonts/`:
- `GeForce` → `geforce/GeForce-Bold.ttf`
- System fonts (Arial, 微软雅黑) → fallback to `jetbrains-mono/JetBrainsMono-Bold.ttf`
- Custom fonts → copied to `res/fonts/turtheme/`

---

## Config (conf/config.yaml)

```yaml
device:
  port: AUTO
  theme: Technology_theme
  log: debug
  turn_off_on_exit: false
  sensors:
    network:
      eth: "enp3s0"
      wlo: "wlp4s0"
    cpu:
      temperature_sensor: "auto"
    disk:
      temperature_sensor: "auto"
    gpu:
      provider: "auto"        # auto | amd | nvidia | none
    weather:
      enabled: true
      city: "Sao Paulo,BR"   # Geocoded via Open-Meteo (no API key needed)
      interval: 30m
  display:
    width: 800
    height: 480
    reverse: false
    brightness: 20
```

---

## Runtime Flow

1. Load theme YAML
2. Compose background: `static_images` (layered) + `static_texts` (with placeholder replacement)
3. Set composed image as Builder background
4. Send background to device (DISPLAY_BITMAP or video overlay)
5. Start sensor goroutines — each polls at its INTERVAL:
   - Read sensor value
   - Render widget (TEXT/GRAPH/RADIAL) on top of background crop
   - If multiple widgets overlap → compose into single update
   - Send UPDATE_BITMAP to device

---

## Conversion Scripts

```bash
# Extract all .turtheme files
python3 FromApp/extract_turtheme.py FromApp/5inchthemeENG/5inchthemeENG/theme/800480 --all -o FromApp/extracted_all

# Convert all extracted themes to Go format
python3 FromApp/convert_turtheme.py --all

# Copy to res/themes/
python3 -c "
import shutil, yaml
from pathlib import Path
src = Path('FromApp/converted_themes')
dst = Path('res/themes')
for d in sorted(src.iterdir()):
    if not d.is_dir(): continue
    y = d / 'theme.yaml'
    if not y.exists(): continue
    data = yaml.safe_load(y.read_text())
    if not data or not data.get('STATS'): continue
    target = dst / d.name
    if target.exists(): shutil.rmtree(target)
    shutil.copytree(d, target)
"
```
