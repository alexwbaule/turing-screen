#!/usr/bin/env python3
"""
Test/diagnostic script for Turing Smart Screen 5" (Rev C).
Runs the init sequence and checks device state.

Usage: python3 test_video.py [/root/video/earth.mp4]
"""

import sys
sys.path.insert(0, sys.path[0])
from turing_common import TuringDevice


def main():
    video_path = "/root/video/earth.mp4"
    if len(sys.argv) > 1:
        video_path = sys.argv[1]

    print(f"=== Turing Screen Diagnostic ===\n")

    dev = TuringDevice()
    try:
        resp = dev.init()
        print(f"    Device: {resp}")

        # Storage
        print("call get storage..")
        info = dev.get_storage_status()
        if info:
            total, used, free = info
            print(f"    Storage: {total} KB total, {used} KB used, {free} KB free")

        # List files
        print()
        files = dev.list_dir("/root/video/")
        if files:
            print(f"    Files ({len(files)}): {', '.join(files)}")
        else:
            print("    No files on device")

        # Check specific file
        print()
        size = dev.get_file_info(video_path)
        if size > 0:
            print(f"    ✓ {video_path} exists ({size} bytes, {size/1024:.1f} KB)")
        else:
            print(f"    ✗ {video_path} NOT found")

        print("\n=== Done ===")
    finally:
        dev.close()


if __name__ == "__main__":
    main()
