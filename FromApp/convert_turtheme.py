#!/usr/bin/env python3
"""
Convert .turtheme extracted data to Go project theme.yaml format.

Reads the extracted theme.yaml (from extract_turtheme.py) and produces a valid
theme.yaml that matches the Go turing-screen project format.

Uses REAL positions, font sizes, and colors from the extracted data instead of
generic layout defaults.

Usage:
    python3 convert_turtheme.py --all              # Convert all themes
    python3 convert_turtheme.py --theme <name>     # Convert a single theme
    python3 convert_turtheme.py --list             # List available themes
"""

import argparse
import re
import shutil
import subprocess
import sys
from pathlib import Path

import yaml

# Optional: auto-translate Chinese text to English
try:
    from deep_translator import GoogleTranslator as _GT
    _cn_translator = _GT(source="zh-CN", target="en")
except ImportError:
    _cn_translator = None

_translate_cache: dict[str, str] = {}


def _has_chinese(text: str) -> bool:
    return any("一" <= c <= "鿿" for c in text)


def _auto_translate(text: str) -> str:
    """Translate Chinese text to English (cached). Falls back to original on error."""
    if not _has_chinese(text):
        return text
    if text in _translate_cache:
        return _translate_cache[text]
    if _cn_translator is None:
        return text
    try:
        result = _cn_translator.translate(text) or text
    except Exception:
        result = text
    _translate_cache[text] = result
    return result

SCRIPT_DIR = Path(__file__).resolve().parent
EXTRACTED_DIR = SCRIPT_DIR / "extracted_52inch"
OUTPUT_DIR = SCRIPT_DIR / "converted_themes"
VIDEO_DIR = SCRIPT_DIR / "52inchthemeENG" / "video" / "7201280"
FONTS_DIR = SCRIPT_DIR / "52inchENG" / "TURZX-V3.1.0-52inchENG" / "fonts"

# Default font to use when the original font isn't available
DEFAULT_FONT = "jetbrains-mono/JetBrainsMono-Bold.ttf"

# Mapping from .turtheme font name -> path relative to res/fonts/
# The Go app loads fonts from: res/fonts/ + <value from FONT field in YAML>
FONT_MAP = {
    "geforce": "geforce/GeForce-Bold.ttf",
    "motorblock": "turtheme/Motorblock.ttf",
    "liquid crystal": "turtheme/LiquidCrystal-Normal.otf",
    "race space regular 1": "turtheme/RACESPACEREGULAR.otf",
    "futura lt condensedextrabold": "turtheme/Futura LT Condensed Extra Bold.ttf",
    "futura lt book": "turtheme/Futura LT Book.ttf",
    "helveticaneuelt pro 35 th": "turtheme/HelveticaNeueLTPro-Th.otf",
    "helveticaneuelt pro 45 lt": "turtheme/HelveticaNeueLTPro-Lt.otf",
    "helveticaneuelt pro 55 roman": "turtheme/HelveticaNeueLTPro-Roman.otf",
    "helveticaneuelt pro 65 md": "turtheme/HelveticaNeueLTPro-Md.otf",
    "helveticaneuelt pro 67 mdcn": "turtheme/HelveticaNeueLTPro-MdCn_1.otf",
    "helveticaneuelt pro 75 bd": "turtheme/HelveticaNeueLTPro-Bd.otf",
    "helveticaneuelt pro 85 hv": "turtheme/HelveticaNeueLTPro-Hv.otf",
    "helveticaneuelt pro 93 blkex": "turtheme/HelveticaNeueLTStd-BlkEx.otf",
    "helveticaneuelt pro 95 blk": "turtheme/HelveticaNeueLTPro-Blk.otf",
    "helveticaneuelt pro 97 blkcn": "turtheme/HelveticaNeueLTPro-BlkCn.otf",
    "kumbh sans": "turtheme/KumbhSans-Regular.ttf",
    "kumbh sans bold": "turtheme/KumbhSans-Bold.ttf",
    "kumbh sans medium": "turtheme/KumbhSans-Medium.ttf",
    "kumbh sans semibold": "turtheme/KumbhSans-SemiBold.ttf",
    "kumbh sans black": "turtheme/KumbhSans-Black.ttf",
    "harmonyo sans": "turtheme/HarmonyOS_Sans_Regular.ttf",
    "harmonyos sans": "turtheme/HarmonyOS_Sans_Regular.ttf",
    "harmonyos sans bold": "turtheme/HarmonyOS_Sans_Bold.ttf",
    "harmonyos sans medium": "turtheme/HarmonyOS_Sans_Medium.ttf",
    "harmonyos sans black": "turtheme/HarmonyOS_Sans_Black.ttf",
    "harmonyos sans light": "turtheme/HarmonyOS_Sans_Light.ttf",
    "azonix": "turtheme/Azonix.otf",
    "quantum": "turtheme/Quantum.otf",
    "alien encounters": "turtheme/ALIEN-ENCOUNTERS-REGULAR.TTF",
    "i fink u freeky": "turtheme/i_fink_u_freeky.ttf",
    "justbubble": "turtheme/JustBubble.ttf",
    "perrygothic": "turtheme/PERRYGOT.TTF",
    "gensenmarugothictw bold": "turtheme/GenSenMaruGothicTW-Bold.ttf",
    "gensenmarugothictw heavy": "turtheme/GENSENMARUGOTHICTW-HEAVY.TTF",
    "思源黑体 cn bold": "turtheme/Source-San-Hans-BLOD.ttf",
    "思源黑体": "turtheme/Source-San-Hans-BLOD.ttf",
    "source san hans": "turtheme/Source-San-Hans-BLOD.ttf",
}

# Reverse map: font filename -> source file in FONTS_DIR
FONT_SOURCE_FILES = {
    "Motorblock.ttf": "Motorblock.ttf",
    "LiquidCrystal-Normal.otf": "LiquidCrystal-Normal.otf",
    "RACESPACEREGULAR.otf": "RACESPACEREGULAR.otf",
    "Futura LT Condensed Extra Bold.ttf": "Futura LT Condensed Extra Bold.ttf",
    "Futura LT Book.ttf": "Futura LT Book.ttf",
    "HelveticaNeueLTPro-Th.otf": "HelveticaNeueLTPro-Th.otf",
    "HelveticaNeueLTPro-Lt.otf": "HelveticaNeueLTPro-Lt.otf",
    "HelveticaNeueLTPro-Roman.otf": "HelveticaNeueLTPro-Roman.otf",
    "HelveticaNeueLTPro-Md.otf": "HelveticaNeueLTPro-Md.otf",
    "HelveticaNeueLTPro-MdCn_1.otf": "HelveticaNeueLTPro-MdCn_1.otf",
    "HelveticaNeueLTPro-Bd.otf": "HelveticaNeueLTPro-Bd.otf",
    "HelveticaNeueLTPro-Hv.otf": "HelveticaNeueLTPro-Hv.otf",
    "HelveticaNeueLTStd-BlkEx.otf": "HelveticaNeueLTStd-BlkEx.otf",
    "HelveticaNeueLTPro-Blk.otf": "HelveticaNeueLTPro-Blk.otf",
    "HelveticaNeueLTPro-BlkCn.otf": "HelveticaNeueLTPro-BlkCn.otf",
    "KumbhSans-Regular.ttf": "KumbhSans-Regular.ttf",
    "KumbhSans-Bold.ttf": "KumbhSans-Bold.ttf",
    "KumbhSans-Medium.ttf": "KumbhSans-Medium.ttf",
    "KumbhSans-SemiBold.ttf": "KumbhSans-SemiBold.ttf",
    "KumbhSans-Black.ttf": "KumbhSans-Black.ttf",
    "HarmonyOS_Sans_Regular.ttf": "HarmonyOS_Sans_Regular.ttf",
    "HarmonyOS_Sans_Bold.ttf": "HarmonyOS_Sans_Bold.ttf",
    "HarmonyOS_Sans_Medium.ttf": "HarmonyOS_Sans_Medium.ttf",
    "HarmonyOS_Sans_Black.ttf": "HarmonyOS_Sans_Black.ttf",
    "HarmonyOS_Sans_Light.ttf": "HarmonyOS_Sans_Light.ttf",
    "Azonix.otf": "Azonix.otf",
    "Quantum.otf": "Quantum.otf",
    "ALIEN-ENCOUNTERS-REGULAR.TTF": "ALIEN-ENCOUNTERS-REGULAR.TTF",
    "i_fink_u_freeky.ttf": "i_fink_u_freeky.ttf",
    "JustBubble.ttf": "JustBubble.ttf",
    "PERRYGOT.TTF": "PERRYGOT.TTF",
    "GenSenMaruGothicTW-Bold.ttf": "GenSenMaruGothicTW-Bold.ttf",
    "GENSENMARUGOTHICTW-HEAVY.TTF": "GENSENMARUGOTHICTW-HEAVY.TTF",
    "Source-San-Hans-BLOD.ttf": "Source San Hans BLOD.ttf",
}


def resolve_font(font_name: str) -> str:
    """Resolve a .turtheme font name to a path relative to res/fonts/.

    Returns a path like 'geforce/GeForce-Bold.ttf' or 'turtheme/Motorblock.ttf'.
    Falls back to DEFAULT_FONT for system fonts (Arial, 微软雅黑, etc).
    """
    if not font_name:
        return DEFAULT_FONT

    key = font_name.lower().strip()

    # Direct match
    if key in FONT_MAP:
        return FONT_MAP[key]

    # Partial match (font name contains key or vice versa)
    for map_key, map_val in FONT_MAP.items():
        if map_key in key or key in map_key:
            return map_val

    # System fonts that aren't distributable - use default
    return DEFAULT_FONT

# Mapping from .turtheme DataName to Go STATS path
# Format: data_name -> (category, subcategory, display_widget)
# display_widget: "TEXT", "PERCENT_TEXT", "GRAPH", "RADIAL"
DATANAME_MAP = {
    # CPU sensors
    "CPULOAD":    ("CPU", "PERCENTAGE", "TEXT"),
    "CPUTEMP":    ("CPU", "TEMPERATURE", "TEXT"),
    "CPUCLOCK":   ("CPU", "FREQUENCY", "TEXT"),
    "CPUPWR":     ("CPU", "POWER", "TEXT"),
    "CPUFAN":     ("CPU", "FAN", "TEXT"),
    "CPUVOLTAGE": ("CPU", "VOLTAGE", "TEXT"),
    "CPUMODEL":   ("CPU",    "MODEL", "TEXT"),
    # GPU sensors
    "GPULOAD":     ("GPU", "PERCENTAGE", "TEXT"),
    "GPUTEMP":     ("GPU", "TEMPERATURE", "TEXT"),
    "GPUPWR":      ("GPU", "POWER", "TEXT"),
    "GPURAM":      ("GPU", "MEMORY", "TEXT"),
    "GPURAMLOAD":  ("GPU", "MEMORY", "TEXT"),
    "GPUVALIDRAM": ("GPU", "MEMORY", "TEXT"),
    "GPUCLOCK":    ("GPU", "FREQUENCY", "TEXT"),
    "GPUFAN":      ("GPU", "FAN", "TEXT"),
    "GPUVOLTAGE":  ("GPU", "VOLTAGE", "TEXT"),
    "GPUMODEL":   ("GPU",    "MODEL", "TEXT"),
    # Memory sensors — all live under MEMORY.RAM
    "RAMLOAD":    ("MEMORY", "RAM", "PERCENT_TEXT"),
    # RAMVALID (available RAM in GB) and RAM (used RAM in GB): no direct equivalent.
    "RAMVALID":   None,
    "RAM":        None,
    # RAMMODEL maps to the SIZE sub-sensor (hwinfo provides MemTotal at runtime).
    "RAMMODEL":   ("MEMORY", "RAM.SIZE", "TEXT"),
    # Disk sensors
    "DRVLOAD":    ("DISK", "USED", "PERCENT_TEXT"),
    "DRVCLOAD":   ("DISK", "USED", "PERCENT_TEXT"),
    "HDDTEMP":    ("DISK", "TEMPERATURE", "TEXT"),
    "DRVMODEL":   ("DISK", "MODEL", "TEXT"),
    # Host / system sensors
    "HOSTNAME":   ("HOST", "HOSTNAME", "TEXT"),
    # Network — direction embedded in sub so it survives display-type override
    "UPSPEED":    ("NET", "ETH.UPLOAD",   "TEXT"),
    "DOWNDSPEED": ("NET", "ETH.DOWNLOAD", "TEXT"),
    # Date/Time
    "TIME":  ("DATE", "HOUR", "TEXT"),
    "DATE":  ("DATE", "DAY", "TEXT"),
    "DAY":   ("DATE", "DAY", "TEXT"),
    # Other
    "FPS":      None,  # No equivalent
    "Weather":  ("WEATHER", "TEMPERATURE", "TEXT"),
    "Volume":   ("VOLUME", None, "TEXT"),
}


# Mapping from DataName to Go template placeholder

# Translation map for Chinese labels found in .turtheme files
LABEL_TRANSLATIONS = {
    # Sensor display names (from M_Data.DisplayName)
    "天气": "Weather",
    "音量": "Volume",
    "Cpu温度": "CPU Temp",
    "Cpu利用率": "CPU Usage",
    "Cpu频率": "CPU Freq",
    "Cpu电压": "CPU Voltage",
    "Cpu功耗": "CPU Power",
    "Cpu风扇": "CPU Fan",
    "显卡温度": "GPU Temp",
    "显卡利用率": "GPU Usage",
    "显卡频率": "GPU Freq",
    "显卡电压": "GPU Voltage",
    "显卡功耗": "GPU Power",
    "显卡风扇": "GPU Fan",
    "显卡型号": "GPU Model",
    "Cpu型号": "CPU Model",
    "内存型号": "RAM Model",
    "内存利用率": "RAM Usage",
    "已用内存": "RAM Used",
    "上传速度": "Upload",
    "下载速度": "Download",
    "日期": "Date",
    "时间": "Time",
    "硬盘温度": "Disk Temp",
    "硬盘温度_1": "Disk Temp",
    "磁盘利用率": "Disk Usage",
    "静态文字": "Label",
    # Common static text labels
    "风扇": "Fan",
    "频率": "Freq",
    "电压": "Voltage",
    "功耗": "Power",
    "温度": "Temp",
    "利用率": "Usage",
    "已用": "Used",
    "总共": "Total",
    "可用": "Free",
}


def translate_label(text: str) -> str:
    """Translate Chinese labels to English (dict first, API fallback)."""
    if not text:
        return text
    # Direct match
    if text in LABEL_TRANSLATIONS:
        return LABEL_TRANSLATIONS[text]
    # Partial match for compound labels
    result = text
    for cn, en in LABEL_TRANSLATIONS.items():
        if cn in result:
            result = result.replace(cn, en)
    # If still has Chinese, call translation API
    if _has_chinese(result):
        result = _auto_translate(result)
    return result

# Font size conversion factor: .turtheme uses points (pt), our Go app uses pixels.
# Standard conversion: 1pt = 1.333px at 96 DPI (DPI/72 = 96/72 ≈ 1.33).
FONT_SIZE_FACTOR = 1.33


def sanitize_name(name: str) -> str:
    """Sanitize theme name for use as directory name, translating Chinese first."""
    if _has_chinese(name):
        name = _auto_translate(name)
    name = re.sub(r'[^\w\s-]', '', name).strip()
    return re.sub(r'[\s]+', '_', name)


def get_display_info(width: int, height: int) -> tuple[str, str]:
    """Determine display SIZE and ORIENTATION from resolution."""
    if (width, height) in [(320, 480), (480, 320)]:
        size = '3.5"'
    else:
        size = '5"'
    # For TURZX: portrait themes (height > width) use reverse_portrait so the
    # driver sends them without rotation (REVERSE_PORTRAIT = native, no flip).
    # Landscape themes stay as landscape.
    if width >= height:
        orientation = "landscape"
    else:
        orientation = "reverse_portrait"
    return size, orientation


def scale_font_size(pt_size: int) -> int:
    """Convert font size from .NET points to pixels for our Go app."""
    return max(8, int(round(pt_size * FONT_SIZE_FACTOR)))





def pick_accent_color(extracted: dict) -> str:
    """Pick the dominant font color from extracted graph items as RGB string."""
    # Count font colors from graph_items
    color_counts = {}
    for item in extracted.get("graph_items", []):
        fc = item.get("font_color")
        if fc:
            color_counts[fc] = color_counts.get(fc, 0) + 1

    if color_counts:
        # Return most common font color
        best = max(color_counts, key=color_counts.get)
        return best

    # Fallback: use front_color or set_color from theme metadata
    if extracted.get("front_color"):
        return extracted["front_color"]
    if extracted.get("set_color"):
        return extracted["set_color"]
    return "255, 255, 255"


def make_text_entry(item: dict, width: int, height: int) -> dict:
    """Create a STATS TEXT entry from a graph item."""
    entry = {
        "SHOW": True,
        "X": item.get("x", 0),
        "Y": item.get("y", 0),
        "FONT": DEFAULT_FONT,
        "FONT_SIZE": scale_font_size(item.get("font_size", 14)),
        "FONT_COLOR": item.get("font_color", "255, 255, 255"),
    }
    if item.get("show_unit") is True:
        entry["SHOW_UNIT"] = True
    elif item.get("show_unit") is False:
        entry["SHOW_UNIT"] = False
    return entry


def convert_theme(theme_name: str) -> bool:
    """Convert a single theme from extracted format to Go theme.yaml format."""
    extracted_path = EXTRACTED_DIR / theme_name / "theme.yaml"
    if not extracted_path.exists():
        print(f"  SKIP: {theme_name} - no theme.yaml found")
        return False

    with open(extracted_path, 'r', encoding='utf-8') as f:
        extracted = yaml.safe_load(f)

    if not extracted:
        print(f"  SKIP: {theme_name} - empty or invalid YAML")
        return False

    width = extracted.get("width", 800)
    height = extracted.get("height", 480)
    graph_items = extracted.get("graph_items", [])
    size, orientation = get_display_info(width, height)
    accent = pick_accent_color(extracted)

    # Sanitized output directory
    out_name = sanitize_name(theme_name)
    output_dir = OUTPUT_DIR / out_name
    output_dir.mkdir(parents=True, exist_ok=True)

    # Copy assets
    assets_src = EXTRACTED_DIR / theme_name / "assets"
    if assets_src.exists():
        assets_dst = output_dir / "assets"
        if assets_dst.exists():
            shutil.rmtree(assets_dst)
        shutil.copytree(assets_src, assets_dst)
        # Classify images: opaque (background) vs transparent (overlay)
        bg_root = output_dir / "background.png"
        png_files = sorted(
            [f for f in assets_dst.iterdir() if f.suffix.lower() == '.png'],
            key=lambda f: f.name,
        )
        opaque_images = []
        overlay_images = []
        try:
            from PIL import Image
            import numpy as np
            for f in png_files:
                # Skip image_0 — it's the preview/thumbnail (all layers composited)
                if f.name == "image_0.png":
                    continue
                img = Image.open(f)
                w_ok = img.size[0] <= width + 1
                h_ok = img.size[1] <= height + 1
                if not (w_ok and h_ok):
                    continue  # skip images larger than display
                if img.mode == 'RGBA':
                    alpha = np.array(img)[:, :, 3]
                    transparent_pct = (alpha < 255).sum() / alpha.size
                    if transparent_pct > 0.3:
                        overlay_images.append(f)
                    else:
                        opaque_images.append(f)
                else:
                    opaque_images.append(f)
        except ImportError:
            # No PIL, fallback: skip image_0, use next largest
            png_files_sorted = sorted(
                [f for f in png_files if f.name != "image_0.png"],
                key=lambda f: f.stat().st_size, reverse=True,
            )
            if png_files_sorted:
                opaque_images = [png_files_sorted[0]]

        # Pick background: first overlay for video themes, first opaque otherwise
        has_video = bool(extracted.get("video_name"))
        if has_video and overlay_images:
            shutil.copy2(overlay_images[0], bg_root)
        elif opaque_images:
            shutil.copy2(opaque_images[0], bg_root)
        elif overlay_images:
            shutil.copy2(overlay_images[0], bg_root)
        elif png_files:
            shutil.copy2(png_files[0], bg_root)

    # Copy video if referenced and available
    video_name = extracted.get("video_name")
    video_copied = False
    if video_name:
        video_src = VIDEO_DIR / video_name
        if video_src.exists():
            video_dst_dir = output_dir / "video"
            video_dst_dir.mkdir(parents=True, exist_ok=True)

            # Copy the MP4
            video_dst = video_dst_dir / video_name
            if not video_dst.exists():
                shutil.copy2(video_src, video_dst)
            video_copied = True

            # Copy the pre-extracted H264 if it exists (named {mp4_name}{timestamp}.h264).
            # The daemon expects {stem}.h264 alongside the mp4 as its cache file.
            stem = Path(video_name).stem  # e.g. "EVANGELION01" from "EVANGELION01.mp4"
            h264_dst = video_dst_dir / f"{stem}.h264"
            if not h264_dst.exists():
                candidates = sorted(VIDEO_DIR.glob(f"{video_name}*.h264"))
                if candidates:
                    shutil.copy2(candidates[0], h264_dst)

            # Extract a preview frame (assets/image_0.png) from the video using ffmpeg.
            # Only if image_0.png doesn't already exist from a bitmap in the theme.
            assets_dir = output_dir / "assets"
            preview_dst = assets_dir / "image_0.png"
            if not preview_dst.exists():
                assets_dir.mkdir(parents=True, exist_ok=True)
                try:
                    subprocess.run(
                        ["ffmpeg", "-y", "-ss", "00:00:01",
                         "-i", str(video_dst),
                         "-vframes", "1", "-f", "image2",
                         str(preview_dst)],
                        capture_output=True, check=True,
                    )
                except Exception as e:
                    print(f"  WARN: could not extract video preview frame: {e}")

    # Collect all font names used in this theme and resolve them
    used_fonts = set()
    for item in graph_items:
        font_name = item.get("font", "")
        if font_name:
            used_fonts.add(font_name)

    # Build font path mapping: original name -> resolved path for YAML
    font_paths = {}
    for fn in used_fonts:
        font_paths[fn] = resolve_font(fn)

    # Copy referenced fonts to res/fonts/turtheme/ (shared across themes)
    res_fonts_dir = SCRIPT_DIR.parent / "res" / "fonts" / "turtheme"
    fonts_copied = set()
    for fn, resolved_path in font_paths.items():
        if resolved_path == DEFAULT_FONT:
            continue
        if not resolved_path.startswith("turtheme/"):
            continue  # Already in res/fonts/ (e.g. geforce/GeForce-Bold.ttf)
        font_filename = Path(resolved_path).name
        # Find the source file
        src_name = FONT_SOURCE_FILES.get(font_filename)
        if not src_name:
            continue
        src = FONTS_DIR / src_name
        if src.exists():
            res_fonts_dir.mkdir(parents=True, exist_ok=True)
            dst = res_fonts_dir / font_filename
            if not dst.exists():
                shutil.copy2(src, dst)
            fonts_copied.add(font_filename)

    # Build theme YAML
    theme = {}

    # --- display ---
    theme["display"] = {
        "SIZE": size,
        "ORIENTATION": orientation,
        "RGB_LED": accent,
        "WIDTH": width,
        "HEIGHT": height,
    }

    # --- video ---
    if video_name and video_copied:
        theme["video"] = {
            "PATH": f"video/{video_name}",
            "X": 0,
            "Y": 0,
            "WIDTH": width,
            "HEIGHT": height,
        }

    # --- static_images ---
    static_imgs = {}
    has_video = bool(extracted.get("video_name"))

    # Get image items (GraphImage) from extracted data for positioning
    image_items = [i for i in graph_items if i.get("type_name") == "Image"]

    # Image mapping from .turtheme binary order:
    #   image_0 = Theme.themePic (preview) -> SKIP
    #   image_1 = GraphAnimation.bitmap (video frame) -> SKIP
    #   image_2 = GraphAnimation.O_bitmap (duplicate) -> SKIP
    #   image_3 = GraphImage[0].bitmap -> USE (layer 1)
    #   image_4 = GraphImage[0].O_bitmap (original, oversized) -> SKIP
    #   image_5 = GraphImage[1].bitmap -> USE (layer 2)
    #   image_6 = GraphImage[1].O_bitmap -> SKIP
    #   ...pattern: use image_(3 + n*2) for GraphImage[n]

    assets_dir = output_dir / "assets"
    all_pngs = sorted(
        [f for f in assets_dir.iterdir() if f.suffix.lower() == '.png'],
        key=lambda f: f.name,
    ) if assets_dir.exists() else []

    if image_items and len(all_pngs) > 1:
        # Use GraphImage.bitmap images
        # Offset depends on whether theme has video (Animation takes 3 bitmap slots)
        # With Animation: image_0=preview, image_1/2/3=animation(bitmap+O+S), image_4+=GraphImages
        # Without Animation: image_0=preview, image_1+=GraphImages
        has_animation = has_video  # Animation exists when theme has video
        if has_animation:
            start_idx = 4  # skip preview + 3 animation bitmaps
        else:
            start_idx = 1  # skip preview only

        for i, img_item in enumerate(image_items):
            img_idx = start_idx + i * 2  # each GraphImage has bitmap + O_bitmap
            img_name = f"image_{img_idx}.png"
            img_file = assets_dir / img_name
            if not img_file.exists():
                continue
            key = "BACKGROUND" if i == 0 else f"LAYER_{i+1}"
            x = img_item.get("x", 0)
            y = img_item.get("y", 0)
            try:
                from PIL import Image as PILImage
                pimg = PILImage.open(img_file)
                iw, ih = pimg.size
            except Exception:
                iw, ih = width, height
            entry = {"PATH": f"assets/{img_name}", "X": x, "Y": y, "WIDTH": iw, "HEIGHT": ih}
            if i > 0:
                entry["INDEX"] = i  # BACKGROUND is implicitly 0; overlays start at 1
            static_imgs[key] = entry
        # Also copy first layer as background.png for BACKGROUND_IMAGE references
        if image_items:
            first_idx = start_idx
            first_img = assets_dir / f"image_{first_idx}.png"
            if first_img.exists():
                shutil.copy2(first_img, output_dir / "background.png")
    elif has_video:
        # Video theme without GraphImage items.
        # Prefer image_1 (first animation frame) but fall back to image_0 when
        # all frames were identical and the extractor deduplicated them to one file.
        overlay_file = assets_dir / "image_1.png"
        if not overlay_file.exists():
            overlay_file = assets_dir / "image_0.png"
        if overlay_file.exists():
            shutil.copy2(overlay_file, output_dir / "background.png")
        static_imgs["BACKGROUND"] = {
            "PATH": "background.png",
            "X": 0, "Y": 0, "WIDTH": width, "HEIGHT": height,
        }
    else:
        # Fallback: use background.png
        static_imgs["BACKGROUND"] = {
            "PATH": "background.png",
            "X": 0, "Y": 0, "WIDTH": width, "HEIGHT": height,
        }

    theme["static_images"] = static_imgs

    # --- Classify graph items ---
    static_texts = {}
    stats_items = {}  # (category, subcategory, widget) -> item data

    def get_font_path(item):
        """Get resolved font path for an item."""
        fn = item.get("font", "")
        return font_paths.get(fn, DEFAULT_FONT)

    for item_index, item in enumerate(graph_items):
        data_name = item.get("data_name")
        type_name = item.get("type_name")

        # Static text items (TypeName == "Text" or DataName == "StaticText")
        if type_name == "Text" or data_name == "StaticText":
            text_val = item.get("static_text", "")
            if not text_val:
                # Try to extract from display_name: "文字--Something" or "Text--Something"
                dn = item.get("display_name", "")
                parts = dn.split("--", 1)
                text_val = parts[1] if len(parts) > 1 else dn
            text_val = translate_label(text_val)
            base_key = sanitize_name(text_val)
            # Guard against very short keys (e.g. "°C" → "C") that collide easily.
            key = base_key if len(base_key) >= 2 else f"LABEL_{len(static_texts)}"
            # Ensure uniqueness by appending a counter when the key already exists.
            if key in static_texts:
                i = 2
                while f"{key}_{i}" in static_texts:
                    i += 1
                key = f"{key}_{i}"
            entry = {
                "TEXT": text_val,
                "X": item.get("x", 0),
                "Y": item.get("y", 0),
                "FONT": get_font_path(item),
                "FONT_SIZE": scale_font_size(item.get("font_size", 12)),
                "FONT_COLOR": item.get("font_color", accent),
            }
            if item.get("align"):
                entry["ALIGN"] = item["align"]
            static_texts[key] = entry
            continue

        # Data sensor items
        if not data_name:
            continue

        mapping = DATANAME_MAP.get(data_name)

        if mapping is None:
            # No Go equivalent -> create static_text
            dn = item.get("display_name", data_name)
            parts = dn.split("--", 1)
            label = parts[1] if len(parts) > 1 else data_name
            label = translate_label(label)
            key = f"{data_name}_LABEL"
            entry = {
                "TEXT": label,
                "X": item.get("x", 0),
                "Y": item.get("y", 0),
                "FONT": get_font_path(item),
                "FONT_SIZE": scale_font_size(item.get("font_size", 12)),
                "FONT_COLOR": item.get("font_color", accent),
            }
            if item.get("align"):
                entry["ALIGN"] = item["align"]
            static_texts[key] = entry
            continue

        cat, sub, widget = mapping
        # Store the item data for STATS building
        # ArchBar -> RADIAL widget, StatuBar -> STATUS_BAR widget
        type_name = item.get("type_name", "")
        if type_name == "ArchBar":
            widget = "RADIAL"
        elif type_name == "StatuBar":
            widget = "STATUS_BAR"

        key = (cat, sub, widget)
        # Don't overwrite if we already have one (first occurrence wins)
        if key not in stats_items:
            stats_items[key] = (item, item_index)

    # --- Build STATS section ---
    stats = build_stats(stats_items, width, height, font_paths)

    if static_texts:
        theme["static_texts"] = static_texts
    theme["STATS"] = stats

    # --- Write output ---
    output_path = output_dir / "theme.yaml"
    with open(output_path, 'w', encoding='utf-8') as f:
        f.write("# Auto-converted from .turtheme\n")
        f.write(f"# Source: {out_name}\n")
        f.write(f"# Resolution: {width}x{height} ({orientation})\n")
        f.write("---\n")
        yaml.dump(theme, f, default_flow_style=False, allow_unicode=True, sort_keys=False)

    n_sensors = len(stats_items)
    n_static = len(static_texts)
    video_info = f", video: {video_name}" if video_copied else ""
    video_warn = f" [WARN: video '{video_name}' not found]" if video_name and not video_copied else ""
    fonts_info = f", {len(fonts_copied)} fonts" if fonts_copied else ""
    print(f"  OK: {theme_name} -> {out_name}/ ({n_sensors} sensors, {n_static} static{fonts_info}{video_info}{video_warn})")
    return True


def build_stats(stats_items: dict, width: int, height: int, font_paths: dict) -> dict:
    """Build the STATS section using real extracted positions.

    Only includes sections that have active (SHOW: True) sensors.
    The Go app's default.yaml fills in missing sections automatically.
    """
    stats = {}

    def _place_entry(stats, cat, sub, widget, entry):
        """Place an entry in the correct position in the stats dict."""
        if cat == "CPU":
            if "CPU" not in stats:
                stats["CPU"] = {}
            if sub not in stats["CPU"]:
                stats["CPU"][sub] = {"INTERVAL": 1}
            stats["CPU"][sub][widget] = entry
        elif cat == "GPU":
            if "GPU" not in stats:
                stats["GPU"] = {"INTERVAL": 1}
            if sub not in stats["GPU"]:
                stats["GPU"][sub] = {}
            stats["GPU"][sub][widget] = entry
        elif cat == "MEMORY":
            if "MEMORY" not in stats:
                stats["MEMORY"] = {"INTERVAL": 5}
            # sub may be a dot-path like "RAM.SIZE" meaning nested keys.
            d = stats["MEMORY"]
            for part in sub.split("."):
                if part not in d:
                    d[part] = {}
                d = d[part]
            d[widget] = entry
        elif cat == "DISK":
            if "DISK" not in stats:
                stats["DISK"] = {"INTERVAL": 10}
            if sub == "USED":
                if "USED" not in stats["DISK"]:
                    stats["DISK"]["USED"] = {}
                stats["DISK"]["USED"][widget] = entry
            elif sub == "TEMPERATURE":
                if "TEMPERATURE" not in stats["DISK"]:
                    stats["DISK"]["TEMPERATURE"] = {}
                stats["DISK"]["TEMPERATURE"][widget] = entry
        elif cat == "NET":
            if "NET" not in stats:
                stats["NET"] = {"INTERVAL": 1}
            # sub format: "ETH.UPLOAD", "ETH.DOWNLOAD", "WLO.UPLOAD", etc.
            parts = sub.split(".", 1)
            iface, direction = parts[0], (parts[1] if len(parts) > 1 else None)
            if iface not in stats["NET"]:
                stats["NET"][iface] = {}
            if direction:
                if direction not in stats["NET"][iface]:
                    stats["NET"][iface][direction] = {}
                stats["NET"][iface][direction][widget] = entry
            else:
                stats["NET"][iface][widget] = entry
        elif cat == "DATE":
            if "DATE" not in stats:
                stats["DATE"] = {"INTERVAL": 1}
            if sub == "HOUR":
                if widget in ("TEXT", "PERCENT_TEXT"):
                    entry["FORMAT"] = "short"
                if "HOUR" not in stats["DATE"]:
                    stats["DATE"]["HOUR"] = {}
                stats["DATE"]["HOUR"][widget] = entry
            elif sub == "DAY":
                if widget in ("TEXT", "PERCENT_TEXT"):
                    entry["FORMAT"] = "medium"
                if "DAY" not in stats["DATE"]:
                    stats["DATE"]["DAY"] = {}
                stats["DATE"]["DAY"][widget] = entry
        elif cat == "WEATHER":
            if "WEATHER" not in stats:
                stats["WEATHER"] = {}
            if sub not in stats["WEATHER"]:
                stats["WEATHER"][sub] = {}
            stats["WEATHER"][sub][widget] = entry
        elif cat == "HOST":
            if "HOST" not in stats:
                stats["HOST"] = {}
            if sub == "HOSTNAME":
                if "HOSTNAME" not in stats["HOST"]:
                    stats["HOST"]["HOSTNAME"] = {}
                stats["HOST"]["HOSTNAME"][widget] = entry
            elif sub and sub.startswith("LOAD."):
                load_key = sub.split(".", 1)[1]  # ONE, FIVE, FIFTEEN
                if "LOAD" not in stats["HOST"]:
                    stats["HOST"]["LOAD"] = {}
                if load_key not in stats["HOST"]["LOAD"]:
                    stats["HOST"]["LOAD"][load_key] = {}
                stats["HOST"]["LOAD"][load_key][widget] = entry
        elif cat == "VOLUME":
            if "VOLUME" not in stats:
                stats["VOLUME"] = {}
            stats["VOLUME"][widget] = entry

    _DIRECTION_MAP = {0: "left", 1: "right", 2: "up", 3: "down"}

    def _apply_graph_extras(entry, item):
        """Append optional new fields (gradient, corner, border, revert, blocks, direction) to a GRAPH or STATUS_BAR entry."""
        if item.get("back_color"):
            entry["EMPTY_COLOR"] = item["back_color"]
        if item.get("gradient_color"):
            entry["GRADIENT_COLOR"] = item["gradient_color"]
        # radius from StatuBar = corner radius; fall back to explicit corner_radius
        cr = item.get("corner_radius") or item.get("radius")
        if cr and int(cr) > 0:
            entry["CORNER_RADIUS"] = int(cr)
        if item.get("border_width"):
            entry["BORDER_WIDTH"] = int(item["border_width"])
        if item.get("revert_value"):
            entry["REVERT_VALUE"] = True
        if item.get("block_width"):
            entry["BLOCK_WIDTH"] = int(item["block_width"])
        dir_val = item.get("direction")
        if dir_val is not None:
            dir_str = _DIRECTION_MAP.get(int(dir_val), "left")
            if dir_str != "left":  # only write if non-default
                entry["DIRECTION"] = dir_str

    # Apply real data from extracted items (only SHOW: True entries)
    for (cat, sub, widget), (item, item_index) in stats_items.items():
        font_name = item.get("font", "")
        resolved_font = font_paths.get(font_name, DEFAULT_FONT)
        font_size_px = scale_font_size(item.get("font_size", 14))

        if widget == "RADIAL":
            # ArchBar -> RADIAL entry
            # turtheme posX/posY = top-left corner, our Go uses center
            diameter = item.get("diameter", 100)
            radius = diameter // 2
            angle_start = item.get("startPer", 0) * 360 // 100 if item.get("startPer") else 120
            block_angle = item.get("block_angle", 0)
            radial_entry = {
                "SHOW": True,
                "INDEX": item_index,
                "X": item.get("x", 0) + radius,
                "Y": item.get("y", 0) + radius,
                "RADIUS": radius,
                "WIDTH": item.get("archWidth", 10),
                "MIN_VALUE": 0,
                "MAX_VALUE": 100,
                "ANGLE_START": angle_start,
                "ANGLE_END": 60,
                "ANGLE_STEPS": 0 if block_angle > 0 else 1,
                "ANGLE_SEP": 4 if block_angle > 0 else 0,
                "CLOCKWISE": True,
                "BAR_COLOR": item.get("bar_color", "255, 255, 255"),
                "SHOW_TEXT": False,
                "SHOW_UNIT": False,
            }
            if block_angle > 0:
                radial_entry["BLOCK_ANGLE"] = block_angle
            if item.get("back_color"):
                radial_entry["EMPTY_COLOR"] = item["back_color"]
            if item.get("gradient_color"):
                radial_entry["GRADIENT_COLOR"] = item["gradient_color"]
            if item.get("round"):
                radial_entry["ROUND"] = True
            if item.get("revert"):
                radial_entry["REVERT"] = True
            if item.get("revert_value"):
                radial_entry["REVERT_VALUE"] = True
            _place_entry(stats, cat, sub, "RADIAL", radial_entry)
            continue

        if widget == "GRAPH":
            # StatusBar -> GRAPH entry (simple progress bar)
            graph_entry = {
                "SHOW": True, "INDEX": item_index,
                "X": item.get("x", 0),
                "Y": item.get("y", 0),
                "WIDTH": item.get("width", 100),
                "HEIGHT": item.get("height", 15),
                "MIN_VALUE": 0,
                "MAX_VALUE": 100,
                "BAR_COLOR": item.get("bar_color", "255, 255, 255"),
                "BAR_OUTLINE": False,
            }
            _apply_graph_extras(graph_entry, item)
            _place_entry(stats, cat, sub, "GRAPH", graph_entry)
            continue

        if widget == "STATUS_BAR":
            # StatuBar from .turtheme
            # Thick bars (height >= 10) -> GRAPH (simple progress bar)
            # Thin bars (height < 10) -> STATUS_BAR (slider with indicator)
            bar_height = item.get("height", 15)
            if bar_height >= 10:
                # Thick bar (or memory sensor) -> simple GRAPH
                graph_entry = {
                    "SHOW": True, "INDEX": item_index,
                    "X": item.get("x", 0),
                    "Y": item.get("y", 0),
                    "WIDTH": item.get("width", 100),
                    "HEIGHT": max(bar_height, 10),
                    "MIN_VALUE": 0,
                    "MAX_VALUE": 100,
                    "BAR_COLOR": item.get("bar_color", "255, 255, 255"),
                    "BAR_OUTLINE": False,
                }
                _apply_graph_extras(graph_entry, item)
                _place_entry(stats, cat, sub, "GRAPH", graph_entry)
            else:
                # Thin bar -> STATUS_BAR with indicator
                status_entry = {
                    "SHOW": True, "INDEX": item_index,
                    "X": item.get("x", 0),
                    "Y": item.get("y", 0),
                    "WIDTH": item.get("width", 100),
                    "HEIGHT": bar_height,
                    "MIN_VALUE": 0,
                    "MAX_VALUE": 100,
                    "BAR_COLOR": item.get("bar_color", "255, 255, 255"),
                    "INDICATOR_COLOR": item.get("bar_color", "255, 255, 255"),
                    "INDICATOR_RADIUS": max(bar_height + 2, 5),
                }
                _apply_graph_extras(status_entry, item)
                _place_entry(stats, cat, sub, "STATUS_BAR", status_entry)
            continue

        # TEXT entry (default)
        text_entry = {
            "SHOW": True, "INDEX": item_index,
            "X": item.get("x", 0),
            "Y": item.get("y", 0),
            "FONT": resolved_font,
            "FONT_SIZE": font_size_px,
            "FONT_COLOR": item.get("font_color", "255, 255, 255"),
        }
        if item.get("show_unit") is True:
            text_entry["SHOW_UNIT"] = True
        if item.get("placeholder"):
            text_entry["PLACEHOLDER"] = item["placeholder"]
        if item.get("align"):
            text_entry["ALIGN"] = item["align"]
        _place_entry(stats, cat, sub, widget, text_entry)

    return stats


def list_themes():
    """List all available extracted themes."""
    if not EXTRACTED_DIR.exists():
        print(f"Directory not found: {EXTRACTED_DIR}")
        return

    themes = sorted([
        d.name for d in EXTRACTED_DIR.iterdir()
        if d.is_dir() and (d / "theme.yaml").exists()
    ])
    print(f"Available themes ({len(themes)}):")
    for t in themes:
        yaml_path = EXTRACTED_DIR / t / "theme.yaml"
        with open(yaml_path, 'r', encoding='utf-8') as f:
            data = yaml.safe_load(f)
        if not data:
            print(f"  {t} (empty)")
            continue
        items = data.get("graph_items", [])
        n_sensors = len([i for i in items if i.get("data_name") and i["data_name"] != "StaticText" and i.get("type_name") != "Text"])
        n_static = len([i for i in items if i.get("type_name") == "Text" or i.get("data_name") == "StaticText"])
        w = data.get("width", "?")
        h = data.get("height", "?")
        video = data.get("video_name", "")
        flags = []
        if n_sensors:
            flags.append(f"{n_sensors} sensors")
        if n_static:
            flags.append(f"{n_static} static")
        if video:
            flags.append(f"video: {video}")
        print(f"  {t} ({w}x{h}) {' | '.join(flags) if flags else 'empty'}")


def main():
    parser = argparse.ArgumentParser(description="Convert extracted .turtheme to Go theme.yaml")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--all", action="store_true", help="Convert all themes")
    group.add_argument("--theme", type=str, help="Convert a specific theme by name")
    group.add_argument("--list", action="store_true", help="List available themes")
    args = parser.parse_args()

    if args.list:
        list_themes()
        return

    if not EXTRACTED_DIR.exists():
        print(f"Error: extracted directory not found: {EXTRACTED_DIR}")
        print("Run extract_turtheme.py first to extract .turtheme files.")
        sys.exit(1)

    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

    if args.all:
        themes = sorted([
            d.name for d in EXTRACTED_DIR.iterdir()
            if d.is_dir() and (d / "theme.yaml").exists()
        ])
        print(f"Converting {len(themes)} themes...\n")
        ok = 0
        for t in themes:
            if convert_theme(t):
                ok += 1
        print(f"\nDone: {ok}/{len(themes)} themes converted to {OUTPUT_DIR}/")
    elif args.theme:
        if convert_theme(args.theme):
            print(f"\nTheme converted successfully.")
        else:
            print(f"\nFailed to convert theme.")
            sys.exit(1)


if __name__ == "__main__":
    main()
