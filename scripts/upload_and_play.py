#!/usr/bin/env python3
"""
Upload a video and immediately play it on Turing Smart Screen 5" (Rev C).

Full sequence (validated from gundam.pcapng, BlueTheme-Flip180.pcapng):
  --- INIT ---
  HELLO
  STOP_VIDEO
  STOP_MEDIA
  SET_BRIGHTNESS
  --- CHECK ---
  GET_STORAGE_STATUS
  GET_FILE_INFO (check if exists)
  GET_STORAGE_STATUS
  --- CLEANUP ---
  STOP_VIDEO
  STOP_MEDIA
  LIST_DIR
  --- UPLOAD ---
  CREATE_FILE -> "create_success"
  [raw file data]
  <- "file_rev_doneimg_show_"
  GET_FILE_INFO (verify)
  --- PLAY ---
  RESTART_DEVICE
  (wait 2s)
  HELLO
  GET_FILE_INFO
  PLAY_VIDEO -> "play_video_success"

Usage: python3 upload_and_play.py <local_file.mp4> [device_filename]
"""

import sys
import os
sys.path.insert(0, sys.path[0])
from turing_common import TuringDevice

MAX_FILE_SIZE = 10 * 1024 * 1024  # 10 MB


def main():
    if len(sys.argv) < 2:
        print("Usage: python3 upload_and_play.py <local_file.mp4> [device_filename]")
        sys.exit(1)

    local_path = sys.argv[1]
    if not os.path.exists(local_path):
        print(f"ERROR: File not found: {local_path}")
        sys.exit(1)

    file_size = os.path.getsize(local_path)
    if file_size > MAX_FILE_SIZE:
        print(f"ERROR: File too large ({file_size} bytes). Max is {MAX_FILE_SIZE // 1024 // 1024} MB.")
        sys.exit(1)

    if len(sys.argv) > 2:
        device_filename = sys.argv[2]
    else:
        device_filename = os.path.basename(local_path)
    device_path = f"/root/video/{device_filename}"

    print(f"=== Upload & Play ===")
    print(f"  Local:  {local_path} ({file_size} bytes, {file_size/1024:.1f} KB)")
    print(f"  Device: {device_path}")
    print()

    dev = TuringDevice()
    try:
        # --- INIT ---
        dev.init()

        # --- CHECK ---
        info = dev.get_storage_status()
        if info:
            total, used, free = info
            print(f"    Storage: {total} KB total, {used} KB used, {free} KB free")
            needed_kb = file_size // 1024 + 1
            if needed_kb > free:
                print(f"    ERROR: Not enough space! Need {needed_kb} KB, have {free} KB free.")
                sys.exit(1)

        existing_size = dev.get_file_info(device_path)
        if existing_size > 0:
            print(f"    File exists ({existing_size} bytes), will overwrite")
            dev.delete_file(device_path)

        dev.get_storage_status()

        # --- CLEANUP ---
        print()
        dev.send("STOP_VIDEO", b'\x79\xef\x69\x00\x00\x00\x01')
        dev.send("STOP_MEDIA", b'\x96\xef\x69\x00\x00\x00\x01', 1024)
        files = dev.list_dir("/root/video/")
        if files:
            print(f"    Current files: {', '.join(files)}")

        # --- UPLOAD ---
        print()
        success = dev.upload_file(local_path, device_path)
        if not success:
            print("\n=== UPLOAD FAILED ===")
            sys.exit(1)

        # Verify upload
        uploaded_size = dev.get_file_info(device_path)
        if uploaded_size != file_size:
            print(f"  WARNING: Size mismatch! Device={uploaded_size}, Expected={file_size}")

        # --- PLAY ---
        print()
        dev.restart_device(wait=2.0)

        size = dev.get_file_info(device_path)
        print(f"    File size after restart: {size} bytes")

        resp = dev.play_video(device_path, loop=True)

        if resp == "play_video_success":
            print(f"\n=== SUCCESS — Video playing! ===")
        else:
            print(f"\n=== PLAY FAILED: {resp} ===")

    finally:
        dev.close()


if __name__ == "__main__":
    main()
