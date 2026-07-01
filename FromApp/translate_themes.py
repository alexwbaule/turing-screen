#!/usr/bin/env python3
"""
Translate Chinese theme names and text content to English.

- Renames theme directories that have Chinese names
- Translates Chinese text in theme.yaml files (keys, values, comments)
- Updates all internal references after renaming

Usage:
    python3 translate_themes.py --dry-run   # preview changes
    python3 translate_themes.py             # apply changes
"""

import argparse
import re
import sys
from pathlib import Path

try:
    from deep_translator import GoogleTranslator
    _translator = GoogleTranslator(source="zh-CN", target="en")
except ImportError:
    print("Error: deep-translator not installed. Run: pip3 install deep-translator --break-system-packages")
    sys.exit(1)

THEMES_DIR = Path(__file__).resolve().parent.parent / "res" / "themes"

# Manual overrides — well-known names that Google Translate gets wrong or awkward
MANUAL_OVERRIDES = {
    "原神": "Genshin_Impact",
    "微星": "MSI",
    "皮卡丘主题": "Pikachu_Theme",
    "ROG星空": "ROG_Starfield",
    "星空主题": "Starfield_Theme",
    "机能风主题": "Tech_Style_Theme",
    "横屏6宫格": "Landscape_6Grid",
    "横屏深空科幻": "Landscape_Deep_Space_SciFi",
    "横屏简洁黑白": "Landscape_Simple_BW",
    "任务管理器主题": "Task_Manager_Theme",
    "数据化界面主题": "Data_Interface_Theme",
    "科技数据化主题": "Tech_Data_Theme",
    "科技横屏主题": "Tech_Landscape_Theme",
    "科技横屏主题中文": "Tech_Landscape_Theme_CN",
    "简约未来紫色": "Minimalist_Future_Purple",
    "赛车主题": "Racing_Theme",
    # Common text fragments
    "晴转多云 20° 东北风": "Partly Cloudy 20° NE Wind",
    "硬盘使用率": "Disk Usage",
    "硬盘使用率_1": "Disk Usage",
    "磁盘0": "Disk0",
    "磁盘1": "Disk1",
    "网络": "Network",
    "内存": "Memory",
    "总内存(GB)": "Total Memory(GB)",
    "速度": "Speed",
    "Fan转速": "Fan Speed",
    "风扇转速": "Fan Speed",
    "水泵转速": "Pump Speed",
    "CPU利用率": "CPU Usage",
    "CPU温度": "CPU Temp",
    "显卡利用率": "GPU Usage",
    "显卡温度": "GPU Temp",
    "已用内存": "RAM Used",
    "内存利用率": "RAM Usage",
    "上传速度": "Upload Speed",
    "下载速度": "Download Speed",
    "日期": "Date",
    "时间": "Time",
}

_translate_cache: dict[str, str] = {}


def has_chinese(text: str) -> bool:
    return any("一" <= c <= "鿿" for c in text)


def translate_text(text: str) -> str:
    """Translate Chinese text to English, using cache and manual overrides."""
    if not has_chinese(text):
        return text
    if text in _translate_cache:
        return _translate_cache[text]
    if text in MANUAL_OVERRIDES:
        result = MANUAL_OVERRIDES[text]
        _translate_cache[text] = result
        return result
    try:
        result = _translator.translate(text)
        if not result:
            result = text
    except Exception as e:
        print(f"  [WARN] Translation failed for '{text}': {e}")
        result = text
    _translate_cache[text] = result
    return result


def slugify(text: str) -> str:
    """Convert translated text to a filesystem/YAML-safe slug."""
    text = re.sub(r"[^\w\s-]", "", text).strip()
    text = re.sub(r"[\s]+", "_", text)
    text = re.sub(r"_+", "_", text)
    return text


def dir_name_to_english(name: str) -> str:
    """Translate a Chinese directory name to an English slug."""
    if not has_chinese(name):
        return name
    translated = translate_text(name)
    return slugify(translated)


def translate_yaml_value(value: str) -> str:
    """Translate a YAML string value that may contain Chinese."""
    if not has_chinese(value):
        return value
    return translate_text(value)


def translate_yaml_key(key: str) -> str:
    """Translate a YAML key that may contain Chinese."""
    if not has_chinese(key):
        return key
    translated = translate_text(key)
    return slugify(translated)


def process_yaml_file(path: Path, dry_run: bool) -> bool:
    """
    Translate Chinese content in a theme.yaml file.
    Handles:
      - Comment lines (# Source: ...)
      - YAML keys containing Chinese
      - TEXT/PLACEHOLDER/# values containing Chinese
    Returns True if changes were made (or would be made).
    """
    content = path.read_text(encoding="utf-8")
    lines = content.splitlines(keepends=True)
    changed = False
    new_lines = []

    for line in lines:
        original = line

        # Comment lines
        if line.lstrip().startswith("#") and has_chinese(line):
            # Translate only the Chinese portion, keep the prefix
            m = re.match(r"(#\s*(?:Source:|Resolution:)?\s*)(.*)", line.rstrip())
            if m:
                prefix, rest = m.groups()
                if has_chinese(rest):
                    rest_translated = translate_text(rest)
                    line = prefix + rest_translated + "\n"
            else:
                line = translate_text(line.rstrip()) + "\n"

        # YAML key: value lines — translate key AND value separately
        elif ":" in line and not line.lstrip().startswith("#"):
            # Detect indent
            indent_m = re.match(r"^(\s*)", line)
            indent = indent_m.group(1) if indent_m else ""

            # Key might be bare (ends with :) or have a value
            key_m = re.match(r"^(\s*)([^:]+):(.*)", line)
            if key_m:
                pre, key, rest = key_m.groups()
                new_key = translate_yaml_key(key.strip())
                if key.strip() != new_key:
                    line = f"{pre}{new_key}:{rest}"

                # Now handle the value part
                if rest.strip():
                    # Inline value
                    # Match: " value" or " 'value'" or ' "value"'
                    val_m = re.match(r"(\s*)(.*)", rest)
                    if val_m:
                        vspc, val = val_m.groups()
                        # Strip quotes if present
                        quoted = val.startswith(("'", '"'))
                        quote_char = val[0] if quoted else ""
                        inner = val.strip("'\"")
                        if has_chinese(inner):
                            translated_val = translate_yaml_value(inner)
                            if quoted:
                                val = quote_char + translated_val + quote_char
                            else:
                                val = translated_val
                            # Reconstruct the line
                            key_part = line.split(":")[0]
                            line = f"{key_part}:{vspc}{val}\n"

        if line != original:
            changed = True
            if dry_run:
                print(f"    - {original.rstrip()}")
                print(f"    + {line.rstrip()}")

        new_lines.append(line)

    if changed and not dry_run:
        path.write_text("".join(new_lines), encoding="utf-8")

    return changed


def rename_dir(old_path: Path, new_name: str, dry_run: bool) -> Path:
    """Rename a theme directory and return the new path."""
    new_path = old_path.parent / new_name
    if old_path == new_path:
        return old_path
    if dry_run:
        print(f"  RENAME dir: {old_path.name} → {new_name}")
        return new_path
    if new_path.exists():
        print(f"  [SKIP] Target already exists: {new_path}")
        return old_path
    old_path.rename(new_path)
    print(f"  RENAMED: {old_path.name} → {new_name}")
    return new_path


def main():
    parser = argparse.ArgumentParser(description="Translate Chinese theme names and text to English")
    parser.add_argument("--dry-run", action="store_true", help="Preview changes without applying")
    args = parser.parse_args()

    print(f"Running in themes directory: {THEMES_DIR}")
    if not THEMES_DIR.exists():
        print(f"Error: themes directory not found: {THEMES_DIR}")
        sys.exit(1)

    dry_run = args.dry_run
    if dry_run:
        print("DRY RUN — no changes will be written\n")

    theme_dirs = sorted([d for d in THEMES_DIR.iterdir() if d.is_dir()])
    renamed = 0
    yaml_changed = 0

    for theme_dir in theme_dirs:
        dir_name = theme_dir.name
        yaml_src_dir = theme_dir  # path where YAML actually lives now
        if has_chinese(dir_name):
            new_name = dir_name_to_english(dir_name)
            theme_dir = rename_dir(theme_dir, new_name, dry_run)
            renamed += 1
            if dry_run:
                # In dry run the rename didn't happen, YAML is at original path
                theme_dir = yaml_src_dir

        yaml_path = theme_dir / "theme.yaml"
        if yaml_path.exists():
            if has_chinese(yaml_path.read_text(encoding="utf-8")):
                if dry_run:
                    print(f"  YAML: {theme_dir.name}/theme.yaml")
                changed = process_yaml_file(yaml_path, dry_run)
                if changed:
                    yaml_changed += 1

    print(f"\nSummary: {renamed} directories renamed, {yaml_changed} YAML files updated")
    if dry_run:
        print("(dry run — nothing was written)")


if __name__ == "__main__":
    main()
