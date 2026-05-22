#!/usr/bin/env python3
"""
Delete a video from Turing Smart Screen 5" (Rev C) device storage.

Usage: python3 delete_video.py <filename>
  Example: python3 delete_video.py earth.mp4
"""

import sys
sys.path.insert(0, sys.path[0])
from turing_common import TuringDevice


def main():
    if len(sys.argv) < 2:
        print("Usage: python3 delete_video.py <filename>")
        print("  Example: python3 delete_video.py earth.mp4")
        sys.exit(1)

    filename = sys.argv[1]
    if not filename.startswith("/"):
        device_path = f"/root/video/{filename}"
    else:
        device_path = filename

    print(f"=== Delete: {device_path} ===\n")

    dev = TuringDevice()
    try:
        dev.init()

        # Check if exists
        size = dev.get_file_info(device_path)
        if size == 0:
            print(f"\n  File not found on device: {device_path}")
            sys.exit(1)
        print(f"    File size: {size} bytes")

        # Delete
        print()
        dev.delete_file(device_path)

        # Verify
        size = dev.get_file_info(device_path)
        if size == 0:
            print(f"\n=== DELETED ===")
        else:
            print(f"\n=== DELETE FAILED (still reports {size} bytes) ===")

    finally:
        dev.close()


if __name__ == "__main__":
    main()
