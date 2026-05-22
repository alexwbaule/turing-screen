#!/usr/bin/env python3
"""
Delete ALL videos from Turing Smart Screen 5" (Rev C) device storage.

Usage: python3 delete_all_videos.py
"""

import sys
sys.path.insert(0, sys.path[0])
from turing_common import TuringDevice


def main():
    print("=== Delete All Videos ===\n")

    dev = TuringDevice()
    try:
        dev.init()

        # List files
        files = dev.list_dir("/root/video/")
        if not files:
            print("  No files to delete.")
            return

        print(f"  Found {len(files)} files: {', '.join(files)}")
        print()

        # Delete each one
        for filename in files:
            filepath = f"/root/video/{filename}"
            dev.delete_file(filepath)

        # Verify
        print()
        dev.get_storage_status()
        remaining = dev.list_dir("/root/video/")
        if not remaining:
            print("\n=== ALL DELETED ===")
        else:
            print(f"\n=== Still remaining: {', '.join(remaining)} ===")

    finally:
        dev.close()


if __name__ == "__main__":
    main()
