#!/usr/bin/env python3
"""
Upload a video file to Turing Smart Screen 5" (Rev C) device storage.

Sequence (validated from BlueTheme-Flip180.pcapng, gundam.pcapng, teste.pcap):
  HELLO -> init
  STOP_VIDEO
  STOP_MEDIA -> "media_stop"
  SET_BRIGHTNESS
  GET_STORAGE_STATUS -> check space
  GET_FILE_INFO -> check if exists
  STOP_VIDEO
  STOP_MEDIA -> "media_stop"
  LIST_DIR -> list current files
  CREATE_FILE(path, size) -> "create_success"
  [raw file bytes — single write, no framing, no chunking]
  <- "file_rev_doneimg_show_"
  GET_FILE_INFO -> verify uploaded size

Usage: python3 upload_video.py <local_file.mp4> [device_filename]
"""

import sys
import os
sys.path.insert(0, sys.path[0])
from turing_common import TuringDevice

MAX_FILE_SIZE = 10 * 1024 * 1024  # 10 MB


def main():
    if len(sys.argv) < 2:
        print("Usage: python3 upload_video.py <local_file.mp4> [device_filename]")
        print("  device_filename defaults to the local filename")
        sys.exit(1)

    local_path = sys.argv[1]
    if not os.path.exists(local_path):
        print(f"ERROR: File not found: {local_path}")
        sys.exit(1)

    file_size = os.path.getsize(local_path)
    if file_size > MAX_FILE_SIZE:
        print(f"ERROR: File too large ({file_size} bytes). Max is {MAX_FILE_SIZE // 1024 // 1024} MB.")
        sys.exit(1)

    # Device filename (can be overridden)
    if len(sys.argv) > 2:
        device_filename = sys.argv[2]
    else:
        device_filename = os.path.basename(local_path)
    device_path = f"/root/video/{device_filename}"

    print(f"=== Upload Video ===")
    print(f"  Local:  {local_path} ({file_size} bytes, {file_size/1024:.1f} KB)")
    print(f"  Device: {device_path}")
    print()

    dev = TuringDevice()
    try:
        dev.init()

        # GET_STORAGE_STATUS — check space
        info = dev.get_storage_status()
        if info:
            total, used, free = info
            print(f"    Storage: {total} KB total, {used} KB used, {free} KB free")
            needed_kb = file_size // 1024 + 1
            if needed_kb > free:
                print(f"    ERROR: Not enough space! Need {needed_kb} KB, have {free} KB free.")
                sys.exit(1)

        # GET_FILE_INFO — check if already exists
        existing_size = dev.get_file_info(device_path)
        if existing_size > 0:
            print(f"    File already exists ({existing_size} bytes), will delete first")
            dev.delete_file(device_path)
            # Verify space after delete
            dev.get_storage_status()

        # Second STOP_VIDEO + STOP_MEDIA (as seen in gundam.pcapng)
        print()
        dev.send("STOP_VIDEO", b'\x79\xef\x69\x00\x00\x00\x01')
        dev.send("STOP_MEDIA", b'\x96\xef\x69\x00\x00\x00\x01', 1024)

        # LIST_DIR (as seen in PCAP — software lists before upload)
        files = dev.list_dir("/root/video/")
        if files:
            print(f"    Current files: {', '.join(files)}")

        # UPLOAD
        print()
        success = dev.upload_file(local_path, device_path)
        if not success:
            print("\n=== UPLOAD FAILED ===")
            sys.exit(1)

        # Verify
        print()
        uploaded_size = dev.get_file_info(device_path)
        print(f"\n=== UPLOAD {'SUCCESS' if uploaded_size == file_size else 'VERIFY FAILED'} ===")
        print(f"  Device reports: {uploaded_size} bytes")
        print(f"  Expected:       {file_size} bytes")

    finally:
        dev.close()


if __name__ == "__main__":
    main()
