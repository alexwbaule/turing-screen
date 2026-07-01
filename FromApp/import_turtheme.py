#!/usr/bin/env python3
"""
Full pipeline: .turtheme → res/themes/  (single script)

Extracts, converts, translates Chinese names/text, and copies themes in one step.

Usage:
    python3 import_turtheme.py <file.turtheme> [<file2.turtheme> ...]
    python3 import_turtheme.py <directory/>         # all .turtheme in that dir
    python3 import_turtheme.py --all                # all .turtheme in FromApp/

Options:
    --dest <path>    Destination themes directory (prompted if omitted)
    --dry-run        Show what would be done without writing anything
"""

import argparse
import shutil
import sys
import tempfile
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPT_DIR.parent
DEFAULT_DEST = PROJECT_ROOT / "res" / "themes"

# Default asset source directories (relative to FromApp/)
DEFAULT_VIDEO_DIR = SCRIPT_DIR / "52inchthemeENG" / "video" / "7201280"
DEFAULT_FONTS_DIR = SCRIPT_DIR / "52inchENG" / "TURZX-V3.1.0-52inchENG" / "fonts"

# ---------------------------------------------------------------------------
# Import pipeline modules (same directory)
# ---------------------------------------------------------------------------
sys.path.insert(0, str(SCRIPT_DIR))
import extract_turtheme as ext_mod
import convert_turtheme as conv_mod


def run_pipeline(turtheme_file: Path, dest_dir: Path, dry_run: bool = False) -> bool:
    """
    Full pipeline for a single .turtheme file:
      1. Extract  → temp/extracted/<name>/
      2. Convert  → temp/converted/<EnglishName>/
      3. Copy     → dest_dir/<EnglishName>/
    """
    theme_name = turtheme_file.stem
    en_name = conv_mod.sanitize_name(theme_name)  # translates Chinese, slugifies
    dest_theme = dest_dir / en_name

    print(f"\n{'[DRY RUN] ' if dry_run else ''}Importing: {turtheme_file.name}")
    print(f"  → {dest_theme}")

    if dry_run:
        return True

    with tempfile.TemporaryDirectory(prefix="turtheme_") as tmp:
        tmp_path = Path(tmp)
        extracted_dir = tmp_path / "extracted"
        converted_dir = tmp_path / "converted"

        # --- Step 1: Extract .turtheme ---
        ext_out = extracted_dir / theme_name
        ext_out.mkdir(parents=True, exist_ok=True)
        try:
            ext_mod.extract_turtheme(str(turtheme_file), str(ext_out))
        except Exception as e:
            print(f"  [ERROR] Extraction failed: {e}")
            return False

        # --- Step 2: Convert to theme.yaml ---
        # Patch module globals so convert uses our temp dirs
        orig_extracted = conv_mod.EXTRACTED_DIR
        orig_output = conv_mod.OUTPUT_DIR
        orig_video = conv_mod.VIDEO_DIR
        orig_fonts = conv_mod.FONTS_DIR
        try:
            conv_mod.EXTRACTED_DIR = extracted_dir
            conv_mod.OUTPUT_DIR = converted_dir
            if DEFAULT_VIDEO_DIR.exists():
                conv_mod.VIDEO_DIR = DEFAULT_VIDEO_DIR
            if DEFAULT_FONTS_DIR.exists():
                conv_mod.FONTS_DIR = DEFAULT_FONTS_DIR
            converted_dir.mkdir(parents=True, exist_ok=True)
            ok = conv_mod.convert_theme(theme_name)
        finally:
            conv_mod.EXTRACTED_DIR = orig_extracted
            conv_mod.OUTPUT_DIR = orig_output
            conv_mod.VIDEO_DIR = orig_video
            conv_mod.FONTS_DIR = orig_fonts

        if not ok:
            print(f"  [ERROR] Conversion failed for {theme_name}")
            return False

        # --- Step 3: Copy to destination ---
        conv_theme_dir = converted_dir / en_name
        if not conv_theme_dir.exists():
            # Fallback: find any dir in converted_dir
            candidates = list(converted_dir.iterdir())
            if not candidates:
                print(f"  [ERROR] No output found in {converted_dir}")
                return False
            conv_theme_dir = candidates[0]

        if dest_theme.exists():
            shutil.rmtree(dest_theme)
        shutil.copytree(conv_theme_dir, dest_theme)
        print(f"  Copied to: {dest_theme}")

    return True


def collect_files(inputs: list[str]) -> list[Path]:
    """Expand file/directory inputs into a list of .turtheme files."""
    files = []
    for inp in inputs:
        p = Path(inp)
        if p.is_file() and p.suffix.lower() == ".turtheme":
            files.append(p)
        elif p.is_dir():
            found = sorted(p.glob("*.turtheme"))
            if not found:
                print(f"[WARN] No .turtheme files found in {p}")
            files.extend(found)
        else:
            print(f"[WARN] Skipping: {inp} (not a .turtheme file or directory)")
    return files


def ask_dest() -> Path:
    """Prompt user for destination directory."""
    default = str(DEFAULT_DEST)
    try:
        answer = input(f"Destination directory [{default}]: ").strip()
    except (EOFError, KeyboardInterrupt):
        print()
        sys.exit(0)
    return Path(answer) if answer else Path(default)


def main():
    parser = argparse.ArgumentParser(
        description="Import .turtheme files into res/themes/ (extract + convert + translate + copy)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        "input", nargs="*",
        help=".turtheme file(s) or directory containing them",
    )
    parser.add_argument(
        "--all", action="store_true",
        help="Process all .turtheme files in the FromApp/ directory",
    )
    parser.add_argument(
        "--dest", type=str, default=None,
        help="Destination themes directory (default: res/themes/)",
    )
    parser.add_argument(
        "--dry-run", action="store_true",
        help="Preview what would be imported without writing files",
    )
    args = parser.parse_args()

    # Collect input files
    if args.all:
        files = sorted(SCRIPT_DIR.rglob("*.turtheme"))
        if not files:
            print(f"No .turtheme files found in {SCRIPT_DIR}")
            sys.exit(1)
    elif args.input:
        files = collect_files(args.input)
    else:
        parser.print_help()
        sys.exit(0)

    if not files:
        print("No .turtheme files to process.")
        sys.exit(1)

    # Determine destination
    if args.dest:
        dest_dir = Path(args.dest)
    elif args.dry_run:
        dest_dir = DEFAULT_DEST
    else:
        dest_dir = ask_dest()

    if not args.dry_run:
        dest_dir.mkdir(parents=True, exist_ok=True)

    print(f"\nProcessing {len(files)} theme(s) → {dest_dir}\n{'─'*60}")

    ok = 0
    fail = 0
    for f in files:
        if run_pipeline(f, dest_dir, dry_run=args.dry_run):
            ok += 1
        else:
            fail += 1

    print(f"\n{'─'*60}")
    if args.dry_run:
        print(f"Dry run: {ok} theme(s) would be imported to {dest_dir}")
    else:
        print(f"Done: {ok} imported, {fail} failed → {dest_dir}")


if __name__ == "__main__":
    main()
