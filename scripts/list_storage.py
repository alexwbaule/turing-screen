#!/usr/bin/env python3
"""
List files on Turing Smart Screen 5" (Rev C) device storage.

Usage: python3 list_storage.py
"""

import sys
sys.path.insert(0, sys.path[0])
from turing_common import TuringDevice


def main():
    print("=== Turing Screen Storage ===\n")

    dev = TuringDevice()
    try:
        dev.init()

        # Storage status
        info = dev.get_storage_status()
        if info:
            total, used, free = info
            print(f"\n  Storage: {total} KB total, {used} KB used, {free} KB free")
            print(f"           ({total/1024:.1f} MB / {used/1024:.1f} MB / {free/1024:.1f} MB)")

        # List files
        print()
        files = dev.list_dir("/root/video/")
        if files:
            print(f"  Files in /root/video/ ({len(files)}):\n")
            for filename in files:
                filepath = f"/root/video/{filename}"
                size = dev.get_file_info(filepath)
                if size > 0:
                    print(f"    {filename:30s} {size:>10} bytes ({size/1024:.1f} KB)")
                else:
                    print(f"    {filename:30s} (not found or empty)")
        else:
            print("  No files found.")

        print("\n=== Done ===")
    finally:
        dev.close()


if __name__ == "__main__":
    main()
